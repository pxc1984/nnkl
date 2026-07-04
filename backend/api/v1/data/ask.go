package data

import (
	"bufio"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
)

type AskRequest struct {
	Query string `json:"query" binding:"required"`
	Mode  string `json:"mode"`
}

type AskResponse struct {
	Answer string `json:"answer"`
	Mode   string `json:"mode"`
}

func (a *DataAPI) ask(c *gin.Context) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	if a.lightrag == nil {
		api.RespondError(c, http.StatusServiceUnavailable, "lightrag service is not configured", "service_unavailable")
		return
	}

	resp, err := a.lightrag.Query(c.Request.Context(), req.Query, req.Mode)
	if err != nil {
		api.RespondError(c, http.StatusServiceUnavailable, "failed to query knowledge base: "+err.Error(), "service_unavailable")
		return
	}

	c.JSON(http.StatusOK, AskResponse{
		Answer: resp.Response,
		Mode:   req.Mode,
	})
}
func (a *DataAPI) askStream(c *gin.Context) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	if !a.lightrag.IsConfigured() {
		api.RespondError(c, http.StatusServiceUnavailable, "lightrag service is not configured", "service_unavailable")
		return
	}
	resp, err := a.lightrag.QueryStream(c.Request.Context(), req.Query, req.Mode)
	if err != nil {
		api.RespondError(c, http.StatusServiceUnavailable, "failed to query knowledge base: "+err.Error(), "service_unavailable")
		return
	}
	defer resp.Body.Close()
	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher, canFlush := c.Writer.(http.Flusher)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		_, _ = c.Writer.Write(append(scanner.Bytes(), '\n'))
		if canFlush {
			flusher.Flush()
		}
	}
}
