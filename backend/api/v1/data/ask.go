package data

import (
	"bufio"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	"github.com/pxc1984/nnkl-backend/store/models"
)

type AskRequest struct {
	Query string `json:"query" binding:"required"`
	Mode  string `json:"mode"`
}

type AskResponse struct {
	Answer     string          `json:"answer"`
	Mode       string          `json:"mode"`
	SessionID  string          `json:"sessionId"`
	References json.RawMessage `json:"references,omitempty"`
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

	// Persist query session
	user, _ := api.CurrentUserFromContext(c)
	var session *models.QuerySession
	if user != nil {
		session, err = a.store.CreateQuerySession(c.Request.Context(), models.CreateQuerySessionParams{
			UserID:     user.ID,
			Query:      req.Query,
			Mode:       req.Mode,
			Response:   resp.Response,
			References: resp.References,
		})
		if err != nil {
			// Non-fatal: log but still return the answer
			_ = err
		}
	}

	askResp := AskResponse{
		Answer: resp.Response,
		Mode:   req.Mode,
	}
	if session != nil {
		askResp.SessionID = session.ID
	}
	if len(resp.References) > 0 {
		askResp.References = resp.References
	}

	c.JSON(http.StatusOK, askResp)
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

	// Create a session record for the streaming query (response will be empty)
	user, _ := api.CurrentUserFromContext(c)
	if user != nil {
		_, _ = a.store.CreateQuerySession(c.Request.Context(), models.CreateQuerySessionParams{
			UserID: user.ID,
			Query:  req.Query,
			Mode:   req.Mode,
		})
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
