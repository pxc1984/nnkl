package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func AuditMiddleware() gin.HandlerFunc {
	redactedHeaders := map[string]struct{}{
		"authorization": {},
		"cookie":        {},
		"set-cookie":    {},
	}

	return func(c *gin.Context) {
		start := time.Now()

		// Read and restore request body so downstream handlers can read it
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		// Wrap response writer to capture response body
		bw := &bodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = bw

		c.Next()

		elapsed := time.Since(start)

		// Collect headers (redact sensitive ones)
		headers := make(map[string]string)
		for k, v := range c.Request.Header {
			if len(v) == 0 {
				continue
			}
			key := k
			if _, ok := redactedHeaders[key]; ok {
				headers[key] = "[redacted]"
			} else {
				headers[key] = v[0]
			}
		}

		var reqJSON *string
		if len(reqBody) > 0 && json.Valid(reqBody) {
			s := string(reqBody)
			reqJSON = &s
		}

		var respJSON *string
		if bw.body.Len() > 0 && json.Valid(bw.body.Bytes()) {
			s := bw.body.String()
			respJSON = &s
		}

		var headersJSON *string
		if len(headers) > 0 {
			h, err := json.Marshal(headers)
			if err == nil {
				s := string(h)
				headersJSON = &s
			}
		}

		entry := &models.AuditLog{
			Timedate:     start,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			RemoteIP:     c.ClientIP(),
			RemoteAgent:  c.Request.UserAgent(),
			ResponseTime: elapsed.Milliseconds(),
			StatusCode:   c.Writer.Status(),
			RequestJSON:  reqJSON,
			ResponseJSON: respJSON,
			Headers:      headersJSON,
		}

		st := store.GetStore()
		if err := st.CreateAuditLog(c.Request.Context(), entry); err != nil {
			slog.Debug("audit log write failed", "error", err)
		}
	}
}
