package v1

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"server/logger"
	"server/model"
	"server/service"
)

type ApplicationApi struct {
	service *service.ApplicationService
}

func NewApplicationApi() *ApplicationApi {
	return &ApplicationApi{service: service.NewApplicationService()}
}

type createApplicationParam struct {
	DatasetID string `json:"datasetId" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

type userBrief struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

type datasetBrief struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type applicationResponse struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Reason    string       `json:"reason"`
	Applicant userBrief    `json:"applicant"`
	Dataset   datasetBrief `json:"dataset"`
	Auditor   *userBrief   `json:"auditor,omitempty"`
	AuditedAt *time.Time   `json:"auditedAt"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (a *ApplicationApi) Create(ctx *gin.Context) {
	var param createApplicationParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "invalid user id"})
		return
	}

	app := &model.Application{
		DatasetID:   param.DatasetID,
		ApplicantID: userIDStr,
		Status:      model.PENDING,
		Reason:      param.Reason,
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
	applications, err := a.service.ListApplications()
	if err != nil {
		logger.Errorf("failed to list applications: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load applications"})
		return
	}

	resp := make([]applicationResponse, 0, len(applications))
	for _, application := range applications {
		resp = append(resp, buildApplicationResponse(application))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":      "success",
		"applications": resp,
	})
}

func (a *ApplicationApi) GetOne(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

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

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "success",
		"application": buildApplicationResponse(*application),
	})
}

func buildApplicationResponse(app model.Application) applicationResponse {
	resp := applicationResponse{
		ID:        app.ID,
		Status:    formatApplicationStatus(app.Status),
		Reason:    app.Reason,
		CreatedAt: app.CreatedAt,
		AuditedAt: app.AuditedAt,
		Applicant: userBrief{
			ID:       app.Applicant.ID,
			Nickname: app.Applicant.NickName,
			Email:    app.Applicant.Email,
		},
		Dataset: datasetBrief{
			ID:    app.Dataset.ID,
			Title: app.Dataset.Title,
		},
	}

	if app.Auditor != nil {
		resp.Auditor = &userBrief{
			ID:       app.Auditor.ID,
			Nickname: app.Auditor.NickName,
			Email:    app.Auditor.Email,
		}
	}

	return resp
}

func formatApplicationStatus(status model.ApplicationStatus) string {
	switch status {
	case model.APPROVED:
		return "approved"
	case model.DISAPPROVED:
		return "disapproved"
	default:
		return "pending"
	}
}
