package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
)

type FileAuditObserver struct {
	filePath string
	logger   logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	eventCh  chan AuditEvent
	writerWg sync.WaitGroup
	errWg    sync.WaitGroup
	writeMu  sync.Mutex
}

func NewFileAuditObserver(ctx context.Context, filePath string, logger logger.Logger) *FileAuditObserver {
	observerCtx, cancel := context.WithCancel(ctx)
	observer := &FileAuditObserver{
		filePath: filePath,
		logger:   logger,
		ctx:      observerCtx,
		cancel:   cancel,
		eventCh:  make(chan AuditEvent, config.AuditEventChannelBuffer),
	}

	observer.writerWg.Add(1)
	go observer.writerLoop(observerCtx)

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
		if event.Timestamp == 0 {
			event.Timestamp = time.Now().Unix()
		}
		select {
		case f.eventCh <- event:
		case <-f.ctx.Done():
			f.logger.Debugf("FileAuditObserver: observer context cancelled, dropping event")
		case <-ctx.Done():
			f.logger.Debugf("FileAuditObserver: request context cancelled, dropping event")
		}
	}
}

func (f *FileAuditObserver) writerLoop(ctx context.Context) {
	defer f.writerWg.Done()

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.logger.Errorf("FileAuditObserver: failed to open file: %v", err)
		return
	}
	defer file.Close()

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case event, ok := <-f.eventCh:
					if !ok {
						return
					}
					if err := f.writeEvent(file, event); err != nil {
						f.logger.Errorf("FileAuditObserver: failed to write audit event: %v", err)
					}
				default:
					return
				}
			}
		case event, ok := <-f.eventCh:
			if !ok {
				return
			}
			if err := f.writeEvent(file, event); err != nil {
				f.logger.Errorf("FileAuditObserver: failed to write audit event: %v", err)
			}
		}
	}
}

func (f *FileAuditObserver) writeEvent(file *os.File, event AuditEvent) error {
	f.writeMu.Lock()
	defer f.writeMu.Unlock()

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := file.Write(jsonData); err != nil {
		return err
	}
	if _, err := file.WriteString("\n"); err != nil {
		return err
	}

	return nil
}

func (f *FileAuditObserver) handleErrors(ctx context.Context) {
	defer f.errWg.Done()
	<-ctx.Done()
}

func (f *FileAuditObserver) Close() {
	f.once.Do(func() {
		if f.cancel != nil {
			f.cancel()
		}
		done := make(chan struct{})
		go func() {
			f.writerWg.Wait()
			close(f.eventCh)
			f.errWg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			f.logger.Warnf("FileAuditObserver: Close() timed out waiting for goroutines")
			close(f.eventCh)
		}
	})
}
