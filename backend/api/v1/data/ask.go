package data

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

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

// EnrichedReference — единый формат источника для фронтенда.
type EnrichedReference struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Number    int       `json:"number,omitempty"`
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

	// Усиливаем запрос числовыми ограничениями, чтобы LLM учитывал их при поиске источников.
	enhancedQuery := enhanceQueryWithNumericConstraints(req.Query)
	if enhancedQuery != req.Query {
		slog.Info("enhanced query with numeric constraints", "original", req.Query, "enhanced", enhancedQuery)
	}

	resp, err := a.lightrag.Query(c.Request.Context(), enhancedQuery, req.Mode)
	if err != nil {
		api.RespondError(c, http.StatusServiceUnavailable, "failed to query knowledge base: "+err.Error(), "service_unavailable")
		return
	}

	// Process references to enhance with document information
	processedReferences, err := a.processReferences(c.Request.Context(), resp.References)
	if err != nil {
		// Log the error but continue with original references
		fmt.Printf("Warning: failed to process references: %v\n", err)
		processedReferences = resp.References
	}

	// Заменяем UUID.md в тексте ответа на имена файлов, сохраняем номера ссылок
	// и удаляем встроенный markdown-блок References, чтобы не дублировать список источников.
	enrichedResponse, processedReferences := enrichResponseReferences(resp.Response, processedReferences)

	// Persist query session
	user, _ := api.CurrentUserFromContext(c)
	var session *models.QuerySession
	if user != nil {
		session, err = a.store.CreateQuerySession(c.Request.Context(), models.CreateQuerySessionParams{
			UserID:     user.ID,
			Query:      req.Query,
			Mode:       req.Mode,
			Response:   enrichedResponse,
			References: processedReferences,
		})
		if err != nil {
			// Non-fatal: log but still return the answer
			_ = err
		}
	}

	askResp := AskResponse{
		Answer: enrichedResponse,
		Mode:   req.Mode,
	}
	if session != nil {
		askResp.SessionID = session.ID
	}
	if len(processedReferences) > 0 {
		askResp.References = processedReferences
	}

	c.JSON(http.StatusOK, askResp)
}

// processReferences превращает сырые references от LightRAG в единый массив
// обогащённых источников с id, именем файла, типом и датой загрузки.
func (a *DataAPI) processReferences(ctx context.Context, rawRefs json.RawMessage) (json.RawMessage, error) {
	if len(rawRefs) == 0 {
		return rawRefs, nil
	}

	var refsData interface{}
	if err := json.Unmarshal(rawRefs, &refsData); err != nil {
		return rawRefs, fmt.Errorf("failed to unmarshal references: %w", err)
	}

	docIDs := make(map[string]struct{})
	collectDocumentIDs(refsData, docIDs)
	if len(docIDs) == 0 {
		return rawRefs, nil
	}

	enriched := make([]EnrichedReference, 0, len(docIDs))
	for id := range docIDs {
		ref := EnrichedReference{ID: id}
		if blob, err := a.store.GetBlobByID(ctx, id); err == nil {
			ref.Filename = blob.Filename
			ref.Type = blob.FileType
			ref.CreatedAt = blob.CreatedAt
		}
		enriched = append(enriched, ref)
	}

	// Детерминированный порядок: сначала по имени файла, затем по id.
	sort.Slice(enriched, func(i, j int) bool {
		if enriched[i].Filename != enriched[j].Filename {
			return enriched[i].Filename < enriched[j].Filename
		}
		return enriched[i].ID < enriched[j].ID
	})

	return json.Marshal(enriched)
}

// collectDocumentIDs рекурсивно извлекает UUID документов из любой структуры references.
func collectDocumentIDs(v interface{}, out map[string]struct{}) {
	switch val := v.(type) {
	case string:
		if id := extractDocumentID(val); id != "" {
			out[id] = struct{}{}
		}
	case []interface{}:
		for _, item := range val {
			collectDocumentIDs(item, out)
		}
	case map[string]interface{}:
		// Сначала проверяем поля, в которых обычно лежит идентификатор.
		for _, key := range []string{"file_path", "source_id", "reference_id", "id", "document_id"} {
			if raw, ok := val[key]; ok {
				collectDocumentIDs(raw, out)
			}
		}
		// Затем ищем UUID в остальных строковых значениях.
		for _, raw := range val {
			collectDocumentIDs(raw, out)
		}
	}
}

// extractDocumentID извлекает UUID документа из строки.
func extractDocumentID(input string) string {
	// UUID с дефисами — основной формат ID в системе.
	re := regexp.MustCompile(`(?i)([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`)
	if m := re.FindStringSubmatch(input); len(m) > 1 {
		return strings.ToLower(m[1])
	}
	// Fallback: 32-символьный hex без дефисов.
	re2 := regexp.MustCompile(`(?i)\b([a-f0-9]{32})\b`)
	if m := re2.FindStringSubmatch(input); len(m) > 1 {
		return strings.ToLower(m[1])
	}
	return ""
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