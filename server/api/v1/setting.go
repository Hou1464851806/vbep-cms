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
			"message": "failed to get setting",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    setting,
	})
}

func SetSetting(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Parameter error: " + err.Error()})
		return
	}

	// Handle template_file: if it's an array, join it to a comma-separated string
	if val, ok := req["template_file"]; ok {
		switch v := val.(type) {
		case []interface{}:
			var strIDs []string
			for _, item := range v {
				if s, ok := item.(string); ok {
					strIDs = append(strIDs, s)
				}
			}
			req["template_file"] = strings.Join(strIDs, ",")
		}
	}

	if err := service.SetSetting(req); err != nil {
		logger.Errorf("failed to set setting: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to set setting",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
