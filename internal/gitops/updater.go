package gitops

import "context"

type UpdateRequest struct {
	Environment string
	AppName     string
	Version     string
}

type Updater interface {
	UpdateImage(ctx context.Context, request UpdateRequest) error
}
