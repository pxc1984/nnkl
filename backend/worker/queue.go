package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

// OCRService is the interface for calling the OCR parse endpoint.
type OCRService interface {
	Parse(ctx context.Context, uploadID, inputBlobID, language string) error
}

// TextIndexer is the interface for sending extracted text to LightRAG.
type TextIndexer interface {
	IsConfigured() bool
	SendText(ctx context.Context, text, fileSource string) error
}

// Job describes a document to process.
type Job struct {
	UploadID     string
	OutputFormat string
	Language     string
	FileType     string // "docx", "pptx", or "pdf"
}

// Queue manages background processing of uploaded documents.
type Queue struct {
	store   store.Store
	ocr     OCRService
	indexer TextIndexer

	simpleJobs chan Job // docx, pptx — direct text extraction
	ocrJobs    chan Job // pdf — send to OCR service

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a new processing queue. bufferSize controls the channel buffer per worker.
func New(s store.Store, ocr OCRService, indexer TextIndexer, bufferSize int) *Queue {
	return &Queue{
		store:      s,
		ocr:        ocr,
		indexer:    indexer,
		simpleJobs: make(chan Job, bufferSize),
		ocrJobs:    make(chan Job, bufferSize),
		stopCh:     make(chan struct{}),
	}
}

// Start launches the two worker goroutines.
func (q *Queue) Start() {
	q.wg.Add(2)
	go q.simpleExtractor()
	go q.ocrProcessor()
	slog.Info("worker queue started")
}

// Stop gracefully shuts down the workers, waiting for in-flight jobs to finish.
func (q *Queue) Stop() {
	close(q.stopCh)
	q.wg.Wait()
	slog.Info("worker queue stopped")
}

// EnqueueSimple enqueues a job for direct text extraction (docx, pptx).
func (q *Queue) EnqueueSimple(job Job) {
	select {
	case q.simpleJobs <- job:
	case <-q.stopCh:
	}
}

// EnqueueOCR enqueues a job for OCR service processing (pdf).
func (q *Queue) EnqueueOCR(job Job) {
	select {
	case q.ocrJobs <- job:
	case <-q.stopCh:
	}
}

// simpleExtractor handles docx and pptx files: extracts text directly from the
// blob content using Go's standard library (archive/zip + encoding/xml),
// creates an output blob with the markdown result, updates the upload, and
// optionally sends to LightRAG.
func (q *Queue) simpleExtractor() {
	defer q.wg.Done()
	for {
		select {
		case <-q.stopCh:
			return
		case job := <-q.simpleJobs:
			q.processSimple(context.Background(), job)
		}
	}
}

func (q *Queue) processSimple(ctx context.Context, job Job) {
	slog.Info("simple extraction started", "upload_id", job.UploadID, "type", job.FileType)
	start := time.Now()

	upload, err := q.store.GetUploadByID(ctx, job.UploadID)
	if err != nil {
		slog.Error("simple extraction: failed to get upload", "upload_id", job.UploadID, "error", err)
		return
	}

	// Mark as processing.
	status := "processing"
	if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{Status: &status}); err != nil {
		slog.Error("simple extraction: failed to set processing status", "upload_id", job.UploadID, "error", err)
		return
	}

	// Extract text from the input blob content.
	markdown, err := extractText(upload.InputBlob.Content, job.FileType)
	if err != nil {
		errMsg := err.Error()
		failed := "failed"
		if _, uErr := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
			Status: &failed,
			Error:  &errMsg,
		}); uErr != nil {
			slog.Error("simple extraction: failed to set failed status", "upload_id", job.UploadID, "error", uErr)
		}
		slog.Error("simple extraction failed", "upload_id", job.UploadID, "error", err)
		return
	}

	// Wrap with source comment — matches OCR service convention.
	source := job.UploadID + ".md"
	output := "<!-- source: " + upload.InputBlob.Filename + " -->\n\n" + markdown

	// Create output blob.
	outBlob, err := q.store.CreateBlob(ctx, models.CreateBlobParams{
		Filename:    upload.InputBlob.Filename + ".md",
		FileType:    "markdown",
		ContentType: "text/markdown",
		SizeBytes:   int64(len(output)),
		Content:     []byte(output),
	})
	if err != nil {
		errMsg := "failed to create output blob: " + err.Error()
		failed := "failed"
		if _, uErr := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
			Status: &failed,
			Error:  &errMsg,
		}); uErr != nil {
			slog.Error("simple extraction: failed to set failed status", "upload_id", job.UploadID, "error", uErr)
		}
		slog.Error("simple extraction: create output blob failed", "upload_id", job.UploadID, "error", err)
		return
	}

	// Update upload as completed with output blob.
	completed := "completed"
	emptyErr := ""
	if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
		OutputBlobID: &outBlob.ID,
		Status:       &completed,
		Language:     &job.Language,
		Error:        &emptyErr,
	}); err != nil {
		slog.Error("simple extraction: failed to finalize upload", "upload_id", job.UploadID, "error", err)
		return
	}

	slog.Info("simple extraction completed", "upload_id", job.UploadID,
		"type", job.FileType, "duration", time.Since(start))

	// Send to LightRAG if configured.
	if q.indexer != nil && q.indexer.IsConfigured() {
		if err := q.indexer.SendText(ctx, output, source); err != nil {
			slog.Warn("simple extraction: lightrag send failed", "upload_id", job.UploadID, "error", err)
		}
	}
}

// ocrProcessor handles pdf files: delegates to the OCR service via HTTP.
func (q *Queue) ocrProcessor() {
	defer q.wg.Done()
	for {
		select {
		case <-q.stopCh:
			return
		case job := <-q.ocrJobs:
			q.processOCR(context.Background(), job)
		}
	}
}

func (q *Queue) processOCR(ctx context.Context, job Job) {
	slog.Info("ocr processing started", "upload_id", job.UploadID)
	start := time.Now()

	upload, err := q.store.GetUploadByID(ctx, job.UploadID)
	if err != nil {
		slog.Error("ocr processing: failed to get upload", "upload_id", job.UploadID, "error", err)
		return
	}

	// Mark as processing.
	status := "processing"
	if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{Status: &status}); err != nil {
		slog.Error("ocr processing: failed to set processing status", "upload_id", job.UploadID, "error", err)
		return
	}

	// Call OCR service. It reads the input blob and updates the upload directly in the DB.
	if err := q.ocr.Parse(ctx, upload.ID, upload.InputBlobID, job.Language); err != nil {
		errMsg := "ocr parse failed: " + err.Error()
		failed := "failed"
		if _, uErr := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
			Status: &failed,
			Error:  &errMsg,
		}); uErr != nil {
			slog.Error("ocr processing: failed to set failed status", "upload_id", job.UploadID, "error", uErr)
		}
		slog.Error("ocr processing failed", "upload_id", job.UploadID, "error", err)
		return
	}

	slog.Info("ocr processing completed", "upload_id", job.UploadID,
		"duration", time.Since(start))

	// Optionally send to LightRAG after OCR is done.
	if q.indexer != nil && q.indexer.IsConfigured() {
		// Re-fetch to get the output blob set by the OCR service.
		updated, err := q.store.GetUploadByID(ctx, job.UploadID)
		if err != nil {
			slog.Warn("ocr processing: failed to re-fetch upload for lightrag", "upload_id", job.UploadID, "error", err)
			return
		}
		if updated.OutputBlob != nil && len(updated.OutputBlob.Content) > 0 {
			source := job.UploadID + ".md"
			if err := q.indexer.SendText(ctx, string(updated.OutputBlob.Content), source); err != nil {
				slog.Warn("ocr processing: lightrag send failed", "upload_id", job.UploadID, "error", err)
			}
		}
	}
}
