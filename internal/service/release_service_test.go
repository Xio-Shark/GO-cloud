package service

import (
	"context"
	"strings"
	"testing"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/gitops"
	"go-cloud/internal/repository"
)

func TestReleaseServiceCreateReleaseRejectsMissingAppName(t *testing.T) {
	svc := NewReleaseService(&releaseRepoStub{}, &gitopsUpdaterStub{})

	_, err := svc.CreateRelease(context.Background(), dto.CreateReleaseRequest{
		Version:     "v1.0.0",
		Environment: "dev",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "app_name is required") {
		t.Fatalf("expected app_name validation, got %v", err)
	}
}

func TestReleaseServiceCreateReleaseWritesGitOpsAndMarksDeployed(t *testing.T) {
	repo := &releaseRepoStub{}
	updater := &gitopsUpdaterStub{}
	svc := NewReleaseService(repo, updater)

	release, err := svc.CreateRelease(context.Background(), dto.CreateReleaseRequest{
		AppName:     "api-server",
		Version:     "v1.0.0",
		Environment: "dev",
		Operator:    "tester",
	})
	if err != nil {
		t.Fatalf("CreateRelease returned error: %v", err)
	}
	if release.Status != domain.ReleaseStatusDeployed {
		t.Fatalf("expected deployed status, got %s", release.Status)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 saved release, got %d", len(repo.items))
	}
	if len(updater.calls) != 1 {
		t.Fatalf("expected 1 gitops call, got %d", len(updater.calls))
	}
	if updater.calls[0].Version != "v1.0.0" {
		t.Fatalf("expected version v1.0.0, got %s", updater.calls[0].Version)
	}
}

func TestReleaseServiceCreateReleaseStoresFailedRecordWhenGitOpsFails(t *testing.T) {
	repo := &releaseRepoStub{}
	updater := &gitopsUpdaterStub{err: ValidationError("gitops failed")}
	svc := NewReleaseService(repo, updater)

	release, err := svc.CreateRelease(context.Background(), dto.CreateReleaseRequest{
		AppName:     "api-server",
		Version:     "v1.0.1",
		Environment: "dev",
		Operator:    "tester",
	})
	if err == nil {
		t.Fatal("expected gitops error")
	}
	if !strings.Contains(err.Error(), "gitops failed") {
		t.Fatalf("expected gitops failure, got %v", err)
	}
	if release != nil {
		t.Fatal("expected nil release on failure")
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected failed release to be stored, got %d items", len(repo.items))
	}
	if repo.items[0].Status != domain.ReleaseStatusFailed {
		t.Fatalf("expected failed status, got %s", repo.items[0].Status)
	}
}

func TestReleaseServiceRollbackReleaseCreatesRolledBackRecord(t *testing.T) {
	repo := &releaseRepoStub{
		items: []domain.ReleaseRecord{
			{
				ID:          1,
				AppName:     "worker",
				Version:     "v1.2.3",
				Environment: "prod",
				Status:      domain.ReleaseStatusDeployed,
			},
		},
	}
	updater := &gitopsUpdaterStub{}
	svc := NewReleaseService(repo, updater)

	release, err := svc.RollbackRelease(context.Background(), 1, dto.RollbackReleaseRequest{
		Operator:  "tester",
		ChangeLog: "rollback reason",
	})
	if err != nil {
		t.Fatalf("RollbackRelease returned error: %v", err)
	}
	if release.Status != domain.ReleaseStatusRolledBack {
		t.Fatalf("expected rolled_back status, got %s", release.Status)
	}
	if len(repo.items) != 2 {
		t.Fatalf("expected 2 release records, got %d", len(repo.items))
	}
	if repo.items[1].Version != "v1.2.3" {
		t.Fatalf("expected rollback version v1.2.3, got %s", repo.items[1].Version)
	}
}

type releaseRepoStub struct {
	items []domain.ReleaseRecord
}

func (s *releaseRepoStub) Create(_ context.Context, release *domain.ReleaseRecord) error {
	release.ID = int64(len(s.items) + 1)
	copyItem := *release
	s.items = append(s.items, copyItem)
	return nil
}

func (s *releaseRepoStub) GetByID(_ context.Context, id int64) (*domain.ReleaseRecord, error) {
	for _, item := range s.items {
		if item.ID == id {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
}

func (s *releaseRepoStub) List(_ context.Context, filter repository.ReleaseListFilter) ([]domain.ReleaseRecord, int64, error) {
	items := make([]domain.ReleaseRecord, 0, len(s.items))
	for _, item := range s.items {
		if filter.Environment != nil && item.Environment != *filter.Environment {
			continue
		}
		if filter.Status != nil && item.Status != *filter.Status {
			continue
		}
		items = append(items, item)
	}
	return items, int64(len(items)), nil
}

type gitopsUpdaterStub struct {
	calls []gitops.UpdateRequest
	err   error
}

func (s *gitopsUpdaterStub) UpdateImage(_ context.Context, request gitops.UpdateRequest) error {
	s.calls = append(s.calls, request)
	return s.err
}
