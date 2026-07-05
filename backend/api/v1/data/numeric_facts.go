package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	"github.com/pxc1984/nnkl-backend/store/models"
)

// listNumericFacts возвращает извлечённые числовые факты для документа.
// Используется для отладки фильтрации по числовым параметрам.
func (a *DataAPI) listNumericFacts(c *gin.Context) {
	docID := c.Param("documentId")
	if docID == "" {
		api.RespondError(c, http.StatusBadRequest, "document id is required", "bad_request")
		return
	}

	facts, err := a.store.ListNumericFacts(c.Request.Context(), models.NumericFactFilter{
		DocumentID: docID,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to list numeric facts", "internal_error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": facts,
	})
}
