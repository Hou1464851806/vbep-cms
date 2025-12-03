package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserApi struct {
}

func (u *UserApi) CreateUser(ctx *gin.Context) {

}

func (u *UserApi) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Delete user id: %s", id)
}

func (u *UserApi) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Update user id: %s", id)
}

func (u *UserApi) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Get user id: %s", id)
}

func (u *UserApi) GetAllUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "all users",
	})
}
