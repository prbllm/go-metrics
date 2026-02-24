package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/encryption"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/retry"
	"github.com/prbllm/go-metrics/internal/threading"
)

type Agent struct {
	client          *http.Client
	collector       *RuntimeMetricsCollector
	logger          logger.Logger
	pool            *threading.WorkerPool
	mu              sync.RWMutex
	runtimeMetrics  []model.Metrics
	gopsutilMetrics []model.Metrics
}

func NewAgent(client *http.Client, collector *RuntimeMetricsCollector, logger logger.Logger) *Agent {
	return &Agent{
		client:          client,
		collector:       collector,
		logger:          logger,
		runtimeMetrics:  []model.Metrics{},
		gopsutilMetrics: []model.Metrics{},
	}
}

func (a *Agent) Start(ctx context.Context) {
	cfg := config.GetConfig()
	a.logger.Infof("Starting agent with server host: %s and agent poll interval: %s and agent report interval: %s", cfg.ServerHost, cfg.AgentPollInterval, cfg.AgentReportInterval)
	if a.collector == nil {
		a.logger.Error("Collector is nil")
		return
	}

	a.pool = threading.NewWorkerPool(cfg.RateLimit)
	a.pool.Start(ctx)

	go a.handleErrors(ctx)

	pollTicker := time.NewTicker(cfg.AgentPollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(cfg.AgentReportInterval)
	defer reportTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				metrics := a.collector.CollectRuntimeMetrics()
				a.mu.Lock()
				a.runtimeMetrics = metrics
				a.mu.Unlock()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				metrics, err := a.collector.CollectGopsutilMetrics()
				if err != nil {
					a.logger.Errorf("Error collecting gopsutil metrics: %v", err)
					continue
				}
				a.mu.Lock()
				a.gopsutilMetrics = metrics
				a.mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Context done, starting graceful shutdown")

			pollTicker.Stop()
			reportTicker.Stop()

			a.mu.RLock()
			runtimeMetrics := make([]model.Metrics, len(a.runtimeMetrics))
			copy(runtimeMetrics, a.runtimeMetrics)
			gopsutilMetrics := make([]model.Metrics, len(a.gopsutilMetrics))
			copy(gopsutilMetrics, a.gopsutilMetrics)
			a.mu.RUnlock()

			combinedMetrics := model.CombineMetrics(runtimeMetrics, gopsutilMetrics)

			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), config.ShutdownTimeout)
			defer cancelShutdown()

			if len(combinedMetrics) > 0 {
				if err := a.SendMetricsBatchJSON(shutdownCtx, combinedMetrics); err != nil {
					a.logger.Errorf("Failed to send final metrics batch during shutdown: %v", err)
				}
			}

			a.pool.StopAndDrain(shutdownCtx)
			return
		case <-reportTicker.C:
			a.mu.RLock()
			runtimeMetrics := make([]model.Metrics, len(a.runtimeMetrics))
			copy(runtimeMetrics, a.runtimeMetrics)
			gopsutilMetrics := make([]model.Metrics, len(a.gopsutilMetrics))
			copy(gopsutilMetrics, a.gopsutilMetrics)
			a.mu.RUnlock()

			if len(runtimeMetrics) > 0 || len(gopsutilMetrics) > 0 {
				combinedMetrics := model.CombineMetrics(runtimeMetrics, gopsutilMetrics)
				if len(combinedMetrics) > 0 {
					a.pool.AddJob(func() error {
						return a.SendMetricsBatchJSON(context.Background(), combinedMetrics)
					})
				}
			}
		}
	}
}

func (a *Agent) handleErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-a.pool.Errors():
			if err != nil {
				a.logger.Errorf("Error from worker pool: %v", err)
			}
		}
	}
}

func (a *Agent) sendMetrics(ctx context.Context, metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}

	for _, metric := range metrics {
		reqCtx, cancel := context.WithTimeout(ctx, config.HTTPRequestTimeout)

		url, err := a.generateURL(metric)
		if err != nil {
			cancel()
			a.logger.Warnf("Error generating url: %v. Skipping...", err)
			continue
		}
		a.logger.Debugf("Sending metric: %s to url: %s", metric.String(), url)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBufferString(""))
		if err != nil {
			cancel()
			a.logger.Errorf("Error creating request: %v. Skipping...", err)
			continue
		}
		req.Header.Set(config.ContentTypeHeader, config.ContentTypeTextPlain)
		response, err := a.client.Do(req)
		cancel()
		if err != nil {
			a.logger.Errorf("Error sending metric: %v. Skipping...", err)
			continue
		}
		a.logger.Debugf("Response: %s", response.Status)
		response.Body.Close()
	}
	return nil
}

func (a *Agent) generateURL(metric model.Metrics) (string, error) {
	var value string

	if metric.MType == model.Counter {
		if metric.Delta == nil {
			return "", fmt.Errorf("metric %s has no delta", metric.ID)
		}
		value = fmt.Sprintf("%d", *metric.Delta)
	} else {
		if metric.Value == nil {
			return "", fmt.Errorf("metric %s has no value", metric.ID)
		}
		value = fmt.Sprintf("%f", *metric.Value)
	}

	baseURL := a.getBaseURL()
	return fmt.Sprintf("%s%s/%s/%s/%s", baseURL, config.UpdatePath, metric.MType, metric.ID, value), nil
}

func (a *Agent) compressJSON(jsonData []byte) ([]byte, error) {
	return compression.CompressData(jsonData)
}

func (a *Agent) SendMetricsJSON(ctx context.Context, metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}
	type encryptedPayload struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	for _, metric := range metrics {
		reqCtx, cancel := context.WithTimeout(ctx, config.HTTPRequestTimeout)

		jsonData, err := json.Marshal(metric)
		if err != nil {
			cancel()
			a.logger.Warnf("Error marshaling metric to JSON: %v. Skipping...", err)
			continue
		}

		updateURL := a.getBaseURL() + config.UpdatePath
		cfg := config.GetConfig()

		response, err := retry.RetryWithBackoffHTTP(reqCtx, a.logger, func() (*http.Response, error) {
			var body []byte

			if cfg.CryptoKey != "" {
				encryptedKey, ciphertext, err := encryption.EncryptHybrid(cfg.CryptoKey, jsonData)
				if err != nil {
					return nil, fmt.Errorf("error encrypting JSON data: %w", err)
				}
				payload := encryptedPayload{
					Key:  base64.StdEncoding.EncodeToString(encryptedKey),
					Data: base64.StdEncoding.EncodeToString(ciphertext),
				}
				body, err = json.Marshal(payload)
				if err != nil {
					return nil, fmt.Errorf("error marshaling encrypted payload: %w", err)
				}
				a.logger.Info("Sending metric via encrypted JSON")
			} else {
				compressedData, err := a.compressJSON(jsonData)
				if err != nil {
					return nil, fmt.Errorf("error compressing JSON data: %w", err)
				}

				stats := compression.GetCompressionStats(jsonData, compressedData)
				a.logger.Info("Sending metric via compressed JSON")
				a.logger.Debugf("Compression stats: original=%d bytes, compressed=%d bytes, ratio=%.2f",
					stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio)

				body = compressedData
			}

			req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, updateURL, bytes.NewBuffer(body))
			if err != nil {
				return nil, fmt.Errorf("error creating request: %w", err)
			}

			req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
			if cfg.CryptoKey == "" {
				req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)
			}

			if cfg.Key != "" {
				hashValue := hash.ComputeHash(cfg.Key, jsonData)
				req.Header.Set(config.HashSHA256Header, hashValue)
			}

			return a.client.Do(req)
		})
		cancel()
		if err != nil {
			a.logger.Errorf("Error sending metric via JSON: %v. Skipping...", err)
			continue
		}

		a.logger.Debugf("JSON Response: %s", response.Status)
		response.Body.Close()
	}
	return nil
}

func (a *Agent) SendMetricsBatchJSON(ctx context.Context, metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}

	if len(metrics) == 0 {
		a.logger.Debug("Skipping empty metrics batch")
		return nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, config.HTTPRequestTimeout)
	defer cancel()

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshaling metrics batch to JSON: %w", err)
	}

	batchURL := a.getBatchURL()
	cfg := config.GetConfig()

	response, err := retry.RetryWithBackoffHTTP(reqCtx, a.logger, func() (*http.Response, error) {
		var body []byte

		if cfg.CryptoKey != "" {
			encryptedKey, ciphertext, err := encryption.EncryptHybrid(cfg.CryptoKey, jsonData)
			if err != nil {
				return nil, fmt.Errorf("error encrypting JSON data: %w", err)
			}
			payload := struct {
				Key  string `json:"key"`
				Data string `json:"data"`
			}{
				Key:  base64.StdEncoding.EncodeToString(encryptedKey),
				Data: base64.StdEncoding.EncodeToString(ciphertext),
			}
			body, err = json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("error marshaling encrypted payload: %w", err)
			}
			a.logger.Infof("Sending batch of %d metrics via encrypted JSON", len(metrics))
		} else {
			compressedData, err := a.compressJSON(jsonData)
			if err != nil {
				return nil, fmt.Errorf("error compressing JSON data: %w", err)
			}

			stats := compression.GetCompressionStats(jsonData, compressedData)
			a.logger.Infof("Sending batch of %d metrics via compressed JSON", len(metrics))
			a.logger.Debugf("Compression stats: original=%d bytes, compressed=%d bytes, ratio=%.2f",
				stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio)

			body = compressedData
		}

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, batchURL, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("error creating batch request: %w", err)
		}

		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		if cfg.CryptoKey == "" {
			req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)
		}

		if cfg.Key != "" {
			hashValue := hash.ComputeHash(cfg.Key, jsonData)
			req.Header.Set(config.HashSHA256Header, hashValue)
		}

		return a.client.Do(req)
	})
	if err != nil {
		return fmt.Errorf("error sending metrics batch: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	a.logger.Debugf("Batch Response: %s", response.Status)
	return nil
}
func (a *Agent) getBaseURL() string {
	return fmt.Sprintf("http://%s", config.GetConfig().ServerHost)
}

func (a *Agent) getBatchURL() string {
	return a.getBaseURL() + config.UpdatesPath
}
