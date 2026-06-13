package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ravikumar/mongodb-inspector/internal/service"
)

type ScannerWorker struct {
	scanner    *service.ScannerService
	queue      chan string
	wg         sync.WaitGroup
	maxRetries int
}

func NewScannerWorker(scanner *service.ScannerService) *ScannerWorker {
	return &ScannerWorker{
		scanner:    scanner,
		queue:      make(chan string, 16),
		maxRetries: 3,
	}
}

func (w *ScannerWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for scanID := range w.queue {
			ctx := context.Background()
			w.runWithRetry(ctx, scanID, 0)
		}
	}()
}

func (w *ScannerWorker) runWithRetry(ctx context.Context, scanID string, attempt int) {
	if err := w.scanner.RunScan(ctx, scanID); err != nil {
		if attempt < w.maxRetries {
			backoff := time.Duration(1<<attempt) * 2 * time.Second
			log.Printf("scan worker error for %s (attempt %d/%d): %v, retrying in %v",
				scanID, attempt+1, w.maxRetries+1, err, backoff)
			time.Sleep(backoff)
			w.runWithRetry(ctx, scanID, attempt+1)
		} else {
			log.Printf("scan worker failed for %s after %d attempts: %v", scanID, attempt+1, err)
		}
	}
}

func (w *ScannerWorker) Enqueue(scanID string) {
	w.queue <- scanID
}

func (w *ScannerWorker) Stop() {
	close(w.queue)
	w.wg.Wait()
}
