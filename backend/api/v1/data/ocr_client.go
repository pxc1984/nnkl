package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OCRClient struct {
	baseURL string
	client  *http.Client
}

type OCRParseRequest struct {
	DocumentID   string `json:"document_id"`
	InputBlobID  string `json:"input_blob_id"`
	OutputFormat string `json:"output_format"`
	Language     string `json:"language"`
}

func NewOCRClient(baseURL string, client *http.Client) *OCRClient {
	return &OCRClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (c *OCRClient) Parse(ctx context.Context, payload OCRParseRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/parse", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
