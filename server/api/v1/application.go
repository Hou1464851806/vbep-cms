package v1

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"server/global"
	"server/logger"
	"server/model"
	"server/service"
)

var (
	errUserNotAuthenticated = errors.New("user not authenticated")
	errInvalidUserID        = errors.New("invalid user id")
	errInvalidUser          = errors.New("invalid user")
	errPermissionDenied     = errors.New("permission denied")
)

type ApplicationApi struct {
	service *service.ApplicationService
}

func NewApplicationApi() *ApplicationApi {
	return &ApplicationApi{service: service.NewApplicationService()}
}

type createApplicationParam struct {
	DatasetID string `json:"datasetId" binding:"required"`
}

type auditApplicationParam struct {
	Result string `json:"result" binding:"required"`
}

type resubmitApplicationParam struct {
	Reason string `json:"reason" binding:"required"`
}

func (a *ApplicationApi) Create(ctx *gin.Context) {
	var param createApplicationParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUserID) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	app := &model.Application{
		DatasetID:   param.DatasetID,
		ApplicantID: userIDStr,
		Status:      model.PENDING,
		Reason:      "申请资料已提交",
	}

	if err := a.service.CreateApplication(app); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "dataset not found"})
			return
		}
		logger.Errorf("failed to create application: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create application"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"id":      app.ID,
	})
}

func (a *ApplicationApi) GetAll(ctx *gin.Context) {
	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUser) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	// 2. 决定查询范围
	var filterID *string
	// 假设 model.Common == 0, model.Admin == 1.
	// 如果角色是 Common (0)，则传入当前用户ID进行筛选
	if currentUser.Role == model.Common {
		filterID = &currentUser.ID
	}
	// 如果是 Admin 或更高，filterID 保持为 nil (查询所有)

	// 3. 调用 Service
	applications, err := a.service.ListApplications(filterID)
	if err != nil {
		logger.Errorf("failed to list applications: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load applications"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":      "success",
		"applications": applications,
	})
}

func (a *ApplicationApi) GetOne(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUser) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	// 2. 查询 Application
	application, err := a.service.GetApplicationByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return
		}
		logger.Errorf("failed to get application %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load application"})
		return
	}

	// 3. 鉴权逻辑：如果是普通用户，且该申请的申请人不是当前用户 -> 403 Forbidden
	if currentUser.Role == model.Common && application.Applicant.ID != currentUser.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied: you do not own this application"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "success",
		"application": *application,
	})
}

func (a *ApplicationApi) GetAudits(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUser) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	if err := ensureApplicationReadable(ctx, currentUser, id); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
		case errors.Is(err, errPermissionDenied):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied"})
		default:
			logger.Errorf("failed to verify application %s: %v", id, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load application audits"})
		}
		return
	}

	audits, err := a.service.GetApplicationAudits(id)
	if err != nil {
		logger.Errorf("failed to load audits for application %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load application audits"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"audits":  audits,
	})
}

func (a *ApplicationApi) Approve(ctx *gin.Context) {
	a.handleAuditAction(ctx, func(id, auditorID, result string) (*model.ApplicationListItem, error) {
		return a.service.ApproveApplication(id, auditorID, result)
	}, "approve")
}

func (a *ApplicationApi) Reject(ctx *gin.Context) {
	a.handleAuditAction(ctx, func(id, auditorID, result string) (*model.ApplicationListItem, error) {
		return a.service.RejectApplication(id, auditorID, result)
	}, "reject")
}

func (a *ApplicationApi) Return(ctx *gin.Context) {
	a.handleAuditAction(ctx, func(id, auditorID, result string) (*model.ApplicationListItem, error) {
		return a.service.ReturnApplication(id, auditorID, result)
	}, "return")
}

func (a *ApplicationApi) Resubmit(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	var (
		reason       string
		keptFileIDs  []string
		newFileHeads []*multipart.FileHeader
	)

	if strings.HasPrefix(ctx.ContentType(), "multipart/form-data") {
		reason = strings.TrimSpace(ctx.PostForm("reason"))
		keptRaw := ctx.PostForm("keptFileIds")
		if keptRaw != "" {
			if err := json.Unmarshal([]byte(keptRaw), &keptFileIDs); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid keptFileIds"})
				return
			}
		}
		form, err := ctx.MultipartForm()
		if err != nil && err != http.ErrNotMultipart {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid attachment payload"})
			return
		}
		if form != nil {
			if files, ok := form.File["files"]; ok {
				newFileHeads = files
			}
		}
	} else {
		var param resubmitApplicationParam
		if err := ctx.ShouldBindJSON(&param); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		reason = strings.TrimSpace(param.Reason)
	}

	if reason == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "reason is required"})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUserID) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	application, err := a.service.ResubmitApplication(id, userIDStr, reason, keptFileIDs, newFileHeads)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return
		case errors.Is(err, service.ErrApplicationPermissionDenied):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied"})
			return
		case errors.Is(err, service.ErrApplicationNotEditable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "application cannot be resubmitted"})
			return
		default:
			logger.Errorf("failed to resubmit application %s: %v", id, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to resubmit application"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "success",
		"application": *application,
	})
}

type auditActionFunc func(id, auditorID, result string) (*model.ApplicationListItem, error)

func (a *ApplicationApi) handleAuditAction(ctx *gin.Context, action auditActionFunc, actionName string) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	var param auditApplicationParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errInvalidUserID) {
			status = http.StatusInternalServerError
		}
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	application, err := action(id, userIDStr, param.Result)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return
		case errors.Is(err, service.ErrApplicationAlreadyProcessed):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "application already processed"})
			return
		default:
			logger.Errorf("failed to %s application %s: %v", actionName, id, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to process application"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "success",
		"application": *application,
	})
}

func getCurrentUser(ctx *gin.Context) (*model.User, error) {
	userObj, exists := ctx.Get(global.CTXUserObj)
	if !exists {
		return nil, errUserNotAuthenticated
	}
	user, ok := userObj.(*model.User)
	if !ok {
		return nil, errInvalidUser
	}
	return user, nil
}

func getUserIDFromContext(ctx *gin.Context) (string, error) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		return "", errUserNotAuthenticated
	}
	userIDStr, ok := userID.(string)
	if !ok {
		return "", errInvalidUserID
	}
	return userIDStr, nil
}

func ensureApplicationReadable(ctx *gin.Context, user *model.User, applicationID string) error {
	var app model.Application
	err := global.DB.WithContext(ctx.Request.Context()).
		Model(&model.Application{}).
		Select("id, applicant_id").
		Where("id = ?", applicationID).
		First(&app).Error
	if err != nil {
		return err
	}
	if user.Role == model.Common && app.ApplicantID != user.ID {
		return errPermissionDenied
	}
	return nil
}
