package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

func NewLightRAGClient(baseURL, apiKey string, client *http.Client) *LightRAGClient {
	return &LightRAGClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  client,
	}
}

func (c *LightRAGClient) IsConfigured() bool {
	return c.baseURL != ""
}

func (c *LightRAGClient) SendText(ctx context.Context, text, fileSource string) error {
	body, err := json.Marshal(lightRAGTextRequest{
		Text:       text,
		FileSource: fileSource,
	})
	if err != nil {
		return fmt.Errorf("marshal lightrag request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/documents/text", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create lightrag request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

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
