package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/threading"
)

type URLAuditObserver struct {
	url    string
	client *http.Client
	logger logger.Logger
	pool   *threading.WorkerPool
	ctx    context.Context
	once   sync.Once
	errWg  sync.WaitGroup
}

func NewURLAuditObserver(ctx context.Context, url string, logger logger.Logger) *URLAuditObserver {
	observer := &URLAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: config.HTTPRequestTimeout,
		},
		logger: logger,
		pool:   threading.NewWorkerPoolWithQueueSize(config.AuditWorkerPoolSize, config.AuditEventChannelBuffer),
		ctx:    ctx,
	}

	observer.pool.Start(ctx)
	observer.errWg.Add(1)
	go observer.handleErrors(ctx)

	return observer
}

func (u *URLAuditObserver) Process(ctx context.Context, event AuditEvent) {
	select {
	case <-u.ctx.Done():
		u.logger.Debugf("URLAuditObserver: observer context cancelled, dropping event")
		return
	case <-ctx.Done():
		u.logger.Debugf("URLAuditObserver: request context cancelled, dropping event")
		return
	default:
		eventCopy := event
		u.pool.AddJob(func() error {
			reqCtx, cancel := context.WithTimeout(u.ctx, config.HTTPRequestTimeout)
			defer cancel()

			return u.sendEvent(reqCtx, eventCopy)
		})
	}
}

func (u *URLAuditObserver) sendEvent(ctx context.Context, event AuditEvent) error {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("received error status %d from audit endpoint", resp.StatusCode)
	}

	return nil
}

func (u *URLAuditObserver) handleErrors(ctx context.Context) {
	defer u.errWg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-u.pool.Errors():
			if !ok {
				return
			}
			if err != nil {
				u.logger.Errorf("URLAuditObserver: failed to send audit event: %v", err)
			}
		}
	}
}

func (u *URLAuditObserver) Close() {
	u.once.Do(func() {
		if u.pool != nil {
			u.pool.Stop()
		}
		u.errWg.Wait()
	})
}
