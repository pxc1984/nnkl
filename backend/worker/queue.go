package worker

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pxc1984/nnkl-backend/metrics"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

const (
	defaultWorkerCount   = 2
	defaultPollInterval  = time.Second
	defaultLeaseDuration = 2 * time.Hour
)

type OCRService interface {
	Parse(ctx context.Context, uploadID, inputBlobID, language string) error
}

type TextIndexer interface {
	IsConfigured() bool
	SendText(ctx context.Context, text, fileSource string) error
}

type Queue struct {
	store   store.Store
	ocr     OCRService
	indexer TextIndexer

	workerID      string
	workerCount   int
	pollInterval  time.Duration
	leaseDuration time.Duration
	wakeCh        chan struct{}

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func New(s store.Store, ocr OCRService, indexer TextIndexer, wakeBuffer int) *Queue {
	if wakeBuffer <= 0 {
		wakeBuffer = 1
	}

	return &Queue{
		store:         s,
		ocr:           ocr,
		indexer:       indexer,
		workerID:      buildWorkerID(),
		workerCount:   defaultWorkerCount,
		pollInterval:  defaultPollInterval,
		leaseDuration: defaultLeaseDuration,
		wakeCh:        make(chan struct{}, wakeBuffer),
		stopCh:        make(chan struct{}),
	}
}

func (q *Queue) Start() {
	if err := q.store.ReconcileUploadJobs(context.Background(), time.Now().UTC()); err != nil {
		slog.Error("upload queue reconcile failed", "error", err)
	} else {
		slog.Info("upload queue reconciled")
	}

	q.wg.Add(q.workerCount)
	for workerIndex := 0; workerIndex < q.workerCount; workerIndex++ {
		go q.workerLoop(workerIndex + 1)
	}
	slog.Info("worker queue started", "workers", q.workerCount, "worker_id", q.workerID)
	q.Notify()
}

func (q *Queue) Stop() {
	close(q.stopCh)
	q.wg.Wait()
	slog.Info("worker queue stopped")
}

func (q *Queue) Notify() {
	select {
	case q.wakeCh <- struct{}{}:
	default:
	}
}

func (q *Queue) workerLoop(workerIndex int) {
	defer q.wg.Done()
	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
		case <-q.wakeCh:
		}

		for {
			select {
			case <-q.stopCh:
				return
			default:
			}

			upload, err := q.store.ClaimNextUploadJob(
				context.Background(),
				q.claimOwner(workerIndex),
				q.leaseDuration,
			)
			if err != nil {
				slog.Error("worker queue claim failed", "worker", workerIndex, "error", err)
				break
			}
			if upload == nil {
				break
			}

			q.processUpload(context.Background(), *upload)
		}
	}
}

func (q *Queue) processUpload(ctx context.Context, upload models.Upload) {
	slog.Info("upload processing started", "upload_id", upload.ID, "type", upload.InputBlob.FileType)
	start := time.Now()

	switch upload.InputBlob.FileType {
	case "docx", "pptx":
		q.processSimple(ctx, upload)
	case "pdf":
		q.processOCR(ctx, upload)
	case "markdown":
		q.processMarkdown(ctx, upload)
	default:
		q.failUpload(ctx, upload.ID, "unsupported file type: "+upload.InputBlob.FileType)
	}

	duration := time.Since(start)
	metrics.UploadProcessingDuration.Observe(duration.Seconds())
	slog.Info("upload processing finished", "upload_id", upload.ID, "type", upload.InputBlob.FileType, "duration", duration)
}

func (q *Queue) processMarkdown(ctx context.Context, upload models.Upload) {
	outputBlobID := upload.InputBlobID
	completed := "completed"
	emptyErr := ""
	if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
		OutputBlobID: &outputBlobID,
		Status:       &completed,
		Error:        &emptyErr,
		ClearClaim:   true,
	}); err != nil {
		slog.Error("markdown finalize failed", "upload_id", upload.ID, "error", err)
	}
	metrics.UploadsTotal.WithLabelValues("completed").Inc()

	if q.indexer != nil && q.indexer.IsConfigured() {
		source := upload.ID + ".md"
		if err := q.indexer.SendText(ctx, string(upload.InputBlob.Content), source); err != nil {
			slog.Warn("markdown processing: lightrag send failed", "upload_id", upload.ID, "error", err)
		}
	}
}

func (q *Queue) processSimple(ctx context.Context, upload models.Upload) {
	markdown, err := extractText(upload.InputBlob.Content, upload.InputBlob.FileType)
	if err != nil {
		q.failUpload(ctx, upload.ID, err.Error())
		return
	}

	output := "<!-- source: " + upload.InputBlob.Filename + " -->\n\n" + markdown
	outBlob, err := q.store.CreateBlob(ctx, models.CreateBlobParams{
		Filename:    upload.InputBlob.Filename + ".md",
		FileType:    "markdown",
		ContentType: "text/markdown",
		SizeBytes:   int64(len(output)),
		Content:     []byte(output),
	})
	if err != nil {
		q.failUpload(ctx, upload.ID, "failed to create output blob: "+err.Error())
		return
	}

	completed := "completed"
	emptyErr := ""
	if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
		OutputBlobID: &outBlob.ID,
		Status:       &completed,
		Language:     &upload.Language,
		Error:        &emptyErr,
		ClearClaim:   true,
	}); err != nil {
		slog.Error("simple extraction finalize failed", "upload_id", upload.ID, "error", err)
		return
	}
	metrics.UploadsTotal.WithLabelValues("completed").Inc()

	if q.indexer != nil && q.indexer.IsConfigured() {
		source := upload.ID + ".md"
		if err := q.indexer.SendText(ctx, output, source); err != nil {
			slog.Warn("simple extraction: lightrag send failed", "upload_id", upload.ID, "error", err)
		}
	}
}

func (q *Queue) processOCR(ctx context.Context, upload models.Upload) {
	if err := q.ocr.Parse(ctx, upload.ID, upload.InputBlobID, upload.Language); err != nil {
		q.failUpload(ctx, upload.ID, "ocr parse failed: "+err.Error())
		return
	}

	updated, err := q.store.GetUploadByID(ctx, upload.ID)
	if err != nil {
		slog.Error("ocr processing: failed to reload upload", "upload_id", upload.ID, "error", err)
		return
	}

	completed := "completed"
	emptyErr := ""
	if updated.OutputBlobID != nil {
		if _, err := q.store.UpdateUpload(ctx, upload.ID, models.UpdateUploadParams{
			Status:     &completed,
			Error:      &emptyErr,
			ClearClaim: true,
		}); err != nil {
			slog.Error("ocr processing: failed to clear claim", "upload_id", upload.ID, "error", err)
		}
		metrics.UploadsTotal.WithLabelValues("completed").Inc()
	} else {
		q.failUpload(ctx, upload.ID, "ocr parse finished without output blob")
		return
	}

	if q.indexer != nil && q.indexer.IsConfigured() && updated.OutputBlob != nil && len(updated.OutputBlob.Content) > 0 {
		source := upload.ID + ".md"
		if err := q.indexer.SendText(ctx, string(updated.OutputBlob.Content), source); err != nil {
			slog.Warn("ocr processing: lightrag send failed", "upload_id", upload.ID, "error", err)
		}
	}
}

func (q *Queue) failUpload(ctx context.Context, uploadID, errorMessage string) {
	failed := "failed"
	if _, err := q.store.UpdateUpload(ctx, uploadID, models.UpdateUploadParams{
		Status:     &failed,
		Error:      &errorMessage,
		ClearClaim: true,
	}); err != nil {
		slog.Error("failed to update failed upload", "upload_id", uploadID, "error", err)
	}
	metrics.UploadsTotal.WithLabelValues("failed").Inc()
	slog.Error("upload processing failed", "upload_id", uploadID, "error", errorMessage)
}

func (q *Queue) claimOwner(workerIndex int) string {
	return q.workerID + ":" + strconv.Itoa(workerIndex)
}

func buildWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "backend"
	}
	return hostname + ":" + strconv.Itoa(os.Getpid())
}
