package service

import (
	"context"
	"errors"
	"fmt"
	"server/global"
	"server/model"

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

func (s *DatasetService) GetAllDatasets() ([]model.DatasetListItem, error) {
	ctx := context.Background()
	var results []model.DatasetListItem

	err := global.DB.WithContext(ctx).
		Model(&model.Dataset{}).
		Select(`
            datasets.id, 
            datasets.title, 
            datasets.created_at, 
            users.nick_name as creator, 
            (SELECT COUNT(*) FROM files WHERE files.owner_id = datasets.id AND files.owner_type = 'datasets') as file_count
        `).
		Joins("LEFT JOIN users ON users.id = datasets.creator_id").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get dataset list: %w", err)
	}

	return results, nil
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
