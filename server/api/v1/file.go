package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type FileApi struct {
	service *service.FileService
}

func NewFileApi() *FileApi {
	return &FileApi{service: service.NewFileService()}
}

func (a *FileApi) DownloadByDataset(ctx *gin.Context) {

}

func (a *FileApi) DownloadByRequest(ctx *gin.Context) {

}

func (a *FileApi) UploadToDataset(ctx *gin.Context) {

}

func (a *FileApi) UploadToRequest(ctx *gin.Context) {

}
