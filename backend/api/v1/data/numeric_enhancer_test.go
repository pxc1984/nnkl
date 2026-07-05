package data

import (
	"testing"

	"github.com/pxc1984/nnkl-backend/store/models"
)

func TestBuildNumericFilters(t *testing.T) {
	tests := []struct {
		name string
		req  AskRequest
		want []models.NumericFactFilter
	}{
		{
			name: "explicit filters only",
			req: AskRequest{
				Query: "test",
				NumericFilters: []NumericFilter{
					{Property: "концентрация", Min: 10, Max: 20, Unit: "мг/л"},
				},
			},
			want: []models.NumericFactFilter{
				{Property: "концентрация", Min: 10, Max: 20, Unit: "мг/л"},
			},
		},
		{
			name: "extracted from query",
			req: AskRequest{
				Query: "концентрация сульфатов от 100 до 200 мг/л",
			},
			want: []models.NumericFactFilter{
				{Property: "концентрация сульфатов", Min: 100, Max: 200, Unit: "мг/л"},
			},
		},
		{
			name: "explicit and extracted merged",
			req: AskRequest{
				Query: "pH больше 7",
				NumericFilters: []NumericFilter{
					{Property: "температура", Min: 20, Max: 30, Unit: "°C"},
				},
			},
			want: []models.NumericFactFilter{
				{Property: "температура", Min: 20, Max: 30, Unit: "°c"},
				{Property: "ph", Min: 7, Max: 0, Unit: ""},
			},
		},
		{
			name: "no filters",
			req: AskRequest{Query: "просто вопрос"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNumericFilters(tt.req)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d filters, got %d: %+v", len(tt.want), len(got), got)
			}
			for i := range got {
				if got[i].Property != tt.want[i].Property ||
					got[i].Min != tt.want[i].Min ||
					got[i].Max != tt.want[i].Max ||
					got[i].Unit != tt.want[i].Unit {
					t.Errorf("filter %d: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEnhanceQueryWithDocumentFilter(t *testing.T) {
	query := "какие способы применялись"
	docIDs := []string{"doc-1", "doc-2"}
	got := enhanceQueryWithDocumentFilter(query, docIDs)
	if got == query {
		t.Error("expected enhanced query to differ from original")
	}
	for _, id := range docIDs {
		if !contains(got, id) {
			t.Errorf("expected enhanced query to contain %q", id)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
