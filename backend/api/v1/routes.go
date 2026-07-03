package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api/v1/auth"
)

func RegisterRoutes(router gin.IRouter) {
	auth.RegisterAuthRoutes(router)
}
