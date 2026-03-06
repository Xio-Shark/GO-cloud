package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"go-cloud/internal/gitops"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	environment := flag.String("environment", "", "target environment")
	appName := flag.String("app", "", "application name")
	version := flag.String("version", "", "image version")
	overlaysRoot := flag.String("overlays-root", "deployments/k8s/overlays", "gitops overlays root")
	flag.Parse()

	updater := gitops.NewFileUpdater(*overlaysRoot)
	if err := updater.UpdateImage(context.Background(), gitops.UpdateRequest{
		Environment: *environment,
		AppName:     *appName,
		Version:     *version,
	}); err != nil {
		slog.Default().Error("release updater failed", "error", err, "environment", *environment, "app_name", *appName, "version", *version)
		os.Exit(1)
	}
	slog.Default().Info("release updater succeeded", "environment", *environment, "app_name", *appName, "version", *version)
}
