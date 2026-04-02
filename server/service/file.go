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
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/uuid"
)

const (
	publicOwnerID = "system"
)

const (
	datasetOwnerType     = "datasets"
	applicationOwnerType = "applications"
	publicOwnerType      = "public_downloads"
)

type FileService struct {
}

func NewFileService() *FileService {
	return &FileService{}
}

// --- Create ---

func (s *FileService) UploadFile(ctx context.Context, tx *gorm.DB, ownerID, ownerType, creatorID, fileType string, fileHeader *multipart.FileHeader) (*model.File, string, error) {
	if fileHeader == nil {
		return nil, "", fmt.Errorf("file header cannot be empty")
	}

	limitMB := global.CFG.File.UploadLimitApplication
	if creatorID != "" {
		var user model.User
		if err := global.DB.Select("role").Where("id = ?", creatorID).First(&user).Error; err == nil {
			if user.Role > model.Common {
				limitMB = global.CFG.File.UploadLimitDataset
			}
		}
	}

	switch ownerType {
	case datasetOwnerType:
		limitMB = global.CFG.File.UploadLimitDataset
	case applicationOwnerType:
		limitMB = global.CFG.File.UploadLimitApplication
	case publicOwnerType:
		limitMB = global.CFG.File.UploadLimitDownload
	default:
		limitMB = global.CFG.File.UploadLimitApplication
	}

	if fileHeader.Size > int64(limitMB)*1024*1024 {
		return nil, "", fmt.Errorf("file too large: maximum allowed size is %d MB", limitMB)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	fileUUID := uuid.NewV4()
	ownerDir := model.GetDir(ownerType, ownerID)
	if err := os.MkdirAll(ownerDir, os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	targetPath := filepath.Clean(filepath.Join(ownerDir, fileUUID))
	tmpFile, err := os.Create(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer tmpFile.Close()

	hasher := md5.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), src)
	if err != nil {
		// 清理可能残留的文件
		tmpFile.Close()
		os.Remove(targetPath)
		return nil, "", fmt.Errorf("failed to write file to disk: %w", err)
	}

	size := uint(written)
	newFile := &model.File{
		Base: model.Base{
			ID: fileUUID,
		},
		Name:      fileHeader.Filename,
		MD5:       hex.EncodeToString(hasher.Sum(nil)),
		Size:      size,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		CreatorID: creatorID,
		Type:      fileType,
	}

	db := global.DB
	if tx != nil {
		db = tx
	}

	if err := db.WithContext(ctx).Create(newFile).Error; err != nil {
		_ = os.Remove(targetPath)
		return nil, "", fmt.Errorf("failed to save file metadata: %w", err)
	}

	return newFile, targetPath, nil
}

func (s *FileService) CreateDatasetFile(datasetID string, creatorID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}

	newFile, _, err := s.UploadFile(ctx, nil, datasetID, datasetOwnerType, creatorID, "", fileHeader)
	if err != nil {
		return nil, err
	}
	return newFile, nil
}

func (s *FileService) CreateApplicationFile(applicationID string, creatorID string, fileType string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	if err := s.ensureApplicationExists(ctx, applicationID); err != nil {
		return nil, err
	}

	newFile, _, err := s.UploadFile(ctx, nil, applicationID, applicationOwnerType, creatorID, fileType, fileHeader)
	if err != nil {
		return nil, err
	}

	// 自动关联到最近一次提交记录 (修复初次提交时 audit_files 为空的问题)
	var latestAudit model.Audit
	if err := global.DB.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at DESC").
		First(&latestAudit).Error; err == nil {

		if latestAudit.Action == model.DRAFT {
			if err := global.DB.WithContext(ctx).Model(&latestAudit).Association("Files").Append(newFile); err != nil {
				logger.Errorf("failed to associate file %s with audit %s: %v", newFile.ID, latestAudit.ID, err)
			}
		}
	}

	return newFile, nil
}

func (s *FileService) CreatePublicFile(creatorID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	newFile, _, err := s.UploadFile(ctx, nil, publicOwnerID, publicOwnerType, creatorID, "", fileHeader)
	if err != nil {
		return nil, err
	}

	return newFile, nil
}

// --- Get ---

func (s *FileService) ListFiles(ctx context.Context, ownerID, ownerType string) ([]model.File, error) {
	var files []model.File
	err := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", ownerID, ownerType).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

// PublicFileListQuery 前端表格的筛选/排序/分页参数
type PublicFileListQuery struct {
	Page      int
	PageSize  int
	SortBy    string // 前端列 prop，如 "name", "size", "createdAt"
	Order     string // "asc" 或 "desc"
	Name      string // 模糊搜索文件名
	DateStart string // "YYYY-MM-DD"
	DateEnd   string // "YYYY-MM-DD"
}

// publicFileSortWhitelist 将前端 el-table prop 映射到安全的数据库列
var publicFileSortWhitelist = map[string]string{
	"name":      "files.name",
	"size":      "files.size",
	"createdAt": "files.created_at",
}

func (s *FileService) ListFilesPaginated(q PublicFileListQuery) ([]model.File, int64, error) {
	ctx := context.Background()
	var files []model.File
	var total int64

	baseQuery := global.DB.WithContext(ctx).
		Model(&model.File{}).
		Where("owner_id = ? AND owner_type = ?", publicOwnerID, publicOwnerType)
	// ── 文件名搜索 ──
	if q.Name != "" {
		kw := "%" + q.Name + "%"
		baseQuery = baseQuery.Where("files.name LIKE ?", kw)
	}
	// ── 日期范围（基于 created_at）──
	loc := time.Local
	if q.DateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateStart, loc); err == nil {
			baseQuery = baseQuery.Where("files.created_at >= ?", t)
		} else {
			logger.Errorf("Failed to parse DateStart. Value: '%s', Error: %v", q.DateStart, err)
		}
	}
	if q.DateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateEnd, loc); err == nil {
			baseQuery = baseQuery.Where("files.created_at < ?", t.AddDate(0, 0, 1))
		} else {
			logger.Errorf("Failed to parse DateEnd. Value: '%s', Error: %v", q.DateEnd, err)
		}
	}
	// ── 计数 ──
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count public files: %w", err)
	}
	// ── 排序（白名单校验）──
	orderClause := "files.created_at DESC"
	if col, ok := publicFileSortWhitelist[q.SortBy]; ok {
		dir := "ASC"
		if q.Order == "desc" {
			dir = "DESC"
		}
		orderClause = col + " " + dir
	}
	// ── 分页查询 ──
	offset := (q.Page - 1) * q.PageSize
	err := baseQuery.
		Order(orderClause).
		Offset(offset).
		Limit(q.PageSize).
		Find(&files).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query public files: %w", err)
	}

	return files, total, err
}

func (s *FileService) GetFile(ctx context.Context, fileID, ownerID, ownerType string) (*model.File, error) {
	var file model.File
	err := global.DB.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, ownerID, ownerType).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileService) GetFilesByDataset(datasetID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}
	return s.ListFiles(ctx, datasetID, datasetOwnerType)
}

func (s *FileService) GetDatasetFile(datasetID, fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, datasetID, datasetOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query dataset file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetFilesByApplication(applicationID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureApplicationExists(ctx, applicationID); err != nil {
		return nil, err
	}
	return s.ListFiles(ctx, applicationID, applicationOwnerType)
}

func (s *FileService) GetApplicationFile(applicationID, fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, applicationID, applicationOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query application file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetPublicFiles() ([]model.File, error) {
	ctx := context.Background()
	return s.ListFiles(ctx, publicOwnerID, publicOwnerType)
}

func (s *FileService) GetPublicFilesPaginated(q PublicFileListQuery) ([]model.File, int64, error) {
	return s.ListFilesPaginated(q)
}

func (s *FileService) GetPublicFile(fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, publicOwnerID, publicOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query public file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetPublicTemplateFiles() ([]model.File, string, error) {
	setting, err := NewSettingService().GetSetting()
	if err != nil {
		return nil, "", err
	}
	if setting.TemplateFile == "" {
		return []model.File{}, setting.TemplateDescription, nil
	}

	ids := strings.Split(setting.TemplateFile, ",")
	var targetIDs []string
	for _, id := range ids {
		if t := strings.TrimSpace(id); t != "" {
			targetIDs = append(targetIDs, t)
		}
	}

	var files []model.File
	err = global.DB.WithContext(context.Background()).
		Where("id IN ? AND owner_id = ? AND owner_type = ?", targetIDs, publicOwnerID, publicOwnerType).
		Find(&files).Error

	return files, setting.TemplateDescription, err
}

// --- Delete ---

func (s *FileService) DeleteFile(ctx context.Context, fileID, ownerID, ownerType string) error {
	file, err := s.GetFile(ctx, fileID, ownerID, ownerType)
	if err != nil {
		return err
	}

	res := global.DB.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, ownerID, ownerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete file record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	removePhysicalFile(file.GetPath())
	cleanupOwnerDirectory(ownerType, ownerID)
	return nil
}

func (s *FileService) DeleteDatasetFile(datasetID, fileID string) error {
	return s.DeleteFile(context.Background(), fileID, datasetID, datasetOwnerType)
}

func (s *FileService) DeleteFilesByDataset(datasetID string) error {
	ctx := context.Background()
	files, err := s.ListFiles(ctx, datasetID, datasetOwnerType)
	if err != nil {
		return fmt.Errorf("failed to query files for dataset %s: %w", datasetID, err)
	}

	for _, file := range files {
		removePhysicalFile(file.GetPath())
	}

	res := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", datasetID, datasetOwnerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete files for dataset %s: %w", datasetID, res.Error)
	}

	if err := os.RemoveAll(model.GetDir(datasetOwnerType, datasetID)); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove dataset directory %s: %v", datasetID, err)
	}
	return nil
}

func (s *FileService) DeleteFilesByApplication(applicationID string) error {
	ctx := context.Background()
	files, err := s.ListFiles(ctx, applicationID, applicationOwnerType)
	if err != nil {
		return fmt.Errorf("failed to query files for application %s: %w", applicationID, err)
	}

	for _, file := range files {
		removePhysicalFile(file.GetPath())
	}

	res := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", applicationID, applicationOwnerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete files for application %s: %w", applicationID, res.Error)
	}

	if err := os.RemoveAll(model.GetDir(applicationOwnerType, applicationID)); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove application directory %s: %v", applicationID, err)
	}
	return nil
}

func (s *FileService) DeleteFilesByAudit(auditID string) error {
	ctx := context.Background()
	var audit model.Audit
	if err := global.DB.WithContext(ctx).Preload("Files").First(&audit, "id = ?", auditID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Audit not found, nothing to do
		}
		return fmt.Errorf("failed to find audit %s: %w", auditID, err)
	}

	if len(audit.Files) == 0 {
		return nil
	}

	filesToDelete := audit.Files
	fileIDs := make([]string, len(filesToDelete))
	for i, f := range filesToDelete {
		fileIDs[i] = f.ID
	}

	// Use a transaction to ensure atomicity of DB operations
	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Clear associations in the join table
		if err := tx.Model(&audit).Association("Files").Clear(); err != nil {
			return fmt.Errorf("failed to clear file associations for audit %s: %w", auditID, err)
		}

		// 2. Delete the file records themselves
		if len(fileIDs) > 0 {
			if err := tx.Where("id IN ?", fileIDs).Delete(&model.File{}).Error; err != nil {
				return fmt.Errorf("failed to delete file records for audit %s: %w", auditID, err)
			}
		}

		return nil
	})

	if err != nil {
		return err // DB operations failed
	}

	// 3. Delete physical files after DB transaction is successful
	for _, f := range filesToDelete {
		removePhysicalFile(f.GetPath())
	}

	return nil
}

// SyncApplicationFiles 同步申请的附件（用于重新提交）
// 1. 检查保留的文件是否存在
// 2. 上传新文件
// 3. 返回最终的文件列表（用于关联到 Audit）
func (s *FileService) SyncApplicationFiles(ctx context.Context, tx *gorm.DB, applicationID, creatorID string, keptFileIDs []string, newFiles []*multipart.FileHeader, newFileTypes []string) ([]model.File, func(bool), error) {
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
		return nil, cleanup, fmt.Errorf("failed to query attachments for application %s: %w", applicationID, err)
	}

	// 收集被保留的文件对象
	var finalFiles []model.File
	for _, file := range existingFiles {
		if _, ok := kept[file.ID]; ok {
			finalFiles = append(finalFiles, file)
		}
	}
	// 注意：不再物理删除未保留的文件，因为它们属于历史版本

	var newFilePaths []string
	for i, header := range newFiles {
		if header == nil {
			continue
		}
		fileType := ""
		if i < len(newFileTypes) {
			fileType = newFileTypes[i]
		}
		fileRecord, storedPath, err := s.UploadFile(ctx, tx, applicationID, applicationOwnerType, creatorID, fileType, header)
		if err != nil {
			return nil, cleanup, err
		}
		newFilePaths = append(newFilePaths, storedPath)
		if fileRecord != nil {
			finalFiles = append(finalFiles, *fileRecord)
		}
	}

	cleanup = func(success bool) {
		if !success {
			// 只有在事务失败时，才清理刚刚上传的新文件
			for _, path := range newFilePaths {
				removePhysicalFile(path)
			}
		}
	}

	return finalFiles, cleanup, nil
}

// GetLatestApplicationFiles 获取申请当前版本（最新一次提交）的文件列表
func (s *FileService) GetLatestApplicationFiles(applicationID string) ([]model.File, error) {
	ctx := context.Background()
	var latestAudit model.Audit

	// 1. 查找最近一次 Submit 或 ReSubmit 的 Audit
	err := global.DB.WithContext(ctx).
		Where("application_id = ? AND action = ?", applicationID, model.PENDING).
		Order("created_at DESC").
		Preload("Files").
		First(&latestAudit).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果找不到提交记录（可能是旧数据或刚创建未关联），回退到查询所有归属该 App 的文件
			// 这是一个兼容性回退，防止初次提交时文件未关联到 Audit 导致列表为空
			var fallbackFiles []model.File
			if err := global.DB.WithContext(ctx).
				Where("owner_id = ? AND owner_type = ?", applicationID, applicationOwnerType).
				Find(&fallbackFiles).Error; err != nil {
				return nil, err
			}
			return fallbackFiles, nil
		}
		return nil, fmt.Errorf("failed to query latest audit: %w", err)
	}

	// 2. 如果 Audit 关联了文件，直接返回
	if len(latestAudit.Files) > 0 {
		return latestAudit.Files, nil
	}

	// 3. 如果是初次提交且没有关联文件（可能是因为文件是在创建 App 后单独上传的），
	// 尝试查找所有未被其他 Audit 关联的“孤儿”文件？或者直接回退到所有文件。
	// 简单起见，如果 Audit 没文件，回退到 Owner 查找。
	var fallbackFiles []model.File
	if err := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", applicationID, applicationOwnerType).
		Find(&fallbackFiles).Error; err != nil {
		return nil, err
	}
	return fallbackFiles, nil
}

func (s *FileService) DeleteApplicationFile(applicationID, fileID string) error {
	return s.DeleteFile(context.Background(), fileID, applicationID, applicationOwnerType)
}

func (s *FileService) DeletePublicFile(fileID string) error {
	return s.DeleteFile(context.Background(), fileID, publicOwnerID, publicOwnerType)
}

// --- Other ---

func (s *FileService) ensureDatasetExists(ctx context.Context, datasetID string) error {
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

func (s *FileService) ensureApplicationExists(ctx context.Context, applicationID string) error {
	_, err := gorm.G[model.Application](global.DB).
		Select("id").
		Where("id = ?", applicationID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return fmt.Errorf("failed to query application %s: %w", applicationID, err)
	}
	return nil
}

func removePhysicalFile(filePath string) {
	targetPath := filePath
	if !filepath.IsAbs(targetPath) {
		if abs, err := filepath.Abs(filePath); err == nil {
			targetPath = abs
		}
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove file %s: %v", targetPath, err)
	}
}

func cleanupOwnerDirectory(ownerType, ownerID string) {
	dir := model.GetDir(ownerType, ownerID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("failed to inspect directory %s: %v", dir, err)
		}
		return
	}
	if len(entries) == 0 {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			logger.Warnf("failed to remove directory %s: %v", dir, err)
		}
	}
}
