package data

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func TestProcessReferencesFiltersLanguage(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewInMemoryStore()

	createUpload := func(filename, language string) *models.Upload {
		t.Helper()
		blob, err := memStore.CreateBlob(ctx, models.CreateBlobParams{
			Filename: filename, FileType: "pdf", ContentType: "application/pdf", Content: []byte("content"),
		})
		if err != nil {
			t.Fatalf("create blob: %v", err)
		}
		upload, err := memStore.CreateUpload(ctx, models.CreateUploadParams{
			InputBlobID: blob.ID, Status: "completed", Language: language, OutputFormat: "markdown",
		})
		if err != nil {
			t.Fatalf("create upload: %v", err)
		}
		return upload
	}

	ru := createUpload("russian.pdf", "ru")
	en := createUpload("english.pdf", "en")
	raw := json.RawMessage(`[{"file_path":"` + ru.ID + `.md"},{"file_path":"` + en.ID + `.md"}]`)

	api := &DataAPI{store: memStore}
	out, err := api.processReferences(ctx, raw, "ru")
	if err != nil {
		t.Fatalf("processReferences: %v", err)
	}
	var refs []EnrichedReference
	if err := json.Unmarshal(out, &refs); err != nil {
		t.Fatalf("unmarshal references: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != ru.ID || refs[0].Language != "ru" {
		t.Fatalf("expected only Russian reference, got %+v", refs)
	}
}
