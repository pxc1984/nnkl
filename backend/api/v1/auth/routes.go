package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/utils"
)

func RegisterAuthRoutes(router gin.IRouter) {
	a := &AuthAPI{
		store:  store.GetStore(),
		tokens: auth2.NewManager(utils.Settings.AuthSecret, utils.Settings.AccessTokenTTL, utils.Settings.RefreshTokenTTL),
	}

	router.POST("/auth/register", a.register)
	router.POST("/auth/login", a.login)
	router.POST("/auth/refresh", a.refresh)

	protected := router.Group("/auth")
	protected.Use(api.RequireAuth(a.store, a.tokens))
	protected.POST("/logout", a.logout)
	protected.POST("/logout-all", a.logoutAll)
	protected.GET("/sessions", a.sessions)
	protected.GET("/me", a.me)
}
