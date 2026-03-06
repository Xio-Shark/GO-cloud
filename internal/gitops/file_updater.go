package gitops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const gitOpsNamespace = "go-cloud"

var defaultImageVersions = map[string]string{
	"api-server": "0.1.0",
	"scheduler":  "0.1.0",
	"worker":     "0.1.0",
	"notifier":   "0.1.0",
}

var imageOrder = []string{"api-server", "scheduler", "worker", "notifier"}

type FileUpdater struct {
	overlaysRoot string
}

type deploymentPatch struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Name  string `yaml:"name"`
					Image string `yaml:"image"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

func NewFileUpdater(overlaysRoot string) *FileUpdater {
	return &FileUpdater{overlaysRoot: overlaysRoot}
}

func (u *FileUpdater) UpdateImage(ctx context.Context, request UpdateRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if request.Environment == "" {
		return errors.New("environment is required")
	}
	if request.AppName == "" {
		return errors.New("app_name is required")
	}
	if request.Version == "" {
		return errors.New("version is required")
	}
	if _, ok := defaultImageVersions[request.AppName]; !ok {
		return fmt.Errorf("unsupported app_name: %s", request.AppName)
	}

	overlayDir := filepath.Join(u.overlaysRoot, request.Environment)
	info, err := os.Stat(overlayDir)
	if err != nil {
		return fmt.Errorf("gitops overlay not found: %w", err)
	}
	if !info.IsDir() {
		return errors.New("gitops overlay path is not directory")
	}

	patchPath := filepath.Join(overlayDir, "patch-images.yaml")
	patches, err := loadPatches(patchPath)
	if err != nil {
		return err
	}
	patches[request.AppName] = newDeploymentPatch(request.AppName, request.Version)

	return writePatches(patchPath, patches)
}

func loadPatches(path string) (map[string]deploymentPatch, error) {
	patches := defaultDeploymentPatches()
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return patches, nil
		}
		return nil, err
	}

	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	for {
		var patch deploymentPatch
		if decodeErr := decoder.Decode(&patch); decodeErr != nil {
			if errors.Is(decodeErr, io.EOF) {
				break
			}
			return nil, decodeErr
		}
		if patch.Metadata.Name == "" {
			continue
		}
		patches[patch.Metadata.Name] = patch
	}
	return patches, nil
}

func writePatches(path string, patches map[string]deploymentPatch) error {
	names := make([]string, 0, len(patches))
	for name := range patches {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return orderOf(names[i]) < orderOf(names[j])
	})

	var builder strings.Builder
	encoder := yaml.NewEncoder(&builder)
	encoder.SetIndent(2)
	for _, name := range names {
		if err := encoder.Encode(patches[name]); err != nil {
			return err
		}
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func newDeploymentPatch(appName string, version string) deploymentPatch {
	patch := deploymentPatch{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	}
	patch.Metadata.Name = appName
	patch.Metadata.Namespace = gitOpsNamespace
	patch.Spec.Template.Spec.Containers = []struct {
		Name  string `yaml:"name"`
		Image string `yaml:"image"`
	}{
		{
			Name:  appName,
			Image: imageRef(appName, version),
		},
	}
	return patch
}

func imageRef(appName string, version string) string {
	return fmt.Sprintf("ghcr.io/example/go-cloud-%s:%s", appName, version)
}

func orderOf(appName string) int {
	for index, name := range imageOrder {
		if name == appName {
			return index
		}
	}
	return len(imageOrder) + 1
}

func defaultDeploymentPatches() map[string]deploymentPatch {
	patches := make(map[string]deploymentPatch, len(defaultImageVersions))
	for appName, version := range defaultImageVersions {
		patches[appName] = newDeploymentPatch(appName, version)
	}
	return patches
}
