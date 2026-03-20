package service

import (
	"context"
	"errors"
	"fmt"
	"server/global"
	"server/model"
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

func (s *DatasetService) GetAllDatasets(userID string, role model.UserRole, page int, pageSize int) ([]model.DatasetListItem, int64, error) {
	ctx := context.Background()
	// var results []model.DatasetListItem
	results := make([]model.DatasetListItem, 0)
	var total int64

	db := global.DB.WithContext(ctx).
		Model(&model.Dataset{}).
		Joins("LEFT JOIN users ON users.id = datasets.creator_id")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count records: %w", err)
	}

	selectQuery := `
		datasets.id, 
		datasets.title, 
		datasets.created_at, 
		users.nick_name as creator, 
		(SELECT COUNT(*) FROM files WHERE files.owner_id = datasets.id AND files.owner_type = 'datasets' AND files.deleted_at IS NULL) as file_count
	`

	if role > model.Common {
		// 管理员：默认拥有所有权限
		selectQuery += ", 1 as has_access"
	} else {
		// 普通用户：检查是否存在已通过的申请
		selectQuery += fmt.Sprintf(`, 
			(SELECT COUNT(*) FROM applications 
			 WHERE applications.dataset_id = datasets.id 
			 AND applications.applicant_id = '%s' 
			 AND applications.status = %d
			 AND (applications.expire_at IS NULL OR applications.expire_at > '%s')) > 0 as has_access`, userID, model.APPROVED, time.Now().Format("2006-01-02 15:04:05"))
	}

	offset := (page - 1) * pageSize
	err := db.Select(selectQuery).
		Offset(offset).
		Limit(pageSize).
		Order("datasets.created_at DESC"). // 默认使用数据集创建时间的降序
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

	// 3. 普通用户查申请记录
	var count int64
	err = global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Where("dataset_id = ? AND applicant_id = ? AND status = ?", id, userID, model.APPROVED).
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

	if err := NewFileService().DeleteFilesByDataset(id); err != nil {
		return fmt.Errorf("failed to delete files under dataset %s: %w", id, err)
	}

	res := global.DB.WithContext(ctx).Where("id = ?", id).Delete(&model.Dataset{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete dataset: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("dataset not found")
	} else if res.RowsAffected > 1 {
		return fmt.Errorf("more than 1 dataset affected")
	}
	return nil
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
