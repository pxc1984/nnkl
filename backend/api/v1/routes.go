package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api/v1/auth"
	"github.com/pxc1984/nnkl-backend/api/v1/data"
)

func RegisterRoutes(router gin.IRouter) {
	auth.RegisterAuthRoutes(router)
	data.RegisterDataRoutes(router)
}

// StopDataQueue stops the background processing queue. Called during server shutdown.
func StopDataQueue() {
	data.StopQueue()
}
