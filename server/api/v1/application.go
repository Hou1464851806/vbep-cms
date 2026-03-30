package v1

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"server/global"
	"server/logger"
	"server/model"
	"server/service"
)

var (
	errUserNotAuthenticated = errors.New("用户未登录")
	errInvalidUserID        = errors.New("无效的用户ID")
	errInvalidUser          = errors.New("无效的用户")
	errPermissionDenied     = errors.New("权限不足")
)

type ApplicationApi struct {
	applicationService *service.ApplicationService
}

func NewApplicationApi(applicationService *service.ApplicationService) *ApplicationApi {
	return &ApplicationApi{
		applicationService: applicationService,
	}
}

type createApplicationParam struct {
	DatasetID string `json:"datasetID" binding:"required"`
}

type auditApplicationParam struct {
	Action   string `json:"action" binding:"required"` // e.g., "approved", "rejected", "returned"
	Result   string `json:"result"`
	Duration int    `json:"duration"` // Duration in months, for approval
}

type submitDraftParam struct {
	Affiliation string `json:"affiliation"`
	Position    string `json:"position"`
}

type resubmitApplicationParam struct {
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
		Status:      model.DRAFT,
	}

	if err := a.applicationService.CreateApplication(app); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "数据集不存在"})
			return
		}
		if errors.Is(err, service.ErrPendingApplicationExists) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "您在该数据集下已有正在进行的申请，请先处理现有申请"})
			return
		}
		logger.Errorf("failed to create application: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "创建申请失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"applicationID": app.ID,
	})
}

func (a *ApplicationApi) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	if err := a.applicationService.DeleteDraftApplication(id, userIDStr); err != nil {
		if errors.Is(err, service.ErrApplicationPermissionDenied) {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
			return
		}
		if errors.Is(err, service.ErrApplicationNotEditable) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "无法删除已提交的申请"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "删除申请失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *ApplicationApi) SubmitDraft(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var param submitDraftParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		// Ignore binding error as body might be empty for legacy clients, but fields will be empty strings
	}

	if err := a.applicationService.SubmitDraftApplication(id, userIDStr, param.Affiliation, param.Position); err != nil {
		if errors.Is(err, service.ErrApplicationPermissionDenied) {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
			return
		}
		if errors.Is(err, service.ErrApplicationAlreadyProcessed) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "申请已提交"})
			return
		}
		logger.Errorf("failed to submit application %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "提交申请失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *ApplicationApi) UpdateDraft(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	userIDStr, err := getUserIDFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var param submitDraftParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		// Ignore binding error
	}

	if err := a.applicationService.UpdateDraftApplication(id, userIDStr, param.Affiliation, param.Position); err != nil {
		if errors.Is(err, service.ErrApplicationPermissionDenied) {
			ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
			return
		}
		if errors.Is(err, service.ErrApplicationNotEditable) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "申请不可编辑"})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		}
		logger.Errorf("failed to update application draft %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "暂存失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
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

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 2. 决定查询范围
	var filterID *string
	excludeDrafts := false
	// 假设 model.Common == 0, model.Admin == 1.
	// 如果角色是 Common (0)，则传入当前用户ID进行筛选
	if currentUser.Role == model.Common {
		filterID = &currentUser.ID
	} else if currentUser.Role == model.Admin {
		// 普通管理员不可见草稿，超级管理员(Role > Admin)可见
		excludeDrafts = true
	}
	// 如果是 Admin 或更高，filterID 保持为 nil (查询所有)

	// 构建查询参数
	query := service.ApplicationListQuery{
		ApplicantID:   filterID,
		ExcludeDrafts: excludeDrafts,
		Page:          page,
		PageSize:      pageSize,
		SortBy:        ctx.Query("sort_by"),
		Order:         ctx.Query("order"),
		Status:        ctx.Query("status"),
		Keyword:       ctx.Query("keyword"),
		DateStart:     ctx.Query("date_start"),
		DateEnd:       ctx.Query("date_end"),
	}

	// 3. 调用 Service
	applications, total, err := a.applicationService.ListApplications(query)
	if err != nil {
		logger.Errorf("failed to list applications: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载申请列表失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"total":        total,
	})
}

func (a *ApplicationApi) GetOne(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
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
	application, err := a.applicationService.GetApplicationByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		}
		logger.Errorf("failed to get application %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载申请失败"})
		return
	}

	// 3. 鉴权逻辑：如果是普通用户，且该申请的申请人不是当前用户 -> 403 Forbidden
	if currentUser.Role == model.Common && application.Applicant.ID != currentUser.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足：您不是该申请的所有者"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"application": *application,
	})
}

func (a *ApplicationApi) GetAuditDetail(ctx *gin.Context) {
	id := ctx.Param("id")
	auditID := ctx.Param("auditID")
	if id == "" || auditID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "参数无效"})
		return
	}

	currentUser, err := getCurrentUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	if err := ensureApplicationReadable(ctx, currentUser, id); err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	detail, err := a.applicationService.GetApplicationAuditDetail(id, auditID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "审批记录不存在"})
		return
	}

	// 构造符合前端命名规范的文件列表
	var fileResps []applicationFileResponse
	if len(detail.Files) > 0 {
		fileResps = make([]applicationFileResponse, 0, len(detail.Files))
		for _, f := range detail.Files {
			fileResps = append(fileResps, applicationFileResponse{
				ID:            f.ID,
				Name:          f.Name,
				Size:          f.Size,
				ApplicationID: f.OwnerID,
				Type:          f.Type,
				CreatedAt:     f.CreatedAt,
			})
		}
	} else {
		fileResps = []applicationFileResponse{}
	}

	ctx.JSON(http.StatusOK, gin.H{"audit": struct {
		*model.AuditDetail
		Files []applicationFileResponse `json:"files"`
	}{detail, fileResps}})
}

func (a *ApplicationApi) AuditApplication(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
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

	actionStatus, err := model.ParseApplicationStatus(param.Action)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	application, err := a.applicationService.ProcessApplicationAudit(id, userIDStr, param.Result, actionStatus, param.Duration)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		case errors.Is(err, service.ErrApplicationAlreadyProcessed):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "申请已处理"})
			return
		default:
			logger.Errorf("failed to process application %s with action %s: %v", id, param.Action, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "处理申请失败"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"application": *application})
}

func (a *ApplicationApi) Resubmit(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	var (
		keptFileIDs  []string
		newFileHeads []*multipart.FileHeader
		newFileTypes []string
		affiliation  string
		position     string
	)

	if strings.HasPrefix(ctx.ContentType(), "multipart/form-data") {
		keptRaw := ctx.PostForm("keptFileIDs")
		if keptRaw != "" {
			if err := json.Unmarshal([]byte(keptRaw), &keptFileIDs); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的保留文件ID列表"})
				return
			}
		}
		form, err := ctx.MultipartForm()
		if err != nil && err != http.ErrNotMultipart {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的附件数据"})
			return
		}
		if form != nil {
			if files, ok := form.File["files"]; ok {
				newFileHeads = files
			}
			if types, ok := form.Value["types"]; ok {
				newFileTypes = types
			}
			affiliation = ctx.PostForm("affiliation")
			position = ctx.PostForm("position")
		}
	} else {
		var param resubmitApplicationParam
		if err := ctx.ShouldBindJSON(&param); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
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

	application, err := a.applicationService.ResubmitApplication(id, userIDStr, affiliation, position, keptFileIDs, newFileHeads, newFileTypes)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		case errors.Is(err, service.ErrApplicationPermissionDenied):
			ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
			return
		case errors.Is(err, service.ErrApplicationNotEditable):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "申请无法重新提交"})
			return
		default:
			logger.Errorf("failed to resubmit application %s: %v", id, err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "重新提交申请失败"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
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
