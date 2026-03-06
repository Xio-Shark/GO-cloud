package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileUpdaterUpdateImageWritesPatchFile(t *testing.T) {
	root := t.TempDir()
	overlayDir := filepath.Join(root, "deployments", "k8s", "overlays", "dev")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	updater := NewFileUpdater(filepath.Join(root, "deployments", "k8s", "overlays"))
	if err := updater.UpdateImage(context.Background(), UpdateRequest{
		Environment: "dev",
		AppName:     "api-server",
		Version:     "v1.2.3",
	}); err != nil {
		t.Fatalf("UpdateImage returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(overlayDir, "patch-images.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ghcr.io/example/go-cloud-api-server:v1.2.3") {
		t.Fatalf("expected api-server image mapping, got %s", content)
	}
	if !strings.Contains(content, "name: api-server") {
		t.Fatalf("expected api-server deployment patch, got %s", content)
	}
	if strings.Contains(content, "---\n---") {
		t.Fatalf("expected no duplicated document separators, got %s", content)
	}
}

func TestFileUpdaterUpdateImageReturnsErrorWhenOverlayMissing(t *testing.T) {
	updater := NewFileUpdater(filepath.Join(t.TempDir(), "deployments", "k8s", "overlays"))

	err := updater.UpdateImage(context.Background(), UpdateRequest{
		Environment: "prod",
		AppName:     "worker",
		Version:     "v2.0.0",
	})
	if err == nil {
		t.Fatal("expected error for missing overlay")
	}
}
