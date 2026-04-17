package service

import (
	"context"
	"errors"
	"fmt"
	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/utils"
	"time"

	"gorm.io/gorm"
)

type DatasetService struct {
}

func NewDatasetService() *DatasetService {
	return &DatasetService{}
}

func (s *DatasetService) CreateDataset(dataset *model.Dataset) error {
	ctx := context.Background()
	err := gorm.G[model.Dataset](global.DB).Create(ctx, dataset)
	if err != nil {
		return fmt.Errorf("failed to create new dataset: %w", err)
	}

	return nil
}

// DatasetListQuery 前端表格的筛选/排序/分页参数
type DatasetListQuery struct {
	UserID    string
	Role      model.UserRole
	Page      int
	PageSize  int
	SortBy    string // 前端列 prop，如 "name", "createdAt", "creator"
	Order     string // "asc" 或 "desc"
	IDPrefix  string // ID 前缀（8位）或完整 UUID（36位）精确匹配
	Name      string // 模糊搜索数据集标题
	Creator   string // 模糊搜索创建者昵称
	DateStart string
	DateEnd   string
}

// datasetSortWhitelist 将前端 el-table prop 映射到安全的数据库列
var datasetSortWhitelist = map[string]string{
	"name":      "datasets.title",
	"createdAt": "datasets.created_at",
	"creator":   "users.nick_name",
	"fileCount": "file_count",
}

func (s *DatasetService) GetAllDatasets(q DatasetListQuery) ([]model.DatasetListItem, int64, error) {
	ctx := context.Background()
	// var results []model.DatasetListItem
	results := make([]model.DatasetListItem, 0)
	var total int64

	baseQuery := global.DB.WithContext(ctx).
		Model(&model.Dataset{}).
		Joins("LEFT JOIN users ON users.id = datasets.creator_id")

	// ── ID 前缀 / 精确匹配 ──
	if q.IDPrefix != "" {
		if len(q.IDPrefix) == 36 {
			baseQuery = baseQuery.Where("datasets.id = ?", q.IDPrefix)
		} else {
			baseQuery = baseQuery.Where("datasets.id LIKE ?", q.IDPrefix+"%")
		}
	}
	// ── 数据集名称搜索 ──
	if q.Name != "" {
		if len(q.Name) == 36 || (len(q.Name) >= 8 && utils.IsHex(q.Name)) {
			baseQuery = baseQuery.Where("datasets.title LIKE ? OR datasets.id LIKE ?", "%"+q.Name+"%", q.Name+"%")
		} else {
			baseQuery = baseQuery.Where("datasets.title LIKE ?", "%"+q.Name+"%")
		}
	}
	// ── 创建者昵称搜索 ──
	if q.Creator != "" {
		if len(q.Creator) == 36 || (len(q.Creator) >= 8 && utils.IsHex(q.Creator)) {
			baseQuery = baseQuery.Where("users.nick_name LIKE ? OR users.id LIKE ?", "%"+q.Creator+"%", q.Creator+"%")
		} else {
			baseQuery = baseQuery.Where("users.nick_name LIKE ?", "%"+q.Creator+"%")
		}
	}
	// ── 日期范围（基于 created_at）──
	loc := time.Local
	if q.DateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateStart, loc); err == nil {
			baseQuery = baseQuery.Where("datasets.created_at >= ?", t)
		} else {
			logger.Errorf("Failed to parse DateStart. Value: '%s', Error: %v", q.DateStart, err)
		}
	}
	if q.DateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateEnd, loc); err == nil {
			baseQuery = baseQuery.Where("datasets.created_at < ?", t.AddDate(0, 0, 1))
		} else {
			logger.Errorf("Failed to parse DateEnd. Value: '%s', Error: %v", q.DateEnd, err)
		}
	}
	// ── 计数 ──
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count records: %w", err)
	}

	selectQuery := `
		datasets.id, 
		datasets.title, 
		datasets.created_at, 
		users.nick_name as creator, 
		(SELECT COUNT(*) FROM files WHERE files.owner_id = datasets.id AND files.owner_type = 'datasets' AND files.deleted_at IS NULL) as file_count
	`

	if q.Role > model.Common {
		selectQuery += ", 1 as has_access"
	} else {
		selectQuery += fmt.Sprintf(`, 
			(SELECT COUNT(*) FROM licenses 
			 WHERE licenses.dataset_id = datasets.id 
			 AND licenses.applicant_id = '%s' 
			 AND licenses.status = %d
			 AND licenses.deleted_at IS NULL
			 AND (licenses.expire_at IS NULL OR licenses.expire_at > '%s')) > 0 as has_access`,
			q.UserID, model.LicenseNormal, time.Now().Format("2006-01-02 15:04:05"))
	}
	// ── 排序（白名单校验）──
	orderClause := "datasets.created_at DESC"
	if col, ok := datasetSortWhitelist[q.SortBy]; ok {
		dir := "ASC"
		if q.Order == "desc" {
			dir = "DESC"
		}
		orderClause = col + " " + dir
	}
	// ── 分页查询 ──
	offset := (q.Page - 1) * q.PageSize
	err := baseQuery.
		Select(selectQuery).
		Order(orderClause).
		Offset(offset).
		Limit(q.PageSize).
		Scan(&results).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get dataset list: %w", err)
	}

	return results, total, nil
}

func (s *DatasetService) GetDatasetDetail(id string, userID string, role model.UserRole) (*model.Dataset, bool, error) {
	var dataset model.Dataset
	ctx := context.Background()

	// 1. 查数据 (包含关联的文件列表)
	err := global.DB.WithContext(ctx).
		Preload("Files").
		Preload("Creator").
		Where("id = ?", id).
		First(&dataset).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, gorm.ErrRecordNotFound
		}
		return nil, false, fmt.Errorf("failed to query dataset: %w", err)
	}

	// 2. 权限判定
	// 如果是管理员，直接拥有权限
	if role > model.Common {
		return &dataset, true, nil
	}

	// 3. 普通用户查许可记录
	var count int64
	err = global.DB.WithContext(ctx).
		Model(&model.License{}).
		Where("dataset_id = ? AND applicant_id = ? AND status = ?", id, userID, model.LicenseNormal).
		Where("expire_at IS NULL OR expire_at > ?", time.Now()).
		Count(&count).Error

	if err != nil {
		return nil, false, fmt.Errorf("failed to verify permission: %w", err)
	}

	// 关键修改：不再报错，而是通过布尔值告知是否有权
	hasAccess := count > 0

	return &dataset, hasAccess, nil
}

func (s *DatasetService) UpdateDataset(id string, title, description string) (*model.Dataset, error) {
	ctx := context.Background()
	updates := map[string]any{
		"title":       title,
		"description": description,
	}

	res := global.DB.WithContext(ctx).Model(&model.Dataset{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("failed to update dataset: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("dataset not found")
	} else if res.RowsAffected > 1 {
		return nil, fmt.Errorf("more than 1 dataset affected")
	}

	updated, err := gorm.G[model.Dataset](global.DB).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query updated dataset: %w", err)
	}

	return &updated, nil
}

func (s *DatasetService) DeleteDataset(id string) error {
	ctx := context.Background()

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := NewFileService().DeleteFilesByDataset(id); err != nil {
			return fmt.Errorf("failed to delete files under dataset %s: %w", id, err)
		}

		if err := tx.Model(&model.License{}).
			Where("dataset_id = ? AND status = ?", id, model.LicenseNormal).
			Updates(map[string]interface{}{
				"status":        model.LicenseRevoked,
				"revoke_reason": "对应数据集已被移除",
			}).Error; err != nil {
			return fmt.Errorf("failed to revoke licenses: %w", err)
		}

		res := tx.Where("id = ?", id).Delete(&model.Dataset{})
		if res.Error != nil {
			return fmt.Errorf("failed to delete dataset: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("dataset not found")
		} else if res.RowsAffected > 1 {
			return fmt.Errorf("more than 1 dataset affected")
		}
		return nil
	})

	return err
}

func (s *DatasetService) GetDatasetStatistics(datasetID string) (*model.DatasetStatistics, error) {
	var stats model.DatasetStatistics
	ctx := context.Background()

	// 1. 本数据集的申请总量
	if err := global.DB.WithContext(ctx).Model(&model.Application{}).
		Where("dataset_id = ?", datasetID).
		Count(&stats.TotalApplications).Error; err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}

	// 2. 申请提交（Audit）总次数
	// 假设 audits 表通过 application_id 关联 applications 表
	if err := global.DB.WithContext(ctx).Model(&model.Audit{}).
		Joins("JOIN applications ON audits.application_id = applications.id").
		Where("applications.dataset_id = ?", datasetID).
		Count(&stats.TotalAudits).Error; err != nil {
		return nil, fmt.Errorf("failed to count audits: %w", err)
	}

	// 3. 数据集申请通过总量
	if err := global.DB.WithContext(ctx).Model(&model.Application{}).
		Where("dataset_id = ? AND status = ?", datasetID, model.APPROVED).
		Count(&stats.ApprovedCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count approved applications: %w", err)
	}

	// 4. 通过率
	if stats.TotalApplications > 0 {
		stats.ApprovalRate = float64(stats.ApprovedCount) / float64(stats.TotalApplications)
	}

	// 5. 提交次数分布 (基于 application.version)
	// version=0 为草稿(未提交)，version=1 为首次提交，version>1 为重新提交
	stats.SubmissionDistribution = make(map[int]int64)
	rows, err := global.DB.WithContext(ctx).Model(&model.Application{}).
		Select("version, count(*) as count").
		Where("dataset_id = ?", datasetID).
		Group("version").
		Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var version int
			var count int64
			if rows.Scan(&version, &count) == nil {
				stats.SubmissionDistribution[version] = count
			}
		}
	}

	return &stats, nil
}
