package data

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func TestProcessReferences(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewInMemoryStore()

	blob, err := memStore.CreateBlob(ctx, models.CreateBlobParams{
		Filename:    "report_nickel_extraction.pdf",
		FileType:    "pdf",
		ContentType: "application/pdf",
		SizeBytes:   12345,
		Content:     []byte("content"),
	})
	if err != nil {
		t.Fatalf("create blob: %v", err)
	}

	api := &DataAPI{store: memStore}

	t.Run("file_path", func(t *testing.T) {
		raw := json.RawMessage(`[{"file_path": "` + blob.ID + `.md", "source_id": "chunk-1"}]`)
		out, err := api.processReferences(ctx, raw)
		if err != nil {
			t.Fatalf("processReferences: %v", err)
		}

		var refs []EnrichedReference
		if err := json.Unmarshal(out, &refs); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}

		if len(refs) != 1 {
			t.Fatalf("expected 1 reference, got %d", len(refs))
		}
		if refs[0].ID != blob.ID {
			t.Errorf("id: got %q, want %q", refs[0].ID, blob.ID)
		}
		if refs[0].Filename != blob.Filename {
			t.Errorf("filename: got %q, want %q", refs[0].Filename, blob.Filename)
		}
		if refs[0].Type != blob.FileType {
			t.Errorf("type: got %q, want %q", refs[0].Type, blob.FileType)
		}
		if refs[0].CreatedAt.IsZero() {
			t.Error("createdAt is zero")
		}
	})

	t.Run("string reference", func(t *testing.T) {
		raw := json.RawMessage(`"` + blob.ID + `"`)
		out, err := api.processReferences(ctx, raw)
		if err != nil {
			t.Fatalf("processReferences: %v", err)
		}

		var refs []EnrichedReference
		if err := json.Unmarshal(out, &refs); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}

		if len(refs) != 1 || refs[0].ID != blob.ID {
			t.Fatalf("expected enriched reference for blob %s, got %+v", blob.ID, refs)
		}
	})

	t.Run("deduplicates same document", func(t *testing.T) {
		raw := json.RawMessage(`[
			{"file_path": "` + blob.ID + `.md"},
			{"source_id": "` + blob.ID + `"},
			{"reference_id": "` + blob.ID + `"}
		]`)
		out, err := api.processReferences(ctx, raw)
		if err != nil {
			t.Fatalf("processReferences: %v", err)
		}

		var refs []EnrichedReference
		if err := json.Unmarshal(out, &refs); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}

		if len(refs) != 1 {
			t.Fatalf("expected 1 deduplicated reference, got %d", len(refs))
		}
	})

	t.Run("missing blob", func(t *testing.T) {
		missingID := "00000000-0000-0000-0000-000000000000"
		raw := json.RawMessage(`[{"file_path": "` + missingID + `.md"}]`)
		out, err := api.processReferences(ctx, raw)
		if err != nil {
			t.Fatalf("processReferences: %v", err)
		}

		var refs []EnrichedReference
		if err := json.Unmarshal(out, &refs); err != nil {
			t.Fatalf("unmarshal result: %v", err)
		}

		if len(refs) != 1 {
			t.Fatalf("expected 1 reference, got %d", len(refs))
		}
		if refs[0].ID != missingID {
			t.Errorf("id: got %q, want %q", refs[0].ID, missingID)
		}
		if refs[0].Filename != "" {
			t.Errorf("expected empty filename for missing blob, got %q", refs[0].Filename)
		}
	})

	t.Run("empty references", func(t *testing.T) {
		raw := json.RawMessage(`[]`)
		out, err := api.processReferences(ctx, raw)
		if err != nil {
			t.Fatalf("processReferences: %v", err)
		}
		if string(out) != "[]" {
			t.Errorf("expected empty array, got %s", string(out))
		}
	})
}
