package service

import (
	"context"
	"fmt"
	"server/global"
	"server/model"
	"server/pkg/utils"
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
	IDPrefix      string // 许可 ID 前缀（8位）或完整 UUID（36位）
	DatasetName   string // 模糊搜索数据集标题
	Applicant     string // 模糊搜索获得者昵称
	Status        string // "normal" or "revoked"
	DateStart     string // "YYYY-MM-DD" 授权日期开始
	DateEnd       string // "YYYY-MM-DD" 授权日期结束
	ExpireStart   string // "YYYY-MM-DD" 过期日期开始
	ExpireEnd     string // "YYYY-MM-DD" 过期日期结束
}

var licenseSortColumnWhitelist = map[string]string{
	"createdAt":    "licenses.created_at",
	"expireAt":     "licenses.expire_at",
	"datasetTitle": "datasets.title",
	"applicantName": "users.nick_name",
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

	// ── 来源申请 ID（精确 UUID，用于从 ApplicationView 跳转）──
	if q.ApplicationID != "" {
		if len(q.ApplicationID) == 36 {
			baseQuery = baseQuery.Where("licenses.application_id = ?", q.ApplicationID)
		} else {
			baseQuery = baseQuery.Where("licenses.application_id LIKE ?", q.ApplicationID+"%")
		}
	}

	// ── 许可 ID 前缀 / 精确匹配 ──
	if q.IDPrefix != "" {
		if len(q.IDPrefix) == 36 {
			baseQuery = baseQuery.Where("licenses.id = ?", q.IDPrefix)
		} else {
			baseQuery = baseQuery.Where("licenses.id LIKE ?", q.IDPrefix+"%")
		}
	}

	// ── 数据集名称搜索 ──
	if q.DatasetName != "" {
		if len(q.DatasetName) == 36 || (len(q.DatasetName) >= 8 && utils.IsHex(q.DatasetName)) {
			baseQuery = baseQuery.Where("datasets.title LIKE ? OR datasets.id LIKE ?", "%"+q.DatasetName+"%", q.DatasetName+"%")
		} else {
			baseQuery = baseQuery.Where("datasets.title LIKE ?", "%"+q.DatasetName+"%")
		}
	}

	// ── 获得者昵称搜索 ──
	if q.Applicant != "" {
		if len(q.Applicant) == 36 || (len(q.Applicant) >= 8 && utils.IsHex(q.Applicant)) {
			baseQuery = baseQuery.Where("users.nick_name LIKE ? OR users.id LIKE ?", "%"+q.Applicant+"%", q.Applicant+"%")
		} else {
			baseQuery = baseQuery.Where("users.nick_name LIKE ?", "%"+q.Applicant+"%")
		}
	}

	// ── 状态筛选 ──
	if q.Status != "" {
		if q.Status == "normal" {
			baseQuery = baseQuery.Where("licenses.status = ? AND (licenses.expire_at IS NULL OR licenses.expire_at > ?)", model.LicenseNormal, time.Now())
		} else if q.Status == "revoked" {
			baseQuery = baseQuery.Where("licenses.status = ?", model.LicenseRevoked)
		} else if q.Status == "expired" {
			baseQuery = baseQuery.Where("licenses.status = ? AND licenses.expire_at <= ?", model.LicenseNormal, time.Now())
		}
	}

	// ── 授权日期范围（基于 created_at）──
	loc := time.Local
	if q.DateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateStart, loc); err == nil {
			baseQuery = baseQuery.Where("licenses.created_at >= ?", t)
		}
	}
	if q.DateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateEnd, loc); err == nil {
			baseQuery = baseQuery.Where("licenses.created_at < ?", t.AddDate(0, 0, 1))
		}
	}
	// ── 过期日期范围（基于 expire_at）──
	if q.ExpireStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.ExpireStart, loc); err == nil {
			baseQuery = baseQuery.Where("licenses.expire_at >= ?", t)
		}
	}
	if q.ExpireEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.ExpireEnd, loc); err == nil {
			baseQuery = baseQuery.Where("licenses.expire_at < ?", t.AddDate(0, 0, 1))
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
			return db.Unscoped().Select("id, nick_name, email, deleted_at")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Unscoped().Select("id, title, deleted_at")
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
