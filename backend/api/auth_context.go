package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
)

const (
	currentUserContextKey = "currentUser"
	authClaimsContextKey  = "authClaims"
)

func SetAuthenticatedRequest(c *gin.Context, claims *auth.AccessClaims, user *store.User) {
	c.Set(authClaimsContextKey, claims)
	c.Set(currentUserContextKey, user)
}

func CurrentUserFromContext(c *gin.Context) (*store.User, bool) {
	user, ok := c.Get(currentUserContextKey)
	if !ok {
		return nil, false
	}
	typedUser, ok := user.(*store.User)
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
