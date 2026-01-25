package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	cancel context.CancelFunc
	once   sync.Once
	errWg  sync.WaitGroup
}

func NewURLAuditObserver(ctx context.Context, auditURL string, logger logger.Logger) (*URLAuditObserver, error) {
	if auditURL == "" {
		return nil, fmt.Errorf("audit URL cannot be empty")
	}

	parsedURL, err := url.Parse(auditURL)
	if err != nil {
		return nil, fmt.Errorf("invalid audit URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("audit URL must use http or https scheme")
	}

	observerCtx, cancel := context.WithCancel(ctx)
	observer := &URLAuditObserver{
		url: auditURL,
		client: &http.Client{
			Timeout: config.HTTPRequestTimeout,
			Transport: &http.Transport{
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				ResponseHeaderTimeout: config.HTTPRequestTimeout,
			},
		},
		logger: logger,
		pool:   threading.NewWorkerPoolWithQueueSize(config.AuditWorkerPoolSize, config.AuditEventChannelBuffer),
		ctx:    observerCtx,
		cancel: cancel,
	}

	observer.pool.Start(observerCtx)
	observer.errWg.Add(1)
	go observer.handleErrors(observerCtx)

	return observer, nil
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
		if u.cancel != nil {
			u.cancel()
		}
		if u.pool != nil {
			u.pool.Stop()
		}
		done := make(chan struct{})
		go func() {
			u.errWg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			u.logger.Warnf("URLAuditObserver: Close() timed out waiting for error handler")
		}
		if u.client != nil {
			if transport, ok := u.client.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
		}
	})
}
