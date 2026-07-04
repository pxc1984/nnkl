package data

import "github.com/pxc1984/nnkl-backend/store"

type DataAPI struct {
	store    store.Store
	ocr      *OCRClient
	lightrag *LightRAGClient
	maxMB    int64
}

type DataUploadParams struct {
	Tags         []string `json:"tags"`
	OutputFormat string   `json:"outputFormat"`
	Language     string   `json:"language"`
}

type DataUpdateParams struct {
	Tags         []string `json:"tags"`
	OutputFormat string   `json:"outputFormat"`
	Language     string   `json:"language"`
}

type DataUploadItem struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type DataUploadResponse struct {
	Items []DataUploadItem `json:"items"`
}
