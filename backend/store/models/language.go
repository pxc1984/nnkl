package models

import (
	"strings"
	"unicode"
)

// ResolveUploadLanguage returns an explicit upload language or detects it from
// extracted text for legacy uploads configured with automatic OCR language.
func ResolveUploadLanguage(upload *Upload) string {
	if upload == nil {
		return "auto"
	}
	language := strings.ToLower(strings.TrimSpace(upload.Language))
	if language == "ru" || language == "en" {
		return language
	}

	var content []byte
	if upload.OutputBlob != nil {
		content = upload.OutputBlob.Content
	} else if upload.InputBlob.FileType == "markdown" {
		content = upload.InputBlob.Content
	}
	if len(content) == 0 {
		return "auto"
	}

	var cyrillic, latin int
	for _, r := range string(content) {
		switch {
		case unicode.In(r, unicode.Cyrillic):
			cyrillic++
		case unicode.In(r, unicode.Latin):
			latin++
		}
	}
	if cyrillic >= 20 && cyrillic >= latin/2 {
		return "ru"
	}
	if latin >= 20 {
		return "en"
	}
	return "auto"
}
