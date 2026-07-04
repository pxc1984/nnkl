package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/worker"
)

func (a *DataAPI) reprocess(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}

	// Enqueue for background processing instead of blocking.
	job := worker.Job{
		UploadID:     upload.ID,
		OutputFormat: "markdown",
		Language:     "auto",
		FileType:     upload.InputBlob.FileType,
	}
	switch upload.InputBlob.FileType {
	case "docx", "pptx":
		a.queue.EnqueueSimple(job)
	case "pdf":
		a.queue.EnqueueOCR(job)
	case "markdown":
		a.finalizeMarkdown(c, upload.ID, "auto", "markdown")
	}

	status := upload.Status
	if status == "" {
		status = "queued"
	}
	c.JSON(http.StatusAccepted, shared.KnowledgeObject{
		KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(&upload.InputBlob),
		Status:                  status,
	})
}
