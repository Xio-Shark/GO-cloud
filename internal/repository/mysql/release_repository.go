package mysqlrepo

import (
	"context"
	"errors"

	"go-cloud/internal/domain"
	"go-cloud/internal/repository"
	"go-cloud/internal/repository/model"

	"gorm.io/gorm"
)

type ReleaseRepository struct {
	db *gorm.DB
}

func NewReleaseRepository(db *gorm.DB) repository.ReleaseRepository {
	return &ReleaseRepository{db: db}
}

func (r *ReleaseRepository) Create(ctx context.Context, release *domain.ReleaseRecord) error {
	record := releaseToModel(*release)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	release.ID = record.ID
	return nil
}

func (r *ReleaseRepository) GetByID(ctx context.Context, id int64) (*domain.ReleaseRecord, error) {
	var record model.ReleaseRecord
	err := r.db.WithContext(ctx).First(&record, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := releaseFromModel(record)
	return &result, nil
}

func (r *ReleaseRepository) List(ctx context.Context, filter repository.ReleaseListFilter) ([]domain.ReleaseRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ReleaseRecord{})
	if filter.Environment != nil {
		query = query.Where("environment = ?", *filter.Environment)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := normalizePage(filter.Page)
	pageSize := normalizePageSize(filter.PageSize)
	var records []model.ReleaseRecord
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.ReleaseRecord, 0, len(records))
	for _, record := range records {
		items = append(items, releaseFromModel(record))
	}
	return items, total, nil
}

func releaseToModel(release domain.ReleaseRecord) model.ReleaseRecord {
	return model.ReleaseRecord{
		ID:          release.ID,
		AppName:     release.AppName,
		Version:     release.Version,
		Environment: release.Environment,
		Status:      release.Status,
		Operator:    release.Operator,
		ChangeLog:   release.ChangeLog,
		CreatedAt:   release.CreatedAt,
		UpdatedAt:   release.UpdatedAt,
	}
}

func releaseFromModel(record model.ReleaseRecord) domain.ReleaseRecord {
	return domain.ReleaseRecord{
		ID:          record.ID,
		AppName:     record.AppName,
		Version:     record.Version,
		Environment: record.Environment,
		Status:      record.Status,
		Operator:    record.Operator,
		ChangeLog:   record.ChangeLog,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
