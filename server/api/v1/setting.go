package v1

import (
	"net/http"
	"server/logger"
	"server/service"

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

	if err := service.SetSetting(req); err != nil {
		logger.Errorf("failed to set setting: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to set setting",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
