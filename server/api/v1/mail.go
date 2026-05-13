package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/service"
)

type MailApi struct {
	service *service.MailService
}

func NewMailApi(service *service.MailService) *MailApi {
	return &MailApi{service: service}
}

func (a *MailApi) SendResetPassword(ctx *gin.Context) {

}

type NotificationRequest struct {
	To      string `json:"to" binding:"required"`
	Message string `json:"message" binding:"required"`
}

func (a *MailApi) SendNotification(ctx *gin.Context) {
	var req NotificationRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "请求参数错误",
		})
		return
	}

	//err := a.service.SendNotification(req.To, req.Message, "")
	//if err != nil {
	//	logger.Errorf("Failed to send notification email by service: %v", err)
	//	ctx.JSON(http.StatusInternalServerError, gin.H{
	//		"message": "邮件发送失败",
	//	})
	//	return
	//}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "邮件发送成功",
	})
}
