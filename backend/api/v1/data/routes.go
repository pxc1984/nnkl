package data

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/utils"
)

func RegisterDataRoutes(router gin.IRouter) {
	client := &http.Client{Timeout: 2 * time.Minute}
	a := &DataAPI{
		store:    store.GetStore(),
		ocr:      NewOCRClient(utils.Settings.OCRServiceURL, client),
		lightrag: NewLightRAGClient(utils.Settings.LightRAGServiceURL, utils.Settings.LightRAGAPIKey, client),
		maxMB:    utils.Settings.MaxUploadSizeMB,
	}

	protected := router.Group("/data")
	protected.Use(api.RequireAuth(a.store, auth2.NewManager(utils.Settings.AuthSecret, utils.Settings.AccessTokenTTL, utils.Settings.RefreshTokenTTL)))
	protected.GET("", a.list)
	protected.POST("", a.upload)
	protected.POST("/ask", a.ask)
	registerObjectRoutes(protected, a)
}
