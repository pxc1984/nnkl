package data

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *DataAPI) download(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", upload.InputBlob.Filename))
	c.Data(http.StatusOK, upload.InputBlob.ContentType, upload.InputBlob.Content)
}
