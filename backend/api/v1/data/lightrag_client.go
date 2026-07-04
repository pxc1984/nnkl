package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type LightRAGClient struct {
	baseURL string
	client  *http.Client
}

type LightRAGQueryRequest struct {
	Query           string `json:"query"`
	Mode            string `json:"mode"`
	OnlyNeedContext bool   `json:"only_need_context"`
}

type LightRAGQueryResponse struct {
	Response string `json:"response"`
}

func NewLightRAGClient(baseURL string, client *http.Client) *LightRAGClient {
	return &LightRAGClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *LightRAGClient) Query(ctx context.Context, query string, mode string) (*LightRAGQueryResponse, error) {
	if mode == "" {
		mode = "hybrid"
	}

	payload := LightRAGQueryRequest{
		Query:           query,
		Mode:            mode,
		OnlyNeedContext: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal lightrag query request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create lightrag query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send lightrag query request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected lightrag status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read lightrag response: %w", err)
	}

	var result LightRAGQueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode lightrag response: %w", err)
	}

	return &result, nil
}

func (c *LightRAGClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected health status %d", resp.StatusCode)
	}
	return nil
}
