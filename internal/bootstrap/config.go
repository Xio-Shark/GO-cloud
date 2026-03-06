package bootstrap

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName                 string
	HTTPAddr                    string
	AdminAddr                   string
	MySQLDSN                    string
	RedisAddr                   string
	RedisPassword               string
	RedisDB                     int
	LogLevel                    string
	SchedulerScanInterval       time.Duration
	QueuePopTimeout             time.Duration
	GracefulShutdownTimeout     time.Duration
	NotifierRequestTimeout      time.Duration
	ExecutorHTTPRequestTimout   time.Duration
	GitOpsOverlaysRoot          string
	KubernetesNamespace         string
	ContainerJobPollInterval    time.Duration
	ContainerJobTTLSeconds      int
	ContainerJobImagePullPolicy string
	ContainerJobServiceAccount  string
}

func LoadConfig(serviceName string) Config {
	return Config{
		ServiceName:                 serviceName,
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		AdminAddr:                   getEnv("ADMIN_ADDR", ":9090"),
		MySQLDSN:                    os.Getenv("MYSQL_DSN"),
		RedisAddr:                   getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:               os.Getenv("REDIS_PASSWORD"),
		RedisDB:                     getEnvAsInt("REDIS_DB", 0),
		LogLevel:                    getEnv("LOG_LEVEL", "info"),
		SchedulerScanInterval:       getEnvAsDuration("SCHEDULER_SCAN_INTERVAL", 5*time.Second),
		QueuePopTimeout:             getEnvAsDuration("QUEUE_POP_TIMEOUT", 5*time.Second),
		GracefulShutdownTimeout:     getEnvAsDuration("GRACEFUL_SHUTDOWN_TIMEOUT", 10*time.Second),
		NotifierRequestTimeout:      getEnvAsDuration("NOTIFIER_REQUEST_TIMEOUT", 10*time.Second),
		ExecutorHTTPRequestTimout:   getEnvAsDuration("EXECUTOR_HTTP_TIMEOUT", 15*time.Second),
		GitOpsOverlaysRoot:          getEnv("GITOPS_OVERLAYS_ROOT", "deployments/k8s/overlays"),
		KubernetesNamespace:         getEnv("KUBERNETES_NAMESPACE", "go-cloud"),
		ContainerJobPollInterval:    getEnvAsDuration("CONTAINER_JOB_POLL_INTERVAL", 2*time.Second),
		ContainerJobTTLSeconds:      getEnvAsInt("CONTAINER_JOB_TTL_SECONDS", 300),
		ContainerJobImagePullPolicy: getEnv("CONTAINER_JOB_IMAGE_PULL_POLICY", "IfNotPresent"),
		ContainerJobServiceAccount:  os.Getenv("CONTAINER_JOB_SERVICE_ACCOUNT"),
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
