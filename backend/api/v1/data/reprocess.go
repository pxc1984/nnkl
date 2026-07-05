package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/metrics"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func (a *DataAPI) reprocess(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}

	status := "pending"
	emptyErr := ""
	outputFormat := upload.OutputFormat
	if outputFormat == "" {
		outputFormat = "markdown"
	}
	language := upload.Language
	if language == "" {
		language = "auto"
	}
	updated, err := a.store.UpdateUpload(c.Request.Context(), upload.ID, models.UpdateUploadParams{
		Status:          &status,
		OutputFormat:    &outputFormat,
		Language:        &language,
		Error:           &emptyErr,
		ClearOutputBlob: true,
		ClearClaim:      true,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to requeue object", "internal_error")
		return
	}
	a.queue.Notify()

	metrics.UploadsTotal.WithLabelValues("reprocess").Inc()

	c.JSON(http.StatusAccepted, shared.KnowledgeObject{
		KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(&updated.InputBlob),
		Status:                  updated.Status,
	})
}
