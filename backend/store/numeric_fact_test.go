package store

import (
	"context"
	"testing"

	"github.com/pxc1984/nnkl-backend/store/models"
)

func TestInMemoryStore_NumericFacts(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryStore()

	fact := &models.NumericFact{
		DocumentID: "doc-1",
		ChunkID:    "chunk-1",
		EntityName: "никелевая руда",
		Property:   "концентрация сульфатов",
		Value:      250,
		Unit:       "мг/л",
		Operator:   "<=",
		RawText:    "сульфаты ≤ 250 мг/л",
	}

	if err := s.CreateNumericFact(ctx, fact); err != nil {
		t.Fatalf("CreateNumericFact: %v", err)
	}
	if fact.ID == "" {
		t.Error("expected fact id to be generated")
	}

	facts, err := s.ListNumericFacts(ctx, models.NumericFactFilter{Property: "концентрация сульфатов"})
	if err != nil {
		t.Fatalf("ListNumericFacts: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Value != 250 {
		t.Errorf("expected value 250, got %f", facts[0].Value)
	}

	facts, err = s.ListNumericFacts(ctx, models.NumericFactFilter{Min: 200, Max: 300})
	if err != nil {
		t.Fatalf("ListNumericFacts range: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact in range, got %d", len(facts))
	}

	facts, err = s.ListNumericFacts(ctx, models.NumericFactFilter{Min: 300, Max: 400})
	if err != nil {
		t.Fatalf("ListNumericFacts range empty: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("expected 0 facts, got %d", len(facts))
	}

	if err := s.DeleteNumericFactsByDocumentID(ctx, "doc-1"); err != nil {
		t.Fatalf("DeleteNumericFactsByDocumentID: %v", err)
	}
	facts, err = s.ListNumericFacts(ctx, models.NumericFactFilter{})
	if err != nil {
		t.Fatalf("ListNumericFacts after delete: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("expected 0 facts after delete, got %d", len(facts))
	}
}

func TestInMemoryStore_FindDocumentsByNumericFacts(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryStore()

	facts := []models.NumericFact{
		{DocumentID: "doc-1", Property: "концентрация сульфатов", Value: 250, Unit: "мг/л"},
		{DocumentID: "doc-1", Property: "температура", Value: 30, Unit: "°c"},
		{DocumentID: "doc-2", Property: "концентрация сульфатов", Value: 150, Unit: "мг/л"},
		{DocumentID: "doc-3", Property: "концентрация хлоридов", Value: 300, Unit: "мг/л"},
	}
	if err := s.CreateNumericFacts(ctx, facts); err != nil {
		t.Fatalf("CreateNumericFacts: %v", err)
	}

	ids, err := s.FindDocumentsByNumericFacts(ctx, []models.NumericFactFilter{
		{Property: "концентрация сульфатов", Min: 200, Max: 300},
	})
	if err != nil {
		t.Fatalf("FindDocumentsByNumericFacts: %v", err)
	}
	if len(ids) != 1 || ids[0] != "doc-1" {
		t.Fatalf("expected [doc-1], got %v", ids)
	}

	ids, err = s.FindDocumentsByNumericFacts(ctx, []models.NumericFactFilter{
		{Property: "концентрация сульфатов", Min: 100, Max: 200},
	})
	if err != nil {
		t.Fatalf("FindDocumentsByNumericFacts: %v", err)
	}
	if len(ids) != 1 || ids[0] != "doc-2" {
		t.Fatalf("expected [doc-2], got %v", ids)
	}

	ids, err = s.FindDocumentsByNumericFacts(ctx, []models.NumericFactFilter{
		{Property: "концентрация сульфатов", Min: 200, Max: 300},
		{Property: "температура", Min: 20, Max: 40},
	})
	if err != nil {
		t.Fatalf("FindDocumentsByNumericFacts multi: %v", err)
	}
	if len(ids) != 1 || ids[0] != "doc-1" {
		t.Fatalf("expected [doc-1] for multi-filter, got %v", ids)
	}
}
