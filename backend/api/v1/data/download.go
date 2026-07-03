package data

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *DataAPI) download(c *gin.Context) {
	blob, err := a.store.GetInputBlobByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", blob.Filename))
	c.Data(http.StatusOK, blob.ContentType, blob.Content)
}
