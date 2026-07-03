package data

import "github.com/gin-gonic/gin"

func registerObjectRoutes(group *gin.RouterGroup, a *DataAPI) {
	group.GET("/:id", a.get)
	group.PATCH("/:id", a.update)
	group.DELETE("/:id", a.delete)
	group.GET("/:id/download", a.download)
	group.POST("/:id/reprocess", a.reprocess)
}
