package data

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/utils"
	"github.com/pxc1984/nnkl-backend/worker"
)

// globalQueue is the background processing queue, exposed for graceful shutdown.
var globalQueue *worker.Queue

// StopQueue stops the background processing queue. Called during server shutdown.
func StopQueue() {
	if globalQueue != nil {
		globalQueue.Stop()
	}
}

// ocrAdapter adapts OCRClient to the worker.OCRService interface.
type ocrAdapter struct {
	client *OCRClient
}

func (a *ocrAdapter) Parse(ctx context.Context, uploadID, inputBlobID, language string) error {
	return a.client.Parse(ctx, OCRParseRequest{
		UploadID:    uploadID,
		InputBlobID: inputBlobID,
		Language:    language,
	})
}

func RegisterDataRoutes(router gin.IRouter) {
	client := &http.Client{Timeout: 10 * time.Minute}
	ocrClient := NewOCRClient(utils.Settings.OCRServiceURL, client)
	lightragClient := NewLightRAGClient(utils.Settings.LightRAGServiceURL, utils.Settings.LightRAGAPIKey, client)

	// Start the background processing queue.
	globalQueue = worker.New(store.GetStore(), &ocrAdapter{client: ocrClient}, lightragClient, 100)
	globalQueue.Start()

	a := &DataAPI{
		store:    store.GetStore(),
		ocr:      ocrClient,
		lightrag: lightragClient,
		maxMB:    utils.Settings.MaxUploadSizeMB,
		queue:    globalQueue,
	}

	protected := router.Group("/data")
	protected.Use(api.RequireAuth(a.store, auth2.NewManager(utils.Settings.AuthSecret, utils.Settings.AccessTokenTTL, utils.Settings.RefreshTokenTTL)))
	protected.GET("", a.list)
	protected.POST("", a.upload)
	protected.POST("/ask", a.ask)
	protected.GET("/ask/sessions", a.listAskSessions)
	protected.GET("/ask/session/:sessionId", a.getAskSession)  // New endpoint for getting specific session
	protected.POST("/ask/stream", a.askStream)
	protected.POST("/graph", a.graph)
	registerObjectRoutes(protected, a)
}