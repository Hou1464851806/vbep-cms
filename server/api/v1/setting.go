package v1

import (
	"net/http"
	"server/logger"
	"server/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type SettingApi struct {
	service *service.SettingService
}

func NewSettingApi() *SettingApi {
	return &SettingApi{service: service.NewSettingService()}
}

func (a *SettingApi) GetSetting(c *gin.Context) {
	setting, err := a.service.GetSetting()
	if err != nil {
		logger.Errorw("failed to get setting", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取设置失败",
		})
		return
	}

	// Map snake_case (DB) to camelCase (Frontend)
	resp := map[string]interface{}{
		"templateFile":        setting.TemplateFile,
		"publicEmail":         setting.PublicEmail,
		"templateDescription": setting.TemplateDescription,
		"userAgreement":       setting.UserAgreement,
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resp,
	})
}

func (a *SettingApi) SetSetting(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求参数错误"})
		return
	}

	// Map camelCase (Frontend) to snake_case (DB)
	dbReq := make(map[string]interface{})

	// Handle templateFile -> template_file
	if val, ok := req["templateFile"]; ok {
		switch v := val.(type) {
		case []interface{}:
			var strIDs []string
			for _, item := range v {
				if s, ok := item.(string); ok {
					strIDs = append(strIDs, s)
				}
			}
			dbReq["template_file"] = strings.Join(strIDs, ",")
		}
	}
	if val, ok := req["publicEmail"]; ok {
		dbReq["public_email"] = val
	}
	if val, ok := req["templateDescription"]; ok {
		dbReq["template_description"] = val
	}
	if val, ok := req["userAgreement"]; ok {
		dbReq["user_agreement"] = val
	}

	if err := a.service.SetSetting(dbReq); err != nil {
		logger.Errorw("failed to set setting", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "保存设置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (a *SettingApi) GetUserAgreement(c *gin.Context) {
	setting, err := a.service.GetSetting()
	if err != nil {
		logger.Errorw("failed to get setting", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取协议失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"content": setting.UserAgreement,
	})
}
