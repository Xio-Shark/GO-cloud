package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterJobRunnerConfig struct {
	Namespace          string
	PollInterval       time.Duration
	TTLSeconds         int
	ImagePullPolicy    string
	ServiceAccountName string
}

type ClusterJobRunner struct {
	client kubernetes.Interface
	config ClusterJobRunnerConfig
}

func NewClusterJobRunner(config ClusterJobRunnerConfig) (JobRunner, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			return nil, err
		}
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	if config.Namespace == "" {
		config.Namespace = "go-cloud"
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.ImagePullPolicy == "" {
		config.ImagePullPolicy = string(corev1.PullIfNotPresent)
	}
	return &ClusterJobRunner{
		client: clientset,
		config: config,
	}, nil
}

func (r *ClusterJobRunner) RunJob(ctx context.Context, request JobRequest) (JobResult, error) {
	namespace := request.Namespace
	if namespace == "" {
		namespace = r.config.Namespace
	}
	activeDeadline := int64(request.TimeoutSeconds)
	ttlSeconds := int32(r.config.TTLSeconds)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: sanitizeJobName(fmt.Sprintf("task-%d-", request.TaskID)),
			Namespace:    namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "go-cloud-task",
				"app.kubernetes.io/component": request.Namespace,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            int32Ptr(0),
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadline,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "task",
							Image:           request.Image,
							ImagePullPolicy: corev1.PullPolicy(r.config.ImagePullPolicy),
							Command:         request.Command,
							Env:             buildEnvVars(request.Env),
						},
					},
				},
			},
		},
	}
	if r.config.ServiceAccountName != "" {
		job.Spec.Template.Spec.ServiceAccountName = r.config.ServiceAccountName
	}

	created, err := r.client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return JobResult{}, err
	}

	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = r.client.BatchV1().Jobs(namespace).Delete(context.Background(), created.Name, metav1.DeleteOptions{})
			return JobResult{}, ctx.Err()
		case <-ticker.C:
			current, getErr := r.client.BatchV1().Jobs(namespace).Get(ctx, created.Name, metav1.GetOptions{})
			if getErr != nil {
				return JobResult{}, getErr
			}
			logs, exitCode, logErr := r.readJobLogsAndExitCode(ctx, namespace, current.Name)
			if logErr != nil && !errors.Is(logErr, errJobPodNotReady) {
				return JobResult{}, logErr
			}
			if current.Status.Succeeded > 0 {
				return JobResult{ExitCode: exitCode, Logs: logs}, nil
			}
			if current.Status.Failed > 0 {
				if exitCode == 0 {
					exitCode = 1
				}
				return JobResult{ExitCode: exitCode, Logs: logs}, nil
			}
		}
	}
}

var errJobPodNotReady = errors.New("job pod not ready")

func (r *ClusterJobRunner) readJobLogsAndExitCode(ctx context.Context, namespace string, jobName string) (string, int, error) {
	pods, err := r.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return "", 0, err
	}
	if len(pods.Items) == 0 {
		return "", 0, errJobPodNotReady
	}
	pod := pods.Items[0]

	logStream, err := r.client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(ctx)
	if err != nil {
		return "", extractExitCode(pod), errJobPodNotReady
	}
	defer logStream.Close()

	content, err := io.ReadAll(logStream)
	if err != nil {
		return "", extractExitCode(pod), err
	}
	return string(content), extractExitCode(pod), nil
}

func extractExitCode(pod corev1.Pod) int {
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated != nil {
			return int(status.State.Terminated.ExitCode)
		}
	}
	return 0
}

func buildEnvVars(values map[string]string) []corev1.EnvVar {
	items := make([]corev1.EnvVar, 0, len(values))
	for key, value := range values {
		items = append(items, corev1.EnvVar{Name: key, Value: value})
	}
	return items
}

func sanitizeJobName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

func int32Ptr(v int32) *int32 {
	return &v
}
