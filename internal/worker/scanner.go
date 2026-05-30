package worker

import (
	"context"
	"log"
	"sync"

	"github.com/ravikumar/mongodb-inspector/internal/service"
)

type ScannerWorker struct {
	scanner *service.ScannerService
	queue   chan string
	wg      sync.WaitGroup
}

func NewScannerWorker(scanner *service.ScannerService) *ScannerWorker {
	return &ScannerWorker{
		scanner: scanner,
		queue:   make(chan string, 16),
	}
}

func (w *ScannerWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for scanID := range w.queue {
			ctx := context.Background()
			if err := w.scanner.RunScan(ctx, scanID); err != nil {
				log.Printf("scan worker error for %s: %v", scanID, err)
			}
		}
	}()
}

func (w *ScannerWorker) Enqueue(scanID string) {
	w.queue <- scanID
}

func (w *ScannerWorker) Stop() {
	close(w.queue)
	w.wg.Wait()
}
