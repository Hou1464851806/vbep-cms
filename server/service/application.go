package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"server/global"
	"server/model"
	"server/pkg/uuid"
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

	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(app).Error; err != nil {
			return fmt.Errorf("failed to create application: %w", err)
		}

		now := time.Now()
		audit := &model.Audit{
			Action:        model.ActionSubmit,
			ApplicationID: app.ID,
			AuditedAt:     &now,
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

func (s *ApplicationService) ListApplications(applicantID *string) ([]model.ApplicationListItem, error) {
	ctx := context.Background()
	var apps []model.Application

	db := global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Order("created_at DESC").
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		}).
		Preload("Audits", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Preload("Auditor", func(db *gorm.DB) *gorm.DB {
				return db.Select("id, nick_name, email")
			})
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

func (s *ApplicationService) GetApplicationByID(id string) (*model.ApplicationListItem, error) {
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
			return db.Order("created_at DESC").Preload("Auditor", func(db *gorm.DB) *gorm.DB {
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

	item := app.ToListItem()
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

func (s *ApplicationService) ApproveApplication(id, auditorID, result string) (*model.ApplicationListItem, error) {
	return s.updateApplicationStatus(id, auditorID, result, model.APPROVED, model.ActionApprove)
}

func (s *ApplicationService) RejectApplication(id, auditorID, result string) (*model.ApplicationListItem, error) {
	return s.updateApplicationStatus(id, auditorID, result, model.REJECTED, model.ActionReject)
}

func (s *ApplicationService) ReturnApplication(id, auditorID, result string) (*model.ApplicationListItem, error) {
	return s.updateApplicationStatus(id, auditorID, result, model.RETURNED, model.ActionReturn)
}

func (s *ApplicationService) updateApplicationStatus(id, auditorID, result string, status model.ApplicationStatus, action model.ActionType) (*model.ApplicationListItem, error) {
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

		now := time.Now()
		audit := &model.Audit{
			Action:        action,
			ApplicationID: id,
			AuditedAt:     &now,
		}
		if auditorID != "" {
			audit.AuditorID = &auditorID
		}
		if result != "" {
			audit.AuditorResult = &result
		}

		if err := tx.Create(audit).Error; err != nil {
			return fmt.Errorf("failed to create audit for application %s: %w", id, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetApplicationByID(id)
}

func (s *ApplicationService) GetApplicationAudits(applicationID string) ([]model.AuditListItem, error) {
	ctx := context.Background()
	var audits []model.Audit

	err := global.DB.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at ASC").
		Preload("Auditor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Application", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, applicant_id").
				Preload("Applicant", func(appDB *gorm.DB) *gorm.DB {
					return appDB.Select("id, nick_name, email")
				})
		}).
		Find(&audits).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query audits for application %s: %w", applicationID, err)
	}

	result := make([]model.AuditListItem, len(audits))
	for i := range audits {
		result[i] = audits[i].ToListItem()
	}

	return result, nil
}

func (s *ApplicationService) ResubmitApplication(id, applicantID, reason string, keptFileIDs []string, newFiles []*multipart.FileHeader) (*model.ApplicationListItem, error) {
	ctx := context.Background()
	cleanup := func(success bool) {}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var application model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, applicant_id, status").
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

		syncCleanup, err := s.syncApplicationAttachmentsTx(tx, id, applicantID, keptFileIDs, newFiles)
		if err != nil {
			return err
		}
		cleanup = syncCleanup

		if err := tx.Model(&model.Application{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"status": model.PENDING,
				"reason": reason,
			}).Error; err != nil {
			return fmt.Errorf("failed to resubmit application %s: %w", id, err)
		}

		now := time.Now()
		audit := &model.Audit{
			Action:        model.ActionReSubmit,
			ApplicationID: id,
			AuditedAt:     &now,
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

func (s *ApplicationService) syncApplicationAttachmentsTx(tx *gorm.DB, applicationID, creatorID string, keptFileIDs []string, newFiles []*multipart.FileHeader) (func(bool), error) {
	cleanup := func(bool) {}
	kept := make(map[string]struct{})
	for _, id := range keptFileIDs {
		if id == "" {
			continue
		}
		kept[id] = struct{}{}
	}

	var existingFiles []model.File
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_id = ? AND owner_type = ?", applicationID, applicationOwnerType).
		Find(&existingFiles).Error; err != nil {
		return cleanup, fmt.Errorf("failed to query attachments for application %s: %w", applicationID, err)
	}

	var filesToDelete []model.File
	for _, file := range existingFiles {
		if _, ok := kept[file.ID]; ok {
			continue
		}
		filesToDelete = append(filesToDelete, file)
	}

	for _, file := range filesToDelete {
		if err := tx.Where("id = ? AND owner_id = ? AND owner_type = ?", file.ID, applicationID, applicationOwnerType).
			Delete(&model.File{}).Error; err != nil {
			return cleanup, fmt.Errorf("failed to delete attachment %s: %w", file.ID, err)
		}
	}

	var newFilePaths []string
	for _, header := range newFiles {
		if header == nil {
			continue
		}
		_, storedPath, err := s.saveApplicationFileWithTx(tx, applicationID, creatorID, header)
		if err != nil {
			return cleanup, err
		}
		newFilePaths = append(newFilePaths, storedPath)
	}

	cleanup = func(success bool) {
		if success {
			for _, file := range filesToDelete {
				removePhysicalFile(file.GetPath())
			}
			if len(filesToDelete) > 0 {
				cleanupApplicationDirectory(applicationID)
			}
		} else {
			for _, path := range newFilePaths {
				removePhysicalFile(path)
			}
		}
	}

	return cleanup, nil
}

func (s *ApplicationService) saveApplicationFileWithTx(tx *gorm.DB, applicationID, creatorID string, fileHeader *multipart.FileHeader) (*model.File, string, error) {
	if fileHeader == nil {
		return nil, "", fmt.Errorf("file header cannot be empty")
	}
	if fileHeader.Size > maxUploadSize {
		return nil, "", fmt.Errorf("file too large: maximum allowed size is %d MB", maxUploadSize/(1024*1024))
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	fileUUID := uuid.NewV4()
	appDir := applicationDirPath(applicationID)
	if err := os.MkdirAll(appDir, os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create application upload directory: %w", err)
	}

	targetPath := filepath.Clean(filepath.Join(appDir, fileUUID))
	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	hasher := md5.New()
	written, err := io.Copy(io.MultiWriter(dst, hasher), src)
	if err != nil {
		removePhysicalFile(targetPath)
		return nil, "", fmt.Errorf("failed to write file to disk: %w", err)
	}

	newFile := &model.File{
		Base: model.Base{
			ID: fileUUID,
		},
		Name:      fileHeader.Filename,
		MD5:       hex.EncodeToString(hasher.Sum(nil)),
		Size:      uint(written),
		OwnerID:   applicationID,
		OwnerType: applicationOwnerType,
		CreatorID: creatorID,
	}

	if err := tx.Create(newFile).Error; err != nil {
		removePhysicalFile(targetPath)
		return nil, "", fmt.Errorf("failed to save file metadata: %w", err)
	}

	return newFile, targetPath, nil
}
