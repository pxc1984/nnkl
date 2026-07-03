package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *AuthAPI) me(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)
	c.JSON(http.StatusOK, shared.ToUserResponse(user))
}
