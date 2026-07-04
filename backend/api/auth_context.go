package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store/models"
)

const (
	currentUserContextKey = "currentUser"
	authClaimsContextKey  = "authClaims"
)

func SetAuthenticatedRequest(c *gin.Context, claims *auth.AccessClaims, user *models.User) {
	c.Set(authClaimsContextKey, claims)
	c.Set(currentUserContextKey, user)
}

func CurrentUserFromContext(c *gin.Context) (*models.User, bool) {
	user, ok := c.Get(currentUserContextKey)
	if !ok {
		return nil, false
	}
	typedUser, ok := user.(*models.User)
	return typedUser, ok
}

func CurrentClaimsFromContext(c *gin.Context) (*auth.AccessClaims, bool) {
	claims, ok := c.Get(authClaimsContextKey)
	if !ok {
		return nil, false
	}
	typedClaims, ok := claims.(*auth.AccessClaims)
	return typedClaims, ok
}
