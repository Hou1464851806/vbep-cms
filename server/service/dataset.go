package service

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"server/global"
	"server/model"
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

func (s *DatasetService) GetAllDatasets() ([]model.Dataset, error) {
	ctx := context.Background()
	datasets, err := gorm.G[model.Dataset](global.DB).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all datasets: %w", err)
	}

	return datasets, nil
}

func (s *DatasetService) GetDatasets() {

}

func (s *DatasetService) UpdateDataset() {

}

func (s *DatasetService) DeleteDataset() {

}
