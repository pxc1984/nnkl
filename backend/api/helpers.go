package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func RespondInvalidCredentials(c *gin.Context) {
	RespondError(c, http.StatusUnauthorized, "invalid email or password", "unauthorized")
}

func RespondError(c *gin.Context, status int, message, code string) {
	c.JSON(status, ErrorResponse{Message: message, Code: code})
}
