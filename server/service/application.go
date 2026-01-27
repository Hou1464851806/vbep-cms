package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"server/global"
	"server/model"
)

var (
	ErrApplicationAlreadyProcessed = errors.New("application already processed")
	ErrApplicationPermissionDenied = errors.New("permission denied for application")
	ErrApplicationNotEditable      = errors.New("application not editable")
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

	app.Version = 0 // 初始创建为草稿状态
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(app).Error; err != nil {
			return fmt.Errorf("failed to create application: %w", err)
		}

		audit := &model.Audit{
			Action:        model.PENDING,
			ApplicationID: app.ID,
		}

		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("failed to create application audit: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *ApplicationService) DeleteDraftApplication(id, applicantID string) error {
	ctx := context.Background()
	var app model.Application

	if err := global.DB.WithContext(ctx).
		Select("id, applicant_id, version").
		Where("id = ?", id).
		First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("failed to query application: %w", err)
	}

	if app.ApplicantID != applicantID {
		return ErrApplicationPermissionDenied
	}

	if app.Version != 0 {
		return ErrApplicationNotEditable // 只能删除草稿
	}

	// 清理文件
	if err := NewFileService().DeleteFilesByApplication(id); err != nil {
		return err
	}

	// 删除 Application (级联删除 Audit)
	if err := global.DB.WithContext(ctx).Delete(&app).Error; err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}

func (s *ApplicationService) SubmitDraftApplication(id, applicantID string) error {
	ctx := context.Background()
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).
			First(&app).Error; err != nil {
			return err
		}

		if app.ApplicantID != applicantID {
			return ErrApplicationPermissionDenied
		}
		if app.Version != 0 {
			return ErrApplicationAlreadyProcessed
		}

		// 更新版本号为1，正式提交
		// 同时更新关联的 Audit 时间，使其反映用户点击“提交”的时间
		if err := tx.Model(&app).Update("version", 1).Error; err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(&model.Audit{}).
			Where("application_id = ? AND action = ?", id, model.PENDING).
			Updates(map[string]interface{}{"created_at": now}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *ApplicationService) ListApplications(applicantID *string) ([]model.ApplicationListItem, error) {
	ctx := context.Background()
	var apps []model.Application

	db := global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Order("updated_at DESC").
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		})

	// 只允许用户查询自己的申请
	if applicantID != nil {
		db = db.Where("applicant_id = ?", *applicantID)
	}

	if err := db.Find(&apps).Error; err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}

	// 模型转换 (Model -> DTO)
	result := make([]model.ApplicationListItem, len(apps))
	for i, app := range apps {
		result[i] = app.ToListItem()
	}

	return result, nil
}

func (s *ApplicationService) GetApplicationByID(id string) (*model.ApplicationDetail, error) {
	ctx := context.Background()
	var app model.Application

	err := global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		}).
		Preload("Audits", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC").Preload("Operator", func(db *gorm.DB) *gorm.DB {
				return db.Select("id, nick_name, email")
			})
		}).
		Where("id = ?", id).
		First(&app).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query application %s: %w", id, err)
	}

	item := app.ToDetail()
	return &item, nil
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

// ProcessApplicationAudit handles the approval, rejection, or return of an application.
// It updates the application's status and the latest audit record.
// This function replaces ApproveApplication, RejectApplication, and ReturnApplication.
func (s *ApplicationService) ProcessApplicationAudit(id, operatorID, result string, status model.ApplicationStatus) (*model.ApplicationDetail, error) {
	// Validate the incoming status to ensure it's an audit action
	ctx := context.Background()
	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, status").
			Where("id = ?", id).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return fmt.Errorf("failed to query application %s: %w", id, err)
		}
		if existing.Status != model.PENDING {
			return ErrApplicationAlreadyProcessed
		}

		if err := tx.Model(&model.Application{}).
			Where("id = ?", id).
			Update("status", status).Error; err != nil {
			return fmt.Errorf("failed to update application %s: %w", id, err)
		}

		// 修改逻辑：不再创建新 Audit，而是更新最近的一条 Audit (代表当前的提交轮次)
		var lastAudit model.Audit
		if err := tx.Where("application_id = ?", id).Order("created_at DESC").First(&lastAudit).Error; err != nil {
			return fmt.Errorf("failed to find pending audit for application %s: %w", id, err)
		}

		updates := map[string]interface{}{
			"action":      status,
			"result":      result,
			"operator_id": operatorID,
		}

		if err := tx.Model(&lastAudit).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update audit for application %s: %w", id, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetApplicationByID(id)
}

func (s *ApplicationService) GetApplicationAuditDetail(applicationID, auditID string) (*model.AuditDetail, error) {
	var audit model.Audit
	err := global.DB.WithContext(context.Background()).
		Where("id = ? AND application_id = ?", auditID, applicationID).
		Preload("Files").
		Preload("Operator", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Application", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, applicant_id").
				Preload("Applicant", func(appDB *gorm.DB) *gorm.DB {
					return appDB.Select("id, nick_name, email")
				})
		}).
		First(&audit).Error
	if err != nil {
		return nil, err
	}
	detail := audit.ToDetail()

	return &detail, nil
}

func (s *ApplicationService) ResubmitApplication(id, applicantID string, keptFileIDs []string, newFiles []*multipart.FileHeader) (*model.ApplicationDetail, error) {
	ctx := context.Background()
	cleanup := func(success bool) {}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var application model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, applicant_id, status, version").
			Where("id = ?", id).
			First(&application).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return fmt.Errorf("failed to query application %s: %w", id, err)
		}

		if application.ApplicantID != applicantID {
			return ErrApplicationPermissionDenied
		}
		if application.Status != model.RETURNED {
			return ErrApplicationNotEditable
		}

		newAuditFiles, syncCleanup, err := NewFileService().SyncApplicationFiles(ctx, tx, id, applicantID, keptFileIDs, newFiles)
		if err != nil {
			return err
		}
		cleanup = syncCleanup // 仅在失败时清理新上传的文件

		if err := tx.Model(&model.Application{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"status":  model.PENDING,
				"version": application.Version + 1,
			}).Error; err != nil {
			return fmt.Errorf("failed to resubmit application %s: %w", id, err)
		}

		audit := &model.Audit{
			Action:        model.PENDING,
			ApplicationID: id,
			Files:         newAuditFiles,
		}
		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("failed to create resubmit audit for application %s: %w", id, err)
		}

		return nil
	})
	if err != nil {
		cleanup(false)
		return nil, err
	}

	cleanup(true)

	return s.GetApplicationByID(id)
}
