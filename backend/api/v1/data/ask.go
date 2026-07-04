package data

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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

	// Process references to enhance with document information
	processedReferences, err := a.processReferences(c.Request.Context(), resp.References)
	if err != nil {
		// Log the error but continue with original references
		fmt.Printf("Warning: failed to process references: %v\n", err)
		processedReferences = resp.References
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
			References: processedReferences,
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
	if len(processedReferences) > 0 {
		askResp.References = processedReferences
	}

	c.JSON(http.StatusOK, askResp)
}

// processReferences enhances the references with document information for frontend linking
func (a *DataAPI) processReferences(ctx context.Context, rawRefs json.RawMessage) (json.RawMessage, error) {
	if len(rawRefs) == 0 {
		return rawRefs, nil
	}

	// Try to unmarshal the references
	var refsData interface{}
	if err := json.Unmarshal(rawRefs, &refsData); err != nil {
		return rawRefs, fmt.Errorf("failed to unmarshal references: %w", err)
	}

	// If refsData is a string that looks like a document ID, create a structured reference
	if strRef, ok := refsData.(string); ok {
		docID := extractDocumentID(strRef)
		if docID != "" {
			// Try to get document info if possible
			blob, err := a.store.GetBlobByID(ctx, docID)  // Fixed method name
			if err == nil {
				enhancedRef := map[string]interface{}{
					"id":       docID,
					"filename": blob.Filename,
					"type":     "document",
				}
				return json.Marshal([]interface{}{enhancedRef})
			} else {
				enhancedRef := map[string]interface{}{
					"id":   docID,
					"type": "document",
				}
				return json.Marshal([]interface{}{enhancedRef})
			}
		}
		return rawRefs, nil
	}

	// If refsData is an array, process each item
	if refsArray, ok := refsData.([]interface{}); ok {
		for i, refItem := range refsArray {
			if refMap, ok := refItem.(map[string]interface{}); ok {
				// Check if this reference contains a document identifier
				docID := ""
				
				// Look for various possible fields that might contain document IDs
				if filePath, ok := refMap["file_path"].(string); ok {
					docID = extractDocumentID(filePath)
				} else if sourceID, ok := refMap["source_id"].(string); ok {
					docID = extractDocumentID(sourceID)
				} else if refID, ok := refMap["reference_id"].(string); ok {
					docID = extractDocumentID(refID)
				} else if id, ok := refMap["id"].(string); ok {
					docID = extractDocumentID(id)
				} else {
					// If the refMap itself looks like a document ID when converted to string
					refStr := fmt.Sprintf("%v", refMap)
					docID = extractDocumentID(refStr)
				}
				
				if docID != "" {
					// Enhance with document information if available
					blob, err := a.store.GetBlobByID(ctx, docID)  // Fixed method name
					if err == nil {
						refMap["document_id"] = docID
						refMap["document_filename"] = blob.Filename
					} else {
						refMap["document_id"] = docID
					}
				}
				
				refsArray[i] = refMap
			}
		}
		return json.Marshal(refsArray)
	}

	// If refsData is an object, process it similarly
	if refObj, ok := refsData.(map[string]interface{}); ok {
		docID := ""
		
		// Look for various possible fields that might contain document IDs
		if filePath, ok := refObj["file_path"].(string); ok {
			docID = extractDocumentID(filePath)
		} else if sourceID, ok := refObj["source_id"].(string); ok {
			docID = extractDocumentID(sourceID)
		} else if refID, ok := refObj["reference_id"].(string); ok {
			docID = extractDocumentID(refID)
		} else if id, ok := refObj["id"].(string); ok {
			docID = extractDocumentID(id)
		}
		
		if docID != "" {
			// Enhance with document information if available
			blob, err := a.store.GetBlobByID(ctx, docID)  // Fixed method name
			if err == nil {
				refObj["document_id"] = docID
				refObj["document_filename"] = blob.Filename
			} else {
				refObj["document_id"] = docID
			}
		}
		
		return json.Marshal(refObj)
	}

	return rawRefs, nil
}

// extractDocumentID extracts document ID from various possible formats
func extractDocumentID(input string) string {
	// Pattern for document IDs like "doc-..." or SHA-like hashes
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:doc-)?([a-f0-9]{32}|[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`),
		regexp.MustCompile(`([a-f0-9]{32})`), // Just the hash
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(input)
		if len(matches) > 1 {
			docID := matches[1]
			
			// Ensure it's a valid document ID format (must be UUID or SHA256-like)
			if len(docID) == 32 || isValidUUID(docID) {
				// If it doesn't have "doc-" prefix but is a 32-char hex string, add it
				if len(docID) == 32 && !strings.HasPrefix(docID, "doc-") {
					docID = "doc-" + docID
				}
				return docID
			}
		}
	}
	
	return ""
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(u string) bool {
	// Remove any "doc-" prefix if present
	id := strings.TrimPrefix(strings.ToLower(u), "doc-")
	
	// Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12 hex chars)
	if len(id) != 36 {
		return false
	}
	
	// Check positions of hyphens
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	
	// Check that all other positions are hexadecimal characters
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphens
		}
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	
	return true
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