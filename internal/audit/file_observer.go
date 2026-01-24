package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/threading"
)

type FileAuditObserver struct {
	filePath string
	logger   logger.Logger
	pool     *threading.WorkerPool
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	errWg    sync.WaitGroup
}

func NewFileAuditObserver(ctx context.Context, filePath string, logger logger.Logger) *FileAuditObserver {
	observerCtx, cancel := context.WithCancel(ctx)
	observer := &FileAuditObserver{
		filePath: filePath,
		logger:   logger,
		pool:     threading.NewWorkerPoolWithQueueSize(config.AuditWorkerPoolSize, config.AuditEventChannelBuffer),
		ctx:      observerCtx,
		cancel:   cancel,
	}

	observer.pool.Start(observerCtx)
	observer.errWg.Add(1)
	go observer.handleErrors(observerCtx)

	return observer
}

func (f *FileAuditObserver) Process(ctx context.Context, event AuditEvent) {
	select {
	case <-f.ctx.Done():
		f.logger.Debugf("FileAuditObserver: observer context cancelled, dropping event")
		return
	case <-ctx.Done():
		f.logger.Debugf("FileAuditObserver: request context cancelled, dropping event")
		return
	default:
		eventCopy := event
		f.pool.AddJob(func() error {
			return f.writeEvent(eventCopy)
		})
	}
}

func (f *FileAuditObserver) handleErrors(ctx context.Context) {
	defer f.errWg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-f.pool.Errors():
			if !ok {
				return
			}
			if err != nil {
				f.logger.Errorf("FileAuditObserver: failed to write audit event: %v", err)
			}
		}
	}
}

func (f *FileAuditObserver) writeEvent(event AuditEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(jsonData); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	return nil
}

func (f *FileAuditObserver) Close() {
	f.once.Do(func() {
		if f.cancel != nil {
			f.cancel()
		}
		if f.pool != nil {
			f.pool.Stop()
		}
		f.errWg.Wait()
	})
}
