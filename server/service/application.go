package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"server/global"
	"server/model"
)

var ErrApplicationAlreadyProcessed = errors.New("application already processed")

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

func (s *ApplicationService) ListApplications(applicantID *string) ([]model.Application, error) {
	ctx := context.Background()
	var applications []model.Application

	db := global.DB.WithContext(ctx).
		Preload("Applicant").
		Preload("Dataset").
		Preload("Auditor").
		Order("created_at DESC")

	// 如果是普通用户，只查询自己的申请
	if applicantID != nil {
		db = db.Where("applicant_id = ?", *applicantID)
	}

	err := db.Find(&applications).Error
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
		Preload("Auditor").
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

func (s *ApplicationService) ApproveApplication(id, auditorID, result string) (*model.Application, error) {
	return s.updateApplicationStatus(id, auditorID, result, model.APPROVED)
}

func (s *ApplicationService) RejectApplication(id, auditorID, result string) (*model.Application, error) {
	return s.updateApplicationStatus(id, auditorID, result, model.DISAPPROVED)
}

func (s *ApplicationService) updateApplicationStatus(id, auditorID, result string, status model.ApplicationStatus) (*model.Application, error) {
	ctx := context.Background()
	updates := map[string]any{
		"status":         status,
		"auditor_id":     auditorID,
		"audited_at":     time.Now(),
		"auditor_result": result,
	}

	res := global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Where("id = ? AND status = ?", id, model.PENDING).
		Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("failed to update application %s: %w", id, res.Error)
	}
	if res.RowsAffected == 0 {
		var existing model.Application
		err := global.DB.WithContext(ctx).Where("id = ?", id).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, fmt.Errorf("failed to query application %s: %w", id, err)
		}
		if existing.Status != model.PENDING {
			return nil, ErrApplicationAlreadyProcessed
		}
		return nil, fmt.Errorf("failed to update application %s", id)
	}

	return s.GetApplicationByID(id)
}
