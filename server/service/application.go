package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"server/internal/tasks"
	"server/internal/templates"
	"server/logger"
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
	ErrPendingApplicationExists    = errors.New("pending application exists")
)

type ApplicationService struct {
	mailService    *MailService
	settingService *SettingService
}

func NewApplicationService() *ApplicationService {
	return &ApplicationService{}
}

func NewApplicationServiceWithOther(mailService *MailService, settingService *SettingService) *ApplicationService {
	return &ApplicationService{
		mailService:    mailService,
		settingService: settingService,
	}
}

func (s *ApplicationService) CreateApplication(app *model.Application) error {
	ctx := context.Background()

	// 检查用户是否已有处于 DRAFT, PENDING 或 RETURNED 状态的申请
	var count int64
	if err := global.DB.WithContext(ctx).Model(&model.Application{}).
		Where("applicant_id = ? AND dataset_id = ? AND status IN ?", app.ApplicantID, app.DatasetID, []model.ApplicationStatus{model.DRAFT, model.PENDING, model.RETURNED}).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check pending applications: %w", err)
	}
	if count > 0 {
		return ErrPendingApplicationExists
	}

	if err := s.ensureDatasetExists(ctx, app.DatasetID); err != nil {
		return err
	}

	app.Version = 0 // 初始创建为草稿状态
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(app).Error; err != nil {
			return fmt.Errorf("failed to create application: %w", err)
		}

		audit := &model.Audit{
			Action:        model.DRAFT,
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
		Select("id, applicant_id, status").
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

	if app.Status != model.DRAFT {
		return ErrApplicationNotEditable // 只能删除草稿
	}

	// 查找关联的 Audits
	var audits []model.Audit
	if err := global.DB.WithContext(ctx).Where("application_id = ?", id).Find(&audits).Error; err != nil {
		return fmt.Errorf("failed to find audits: %w", err)
	}

	fileService := NewFileService()
	for _, audit := range audits {
		if err := fileService.DeleteFilesByAudit(audit.ID); err != nil {
			return fmt.Errorf("failed to delete files for audit %s: %w", audit.ID, err)
		}
	}

	// 再次调用以清理可能存在的孤儿文件（未关联到 Audit 的文件）以及移除申请文件夹
	// 这是为了确保文件系统的整洁，利用 FileService 封装好的目录清理逻辑
	if err := fileService.DeleteFilesByApplication(id); err != nil {
		return fmt.Errorf("failed to cleanup application files: %w", err)
	}

	if err := global.DB.WithContext(ctx).Where("application_id = ?", id).Delete(&model.Audit{}).Error; err != nil {
		return fmt.Errorf("failed to delete audits: %w", err)
	}

	// 删除 Application (级联删除 Audit)
	if err := global.DB.WithContext(ctx).Delete(&app).Error; err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}

func (s *ApplicationService) SubmitDraftApplication(id, applicantID, affiliation, position string) error {
	ctx := context.Background()

	var datasetTitle string

	// 申请提交事务
	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Dataset").
			Where("id = ?", id).
			First(&app).Error; err != nil {
			return err
		}

		if app.ApplicantID != applicantID {
			return ErrApplicationPermissionDenied
		}
		if app.Status != model.DRAFT {
			return ErrApplicationAlreadyProcessed
		}

		// 更新状态为 PENDING，版本号为1，正式提交
		// 同时更新关联的 Audit 时间，使其反映用户点击“提交”的时间
		if err := tx.Model(&app).Updates(map[string]interface{}{
			"status":      model.PENDING,
			"version":     1,
			"affiliation": affiliation,
			"position":    position,
		}).Error; err != nil {
			return err
		}

		now := time.Now()
		if err := tx.Model(&model.Audit{}).
			Where("application_id = ? AND action = ?", id, model.DRAFT).
			Updates(map[string]interface{}{"created_at": now, "action": model.PENDING}).Error; err != nil {
			return err
		}

		datasetTitle = app.Dataset.Title

		return nil
	})
	// 事务执行失败则不继续发送邮件
	if err != nil {
		return err
	}

	subject := "数据集有新的申请"
	url := fmt.Sprintf("%s/home/applications?id=%s", global.CFG.Web.Url, id)
	s.sendNewAppNotificationEmail(subject, datasetTitle, url)

	return nil
}

func (s *ApplicationService) UpdateDraftApplication(id, applicantID, affiliation, position string) error {
	ctx := context.Background()
	var app model.Application

	if err := global.DB.WithContext(ctx).
		Select("id, applicant_id, status").
		Where("id = ?", id).
		First(&app).Error; err != nil {
		return err
	}

	if app.ApplicantID != applicantID {
		return ErrApplicationPermissionDenied
	}

	if app.Status != model.DRAFT && app.Status != model.RETURNED {
		return ErrApplicationNotEditable
	}

	if err := global.DB.WithContext(ctx).Model(&app).Updates(map[string]interface{}{
		"affiliation": affiliation,
		"position":    position,
	}).Error; err != nil {
		return fmt.Errorf("failed to update application draft: %w", err)
	}

	return nil
}

// ApplicationListQuery 前端表格的筛选/排序/分页参数
type ApplicationListQuery struct {
	ApplicantID   *string
	ExcludeDrafts bool
	Page          int
	PageSize      int
	SortBy        string // 前端列 prop，如 "auditedAt", "status"
	Order         string // "asc" 或 "desc"
	Status        string // 状态筛选
	Keyword       string // 模糊搜索（申请人昵称 / 数据集标题）
	DateStart     string // "YYYY-MM-DD"
	DateEnd       string // "YYYY-MM-DD"
}

// sortColumnWhitelist 将前端 el-table prop 映射到安全的数据库列
// 只允许白名单内的列排序，防止 SQL 注入
var sortColumnWhitelist = map[string]string{
	"applicantName": "users.nick_name",
	"datasetTitle":  "datasets.title",
	"auditedAt":     "applications.updated_at",
}

// validStatuses 合法的状态值
var validStatuses = map[string]bool{
	"draft": true, "pending": true, "approved": true,
	"rejected": true, "returned": true,
}

func (s *ApplicationService) ListApplications(q ApplicationListQuery) ([]model.ApplicationListItem, int64, error) {
	ctx := context.Background()
	var apps []model.Application
	var total int64

	// 基础查询 + LEFT JOIN（用于关键词搜索 & 关联列排序）
	baseQuery := global.DB.WithContext(ctx).
		Model(&model.Application{}).
		Joins("LEFT JOIN users ON users.id = applications.applicant_id").
		Joins("LEFT JOIN datasets ON datasets.id = applications.dataset_id")
	// ── 权限过滤 ──
	if q.ApplicantID != nil {
		baseQuery = baseQuery.Where("applications.applicant_id = ?", *q.ApplicantID)
	}
	if q.ExcludeDrafts {
		baseQuery = baseQuery.Where("applications.status != ?", model.DRAFT)
	}
	// ── 状态筛选 ──
	if q.Status != "" && validStatuses[q.Status] {
		status, _ := model.ParseApplicationStatus(q.Status)
		baseQuery = baseQuery.Where("applications.status = ?", status)
	}
	// ── 关键词搜索（申请人昵称 / 数据集标题）──
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		baseQuery = baseQuery.Where(
			"users.nick_name LIKE ? OR datasets.title LIKE ?", kw, kw,
		)
	}
	// ── 日期范围（基于 updated_at）──
	loc := time.Local
	if q.DateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateStart, loc); err == nil {
			baseQuery = baseQuery.Where("applications.updated_at >= ?", t)
		} else {
			logger.Errorf("Failed to parse DateStart. Value: '%s', Error: %v", q.DateStart, err)
		}
	}
	if q.DateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", q.DateEnd, loc); err == nil {
			// 包含当天整天：上界取次日零点
			baseQuery = baseQuery.Where("applications.updated_at < ?", t.AddDate(0, 0, 1))
		} else {
			logger.Errorf("Failed to parse DateEnd. Value: '%s', Error: %v", q.DateEnd, err)
		}
	}
	// ── 计数 ──
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count applications: %w", err)
	}
	// ── 排序（白名单校验）──
	orderClause := "applications.updated_at DESC" // 默认
	if col, ok := sortColumnWhitelist[q.SortBy]; ok {
		dir := "ASC"
		if q.Order == "desc" {
			dir = "DESC"
		}
		orderClause = col + " " + dir
	}
	// ── 分页查询 ──
	offset := (q.Page - 1) * q.PageSize
	err := baseQuery.
		Select("applications.*"). // 避免 JOIN 列干扰 GORM 扫描
		Order(orderClause).
		Offset(offset).
		Limit(q.PageSize).
		Preload("Applicant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nick_name, email")
		}).
		Preload("Dataset", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		}).
		Find(&apps).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query applications: %w", err)
	}

	// 模型转换 (Model -> DTO)
	result := make([]model.ApplicationListItem, len(apps))
	for i, app := range apps {
		result[i] = app.ToListItem()
	}

	return result, total, nil
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
func (s *ApplicationService) ProcessApplicationAudit(id, operatorID, result string, status model.ApplicationStatus, duration int) (*model.ApplicationDetail, error) {
	// Validate the incoming status to ensure it's an audit action
	ctx := context.Background()
	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, status, dataset_id, applicant_id").
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

		appUpdates := map[string]interface{}{"status": status}
		var expireAt *time.Time
		if status == model.APPROVED && duration > 0 {
			t := time.Now().AddDate(0, duration, 0)
			expireAt = &t
			appUpdates["expire_at"] = expireAt
		}

		if err := tx.Model(&model.Application{}).
			Where("id = ?", id).
			Updates(appUpdates).Error; err != nil {
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
			"expire_at":   expireAt,
		}

		if err := tx.Model(&lastAudit).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update audit for application %s: %w", id, err)
		}

		if status == model.APPROVED {
			var license model.License
			if err := tx.Where("application_id = ?", id).First(&license).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					newLicense := model.License{
						DatasetID:     existing.DatasetID,
						ApplicantID:   existing.ApplicantID,
						ApplicationID: id,
						Status:        model.LicenseNormal,
						ExpireAt:      expireAt,
					}
					if err := tx.Create(&newLicense).Error; err != nil {
						return fmt.Errorf("failed to create license: %w", err)
					}
				} else {
					return fmt.Errorf("failed to query license: %w", err)
				}
			} else {
				if err := tx.Model(&license).Updates(map[string]interface{}{
					"status":        model.LicenseNormal,
					"expire_at":     expireAt,
					"revoke_reason": "",
				}).Error; err != nil {
					return fmt.Errorf("failed to update license: %w", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	appDetail, err := s.GetApplicationByID(id)
	if err != nil {
		return nil, err
	}

	userEmail := appDetail.Applicant.Email
	if userEmail == "" {
		logger.Warnf("user email is empty")
		return appDetail, nil
	}

	var statusStr string
	switch status {
	case model.APPROVED:
		statusStr = "已经通过"
	case model.REJECTED:
		statusStr = "已被拒绝"
	case model.RETURNED:
		statusStr = "已被退回"
	default:
		statusStr = "状态更新"
	}

	subject := fmt.Sprintf("您的申请%s", statusStr)
	datasetTitle := appDetail.Dataset.Title
	url := fmt.Sprintf("%s/home/applications?id=%s", global.CFG.Web.Url, id)
	content := templates.AppChangeNotificationContent{
		DatasetTitle: datasetTitle,
		Status:       statusStr,
		Url:          url,
	}
	err = s.mailService.SendNewMail(userEmail, subject, templates.AppChangeNotificationHtml, content, tasks.QueueDefault, 0)
	if err != nil {
		logger.Errorf("failed to send app change notification to user email %s: %v", userEmail, err)
	}

	return appDetail, nil
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

func (s *ApplicationService) ResubmitApplication(id, applicantID, affiliation, position string, keptFileIDs []string, newFiles []*multipart.FileHeader, newFileTypes []string) (*model.ApplicationDetail, error) {
	ctx := context.Background()
	cleanup := func(success bool) {}

	var datasetTitle string

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var application model.Application
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id, applicant_id, status, version, dataset_id").
			Preload("Dataset").
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

		newAuditFiles, syncCleanup, err := NewFileService().SyncApplicationFiles(ctx, tx, id, applicantID, keptFileIDs, newFiles, newFileTypes)
		if err != nil {
			return err
		}
		cleanup = syncCleanup // 仅在失败时清理新上传的文件

		if err := tx.Model(&model.Application{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"status":      model.PENDING,
				"version":     application.Version + 1,
				"affiliation": affiliation,
				"position":    position,
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

		datasetTitle = application.Dataset.Title

		return nil
	})
	if err != nil {
		cleanup(false)
		return nil, err
	}

	cleanup(true)

	subject := "数据集有新的申请"
	url := fmt.Sprintf("%s/home/applications?id=%s", global.CFG.Web.Url, id)
	s.sendNewAppNotificationEmail(subject, datasetTitle, url)

	return s.GetApplicationByID(id)
}

func (s *ApplicationService) sendNewAppNotificationEmail(subject string, datasetTitle string, url string) {
	setting, err := s.settingService.GetSetting()
	if err != nil {
		logger.Errorf("failed to get setting: %v", err)
		return
	}
	publicEmail := setting.PublicEmail
	if publicEmail == "" {
		logger.Warnf("public email is empty")
		return
	}
	content := templates.NewAppNotificationContent{
		DatasetTitle: datasetTitle,
		Url:          url,
	}
	err = s.mailService.SendNewMail(publicEmail, subject, templates.NewAppNotificationHtml, content, tasks.QueueDefault, 0)
	if err != nil {
		logger.Errorf("failed to send new application notification to public mail: %v", err)
		return
	}
}
