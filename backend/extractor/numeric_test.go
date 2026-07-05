package extractor

import (
	"context"
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"```json\n[{\"value\": 1}]\n```", "[{\"value\": 1}]"},
		{"```\n[{\"value\": 1}]\n```", "[{\"value\": 1}]"},
		{"[{\"value\": 1}]", "[{\"value\": 1}]"},
	}

	for _, tt := range tests {
		got := extractJSON(tt.input)
		if got != tt.want {
			t.Errorf("extractJSON(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeOperator(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<=", "<="},
		{"≤", "<="},
		{"не более", "<="},
		{">=", ">="},
		{"between", "between"},
		{"", "="},
	}

	for _, tt := range tests {
		got := normalizeOperator(tt.input)
		if got != tt.want {
			t.Errorf("normalizeOperator(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNumericFactExtractor_Extract(t *testing.T) {
	e := NewNumericFactExtractor()
	if !e.IsConfigured() {
		t.Skip("LLM not configured")
	}

	ctx := context.Background()
	text := "В растворе содержание сульфатов составляет 250 мг/л, pH равен 8,5, а температура поддерживается в диапазоне от 40 до 60 °C."

	facts, err := e.Extract(ctx, text, "doc-1", "chunk-1")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(facts) == 0 {
		t.Fatal("expected at least one fact")
	}

	t.Logf("extracted facts: %+v", facts)
}
