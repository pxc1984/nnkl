package data

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractNumericConstraints(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
		first NumericConstraint
	}{
		{
			name:  "less-or-equal with unicode",
			query: "Какие методы обессоливания подходят, если сульфаты ≤300 мг/л?",
			want:  1,
			first: NumericConstraint{Property: "сульфаты", Operator: "<=", Value: 300, Unit: "мг/л"},
		},
		{
			name:  "greater-than with celsius",
			query: "сплавы с температурой плавления > 1000 °C",
			want:  1,
			first: NumericConstraint{Property: "температурой плавления", Operator: ">", Value: 1000, Unit: "°c"},
		},
		{
			name:  "less-or-equal without unit",
			query: "pH ≤ 8,5",
			want:  1,
			first: NumericConstraint{Property: "ph", Operator: "<=", Value: 8.5, Unit: ""},
		},
		{
			name:  "russian word operator",
			query: "концентрация хлоридов не более 200 мг/л",
			want:  1,
			first: NumericConstraint{Property: "концентрация хлоридов", Operator: "<=", Value: 200, Unit: "мг/л"},
		},
		{
			name:  "range",
			query: "скорость потока от 10 до 50 м/с",
			want:  1,
			first: NumericConstraint{Property: "потока", Operator: "between", Value: 10, Value2: 50, Unit: "м/с"},
		},
		{
			name:  "decimal separator comma",
			query: "pH ≤ 8,5",
			want:  1,
			first: NumericConstraint{Property: "ph", Operator: "<=", Value: 8.5, Unit: ""},
		},
		{
			name:  "no constraints",
			query: "Какие методы обессоливания подходят?",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNumericConstraints(tt.query)
			if len(got) != tt.want {
				t.Fatalf("expected %d constraints, got %d: %+v", tt.want, len(got), got)
			}
			if tt.want > 0 && len(got) > 0 {
				if got[0].Property != tt.first.Property || got[0].Operator != tt.first.Operator || got[0].Value != tt.first.Value || got[0].Unit != tt.first.Unit {
					t.Errorf("first constraint mismatch: got %+v, want %+v", got[0], tt.first)
				}
			}
		})
	}
}

func TestEnhanceQueryWithNumericConstraints(t *testing.T) {
	query := "методы при сульфатах ≤300 мг/л"
	enhanced := enhanceQueryWithNumericConstraints(query)

	if !strings.Contains(enhanced, query) {
		t.Error("enhanced query should contain original query")
	}
	if !strings.Contains(enhanced, "не более 300 мг/л") {
		t.Errorf("enhanced query should contain constraint instruction, got:\n%s", enhanced)
	}
}

func TestEnhanceQueryWithoutConstraints(t *testing.T) {
	query := "расскажи про электроэкстракцию никеля"
	enhanced := enhanceQueryWithNumericConstraints(query)
	if enhanced != query {
		t.Errorf("expected unchanged query, got:\n%s", enhanced)
	}
}

func TestEnrichResponseReferences(t *testing.T) {
	refs := json.RawMessage(`[
		{"id": "14bd1a61-3a2f-4936-a2bd-00c5266ac123", "filename": "Bindura_2010.pdf", "type": "pdf", "createdAt": "2026-01-01T00:00:00Z"},
		{"id": "aea34f4a-afb9-40bf-8db4-0fb0391bfccf", "filename": "Cunico Resources.pdf", "type": "pdf", "createdAt": "2026-01-01T00:00:00Z"}
	]`)

	response := "## References\n- [1] 14bd1a61-3a2f-4936-a2bd-00c5266ac123.md\n- [3] aea34f4a-afb9-40bf-8db4-0fb0391bfccf.md\n\nSome text with [1] and [3]."

	enrichedResponse, enrichedRefs := enrichResponseReferences(response, refs)

	if strings.Contains(enrichedResponse, "## References") {
		t.Error("References block should be removed")
	}
	if strings.Contains(enrichedResponse, "14bd1a61-3a2f-4936-a2bd-00c5266ac123") {
		t.Error("uuid should be replaced with filename")
	}
	if !strings.Contains(enrichedResponse, "Some text with [1] and [3].") {
		t.Errorf("text after References block should be preserved, got:\n%s", enrichedResponse)
	}

	var parsedRefs []EnrichedReference
	if err := json.Unmarshal(enrichedRefs, &parsedRefs); err != nil {
		t.Fatalf("failed to unmarshal enriched refs: %v", err)
	}
	if len(parsedRefs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(parsedRefs))
	}
	if parsedRefs[0].Number != 1 {
		t.Errorf("expected ref 0 number 1, got %d", parsedRefs[0].Number)
	}
	if parsedRefs[1].Number != 3 {
		t.Errorf("expected ref 1 number 3, got %d", parsedRefs[1].Number)
	}
}
