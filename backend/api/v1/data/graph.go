package data

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
)

type GraphRequest struct {
	Query string `json:"query" binding:"required,min=3"`
	Mode  string `json:"mode"`
}

type GraphNode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type GraphEdge struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Weight      float64 `json:"weight,omitempty"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Mode  string      `json:"mode"`
}

func convertLightRAGDataToGraph(resp *LightRAGQueryDataResponse, mode string) GraphResponse {
	nodes := make([]GraphNode, 0, len(resp.Data.Entities))
	nodeIDs := make(map[string]struct{})
	for _, entity := range resp.Data.Entities {
		id := strings.TrimSpace(entity.EntityName)
		if id == "" {
			continue
		}
		if _, exists := nodeIDs[id]; exists {
			continue
		}
		nodeIDs[id] = struct{}{}
		nodes = append(nodes, GraphNode{
			ID:          id,
			Label:       id,
			Type:        strings.TrimSpace(entity.EntityType),
			Description: strings.TrimSpace(entity.Description),
		})
	}

	edges := make([]GraphEdge, 0, len(resp.Data.Relationships))
	for _, relation := range resp.Data.Relationships {
		src := strings.TrimSpace(relation.SrcID)
		tgt := strings.TrimSpace(relation.TgtID)
		if src == "" || tgt == "" {
			continue
		}
		if _, exists := nodeIDs[src]; !exists {
			continue
		}
		if _, exists := nodeIDs[tgt]; !exists {
			continue
		}
		label := strings.TrimSpace(relation.Keywords)
		if label == "" {
			label = "связан_с"
		}
		edges = append(edges, GraphEdge{
			Source:      src,
			Target:      tgt,
			Label:       label,
			Description: strings.TrimSpace(relation.Description),
			Weight:      relation.Weight,
		})
	}

	return GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Mode:  mode,
	}
}

func (a *DataAPI) graph(c *gin.Context) {
	var req GraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	if !a.lightrag.IsConfigured() {
		api.RespondError(c, http.StatusServiceUnavailable, "lightrag service is not configured", "service_unavailable")
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "hybrid"
	}

	resp, err := a.lightrag.QueryData(c.Request.Context(), req.Query, mode)
	if err != nil {
		api.RespondError(c, http.StatusServiceUnavailable, "failed to query knowledge graph: "+err.Error(), "service_unavailable")
		return
	}

	c.JSON(http.StatusOK, convertLightRAGDataToGraph(resp, mode))
}
