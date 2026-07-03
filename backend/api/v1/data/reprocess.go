package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *DataAPI) reprocess(c *gin.Context) {
	blob, err := a.store.GetInputBlobByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	if err := a.reprocessBlob(c, blob.ID, "markdown", "auto"); err != nil {
		return
	}
	job, _ := a.store.GetParseJobByDocumentID(c.Request.Context(), blob.ID)
	status := "queued"
	if job != nil && job.Status != "" {
		status = job.Status
	}
	c.JSON(http.StatusAccepted, shared.KnowledgeObject{
		KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(blob),
		Status:                  status,
	})
}
