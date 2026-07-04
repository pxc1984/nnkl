package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *DataAPI) delete(c *gin.Context) {
	if err := a.store.DeleteUploadByID(c.Request.Context(), c.Param("id")); err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	c.Status(http.StatusNoContent)
}
