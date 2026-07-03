package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/store"
)

func HealthCheck(ctx *gin.Context) {
	st := store.GetStore()
	pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()

	if err := st.Ping(pingCtx); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"store":    st.Backend(),
			"database": "down",
			"error":    err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"store":    st.Backend(),
		"database": "up",
	})
}
