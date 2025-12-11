package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"server/global"
	"server/model"
)

type ApplicationService struct{}

func NewApplicationService() *ApplicationService {
	return &ApplicationService{}
}

func (s *ApplicationService) CreateApplication(app *model.Application) error {
	ctx := context.Background()

	if err := s.ensureDatasetExists(ctx, app.DatasetID); err != nil {
		return err
	}

	if err := gorm.G[model.Application](global.DB).Create(ctx, app); err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return nil
}

func (s *ApplicationService) ListApplications() ([]model.Application, error) {
	ctx := context.Background()
	var applications []model.Application
	err := global.DB.WithContext(ctx).
		Preload("Applicant").
		Preload("Dataset").
		Order("created_at DESC").
		Find(&applications).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	return applications, nil
}

func (s *ApplicationService) GetApplicationByID(id string) (*model.Application, error) {
	ctx := context.Background()
	var application model.Application
	err := global.DB.WithContext(ctx).
		Preload("Applicant").
		Preload("Dataset").
		Where("id = ?", id).
		First(&application).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query application %s: %w", id, err)
	}
	return &application, nil
}

func (s *ApplicationService) ensureDatasetExists(ctx context.Context, datasetID string) error {
	_, err := gorm.G[model.Dataset](global.DB).
		Select("id").
		Where("id = ?", datasetID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return fmt.Errorf("failed to query dataset %s: %w", datasetID, err)
	}
	return nil
}
