package v1

import (
	"net/http"
	"server/logger"
	"server/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetSetting(c *gin.Context) {
	setting, err := service.GetSetting()
	if err != nil {
		logger.Errorf("failed to get setting: %v", err)
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
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resp,
	})
}

func SetSetting(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误: " + err.Error()})
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

	if err := service.SetSetting(dbReq); err != nil {
		logger.Errorf("failed to set setting: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "保存设置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
