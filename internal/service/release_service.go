package service

import (
	"context"
	"fmt"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/gitops"
	"go-cloud/internal/repository"
)

type ReleaseService interface {
	CreateRelease(ctx context.Context, req dto.CreateReleaseRequest) (*domain.ReleaseRecord, error)
	GetRelease(ctx context.Context, id int64) (*domain.ReleaseRecord, error)
	ListReleases(ctx context.Context, filter repository.ReleaseListFilter) ([]domain.ReleaseRecord, int64, error)
	RollbackRelease(ctx context.Context, id int64, req dto.RollbackReleaseRequest) (*domain.ReleaseRecord, error)
}

type releaseService struct {
	releaseRepo repository.ReleaseRepository
	updater     gitops.Updater
}

func NewReleaseService(releaseRepo repository.ReleaseRepository, updater gitops.Updater) ReleaseService {
	return &releaseService{
		releaseRepo: releaseRepo,
		updater:     updater,
	}
}

func (s *releaseService) CreateRelease(ctx context.Context, req dto.CreateReleaseRequest) (*domain.ReleaseRecord, error) {
	if err := validateCreateRelease(req); err != nil {
		return nil, err
	}
	if err := s.syncGitOps(ctx, req.AppName, req.Version, req.Environment, req.Operator, req.ChangeLog); err != nil {
		return nil, err
	}

	release := newReleaseRecord(req.AppName, req.Version, req.Environment, defaultReleaseStatus(req.Status), req.Operator, req.ChangeLog)
	if release.Status == domain.ReleaseStatusPending {
		release.Status = domain.ReleaseStatusDeployed
	}
	if err := s.releaseRepo.Create(ctx, release); err != nil {
		return nil, err
	}
	return release, nil
}

func (s *releaseService) GetRelease(ctx context.Context, id int64) (*domain.ReleaseRecord, error) {
	release, err := s.releaseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, NotFoundError("release not found")
	}
	return release, nil
}

func (s *releaseService) ListReleases(ctx context.Context, filter repository.ReleaseListFilter) ([]domain.ReleaseRecord, int64, error) {
	return s.releaseRepo.List(ctx, filter)
}

func (s *releaseService) RollbackRelease(ctx context.Context, id int64, req dto.RollbackReleaseRequest) (*domain.ReleaseRecord, error) {
	target, err := s.GetRelease(ctx, id)
	if err != nil {
		return nil, err
	}
	if target.Status == domain.ReleaseStatusFailed {
		return nil, ConflictError("failed release cannot be rollback target")
	}
	if err := s.syncGitOps(ctx, target.AppName, target.Version, target.Environment, req.Operator, req.ChangeLog); err != nil {
		return nil, err
	}

	release := newReleaseRecord(
		target.AppName,
		target.Version,
		target.Environment,
		domain.ReleaseStatusRolledBack,
		req.Operator,
		appendChangeLog(req.ChangeLog, fmt.Sprintf("rollback_target_release_id=%d", target.ID)),
	)
	if err := s.releaseRepo.Create(ctx, release); err != nil {
		return nil, err
	}
	return release, nil
}

func validateCreateRelease(req dto.CreateReleaseRequest) error {
	if req.AppName == "" {
		return ValidationError("app_name is required")
	}
	if req.Version == "" {
		return ValidationError("version is required")
	}
	if req.Environment == "" {
		return ValidationError("environment is required")
	}
	return nil
}

func defaultReleaseStatus(status string) string {
	if status == "" {
		return domain.ReleaseStatusPending
	}
	return status
}

func newReleaseRecord(appName string, version string, environment string, status string, operator string, changeLog string) *domain.ReleaseRecord {
	now := time.Now().UTC()
	return &domain.ReleaseRecord{
		AppName:     appName,
		Version:     version,
		Environment: environment,
		Status:      status,
		Operator:    defaultActor(operator),
		ChangeLog:   changeLog,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func appendChangeLog(base string, extra string) string {
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + "\n" + extra
}

func (s *releaseService) syncGitOps(ctx context.Context, appName string, version string, environment string, operator string, changeLog string) error {
	if s.updater == nil {
		return ValidationError("gitops updater is not configured")
	}
	if err := s.updater.UpdateImage(ctx, gitops.UpdateRequest{
		Environment: environment,
		AppName:     appName,
		Version:     version,
	}); err != nil {
		return s.saveFailedRelease(ctx, appName, version, environment, operator, changeLog, err)
	}
	return nil
}

func (s *releaseService) saveFailedRelease(ctx context.Context, appName string, version string, environment string, operator string, changeLog string, failure error) error {
	release := newReleaseRecord(appName, version, environment, domain.ReleaseStatusFailed, operator, appendChangeLog(changeLog, failure.Error()))
	if err := s.releaseRepo.Create(ctx, release); err != nil {
		return err
	}
	return failure
}
