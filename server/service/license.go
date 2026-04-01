package service

import (
	"context"
	"fmt"
	"server/global"
	"server/model"
	"time"

	"gorm.io/gorm"
)

type LicenseService struct{}

func NewLicenseService() *LicenseService {
	return &LicenseService{}
}

type LicenseListQuery struct {
	ApplicantID   *string
	ApplicationID string
	Page          int
	PageSize      int
	SortBy        string
	Order         string
	Keyword       string
	Status        string // "normal" or "revoked"
}

var licenseSortColumnWhitelist = map[string]string{
	"createdAt":    "licenses.created_at",
	"expireAt":     "licenses.expire_at",
	"datasetTitle": "datasets.title",
	"applicant":    "users.nick_name",
}

func (s *LicenseService) ListLicenses(q LicenseListQuery) ([]model.LicenseListItem, int64, error) {
	ctx := context.Background()
	var licenses []model.License
	var total int64

	baseQuery := global.DB.WithContext(ctx).
		Model(&model.License{}).
		Joins("LEFT JOIN users ON users.id = licenses.applicant_id").
		Joins("LEFT JOIN datasets ON datasets.id = licenses.dataset_id")

	if q.ApplicantID != nil {
		baseQuery = baseQuery.Where("licenses.applicant_id = ?", *q.ApplicantID)
	}

	if q.ApplicationID != "" {
		baseQuery = baseQuery.Where("licenses.application_id = ?", q.ApplicationID)
	}

	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		baseQuery = baseQuery.Where("users.nick_name LIKE ? OR datasets.title LIKE ? OR licenses.application_id = ? OR licenses.id = ?", kw, kw, q.Keyword, q.Keyword)
	}

	if q.Status != "" {
		if q.Status == "normal" {
			baseQuery = baseQuery.Where("licenses.status = ? AND (licenses.expire_at IS NULL OR licenses.expire_at > ?)", model.LicenseNormal, time.Now())
		} else if q.Status == "revoked" {
			baseQuery = baseQuery.Where("licenses.status = ? OR (licenses.status = ? AND licenses.expire_at <= ?)", model.LicenseRevoked, model.LicenseNormal, time.Now())
		}
	}

	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count licenses: %w", err)
	}

	orderClause := "licenses.created_at DESC"
	if col, ok := licenseSortColumnWhitelist[q.SortBy]; ok {
		dir := "ASC"
		if q.Order == "desc" {
			dir = "DESC"
		}
		orderClause = col + " " + dir
	}

	offset := (q.Page - 1) * q.PageSize
	err := baseQuery.
		Select("licenses.*").
		Order(orderClause).
		Offset(offset).
		Limit(q.PageSize).
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		}).
		Find(&licenses).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query licenses: %w", err)
	}

	result := make([]model.LicenseListItem, len(licenses))
	for i, lic := range licenses {
		result[i] = lic.ToListItem()
	}

	return result, total, nil
}

func (s *LicenseService) RevokeLicense(id string, reason string) error {
	ctx := context.Background()
	res := global.DB.WithContext(ctx).Model(&model.License{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        model.LicenseRevoked,
			"revoke_reason": reason,
		})

	if res.Error != nil {
		return fmt.Errorf("failed to revoke license: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}
