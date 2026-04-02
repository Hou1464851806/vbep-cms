package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"server/logger"
	"server/model"
	"server/service"
)

type LicenseApi struct {
	licenseService *service.LicenseService
}

func NewLicenseApi(licenseService *service.LicenseService) *LicenseApi {
	return &LicenseApi{
		licenseService: licenseService,
	}
}

func (a *LicenseApi) GetAll(ctx *gin.Context) {
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

	var filterID *string
	// If user is common, only see their own licenses
	if currentUser.Role == model.Common {
		filterID = &currentUser.ID
	}

	query := service.LicenseListQuery{
		ApplicantID:   filterID,
		ApplicationID: ctx.Query("application_id"),
		Page:          page,
		PageSize:      pageSize,
		SortBy:        ctx.Query("sort_by"),
		Order:         ctx.Query("order"),
		IDPrefix:      ctx.Query("id_prefix"),
		DatasetName:   ctx.Query("dataset_name"),
		Applicant:     ctx.Query("applicant"),
		Status:        ctx.Query("status"),
		DateStart:     ctx.Query("date_start"),
		DateEnd:       ctx.Query("date_end"),
		ExpireStart:   ctx.Query("expire_start"),
		ExpireEnd:     ctx.Query("expire_end"),
	}

	licenses, total, err := a.licenseService.ListLicenses(query)
	if err != nil {
		logger.Errorf("failed to list licenses: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载许可列表失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"licenses": licenses,
		"total":    total,
	})
}

type revokeLicenseParam struct {
	Reason string `json:"reason" binding:"required"`
}

func (a *LicenseApi) Revoke(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的许可ID"})
		return
	}

	var param revokeLicenseParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "必须提供撤销理由"})
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

	if currentUser.Role <= model.Common {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	err = a.licenseService.RevokeLicense(id, param.Reason)
	if err != nil {
		logger.Errorf("failed to revoke license: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "撤销许可失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "许可已撤销"})
}
