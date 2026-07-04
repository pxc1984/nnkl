package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *DataAPI) reprocess(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	if err := a.reprocessBlob(c, upload.ID, "markdown", "auto"); err != nil {
		return
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
