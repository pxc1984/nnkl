package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/metrics"
)

func (a *DataAPI) delete(c *gin.Context) {
	if err := a.store.DeleteUploadByID(c.Request.Context(), c.Param("id")); err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	metrics.UploadsTotal.WithLabelValues("deleted").Inc()
	c.Status(http.StatusNoContent)
}
