package models

import "testing"

func TestResolveUploadLanguage(t *testing.T) {
	tests := []struct {
		name     string
		upload   Upload
		expected string
	}{
		{name: "explicit Russian", upload: Upload{Language: "ru"}, expected: "ru"},
		{name: "explicit English", upload: Upload{Language: "EN"}, expected: "en"},
		{
			name:     "detect Russian",
			upload:   Upload{Language: "auto", OutputBlob: &Blob{Content: []byte("Исследование технологий переработки никеля и медных руд на российских предприятиях.")}},
			expected: "ru",
		},
		{
			name:     "detect English",
			upload:   Upload{Language: "auto", OutputBlob: &Blob{Content: []byte("Research into nickel processing and copper extraction technologies for industrial facilities.")}},
			expected: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveUploadLanguage(&tt.upload); got != tt.expected {
				t.Fatalf("ResolveUploadLanguage() = %q, want %q", got, tt.expected)
			}
		})
	}
}
