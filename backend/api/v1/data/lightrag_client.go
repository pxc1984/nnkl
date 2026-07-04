package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type LightRAGClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type lightRAGTextRequest struct {
	Text       string `json:"text"`
	FileSource string `json:"file_source"`
}

type LightRAGQueryRequest struct {
	Query             string `json:"query"`
	Mode              string `json:"mode"`
	OnlyNeedContext   bool   `json:"only_need_context"`
	Stream            bool   `json:"stream,omitempty"`
	IncludeReferences bool   `json:"include_references,omitempty"`
	ResponseType      string `json:"response_type,omitempty"`
	TopK              int    `json:"top_k,omitempty"`
	MaxTotalTokens    int    `json:"max_total_tokens,omitempty"`
	ChunkTopK         int    `json:"chunk_top_k,omitempty"`
	EnableRerank      bool   `json:"enable_rerank"`
}
type LightRAGQueryResponse struct {
	Response   string          `json:"response"`
	References json.RawMessage `json:"references,omitempty"`
}

func NewLightRAGClient(baseURL, apiKey string, client *http.Client) *LightRAGClient {
	return &LightRAGClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, client: client}
}

func (c *LightRAGClient) IsConfigured() bool { return c.baseURL != "" }

func (c *LightRAGClient) SendText(ctx context.Context, text, fileSource string) error {
	body, err := json.Marshal(lightRAGTextRequest{Text: text, FileSource: fileSource})
	if err != nil {
		return fmt.Errorf("marshal lightrag request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/documents/text", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create lightrag request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAPIKey(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("lightrag request failed: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusConflict:
		slog.Warn("lightrag document already exists, skipping", "file_source", fileSource)
		return nil
	default:
		return fmt.Errorf("lightrag returned status %d", resp.StatusCode)
	}
}

func (c *LightRAGClient) Query(ctx context.Context, query, mode string) (*LightRAGQueryResponse, error) {
	if mode == "" {
		mode = "naive"
	}
	body, err := json.Marshal(LightRAGQueryRequest{Query: query, Mode: mode, IncludeReferences: true})
	if err != nil {
		return nil, fmt.Errorf("marshal lightrag query request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create lightrag query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAPIKey(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send lightrag query request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected lightrag status %d: %s", resp.StatusCode, string(respBody))
	}
	var result LightRAGQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode lightrag response: %w", err)
	}
	return &result, nil
}

func (c *LightRAGClient) QueryStream(ctx context.Context, query, mode string) (*http.Response, error) {
	if mode == "" {
		mode = "naive"
	}
	body, err := json.Marshal(LightRAGQueryRequest{
		Query: query, Mode: mode, Stream: true, IncludeReferences: true,
		ResponseType: "Short answer", TopK: 10, ChunkTopK: 3, MaxTotalTokens: 6000, EnableRerank: false,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal lightrag streaming request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query/stream", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create lightrag streaming request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAPIKey(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send lightrag streaming request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected lightrag status %d: %s", resp.StatusCode, string(respBody))
	}
	return resp, nil
}

func (c *LightRAGClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	c.setAPIKey(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected health status %d", resp.StatusCode)
	}
	return nil
}

func (c *LightRAGClient) setAPIKey(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}
