package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/service"
)

type Worker struct {
	cfg          *config.Config
	provisioning *service.ProvisioningService
	logger       *slog.Logger
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewWorker(
	cfg *config.Config,
	provisioning *service.ProvisioningService,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		cfg:          cfg,
		provisioning: provisioning,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	if !w.cfg.WorkerEnabled {
		w.logger.Info("background worker is disabled by configuration")
		return
	}

	w.logger.Info("starting background worker pool",
		"concurrency", w.cfg.WorkerConcurrency,
		"pollInterval", w.cfg.WorkerPollInterval,
	)

	w.wg.Add(1)
	go w.runProvisioningLoop(ctx)
}

func (w *Worker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("background worker pool stopped")
}

func (w *Worker) runProvisioningLoop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.cfg.WorkerPollInterval)
	defer ticker.Stop()

	// Semaphore to bound concurrent job execution
	sem := make(chan struct{}, w.cfg.WorkerConcurrency)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollAndProcess(ctx, sem)
		}
	}
}

func (w *Worker) pollAndProcess(ctx context.Context, sem chan struct{}) {
	batchSize := w.cfg.WorkerConcurrency * 2
	processed, err := w.provisioning.ProcessPendingBatch(ctx, batchSize)
	if err != nil {
		w.logger.Error("error processing provisioning batch", "error", err)
	} else if processed > 0 {
		w.logger.Info("processed provisioning batch", "count", processed)
	}
}
