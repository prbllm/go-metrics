package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/retry"
)

type Agent struct {
	client    *http.Client
	collector *RuntimeMetricsCollector
	logger    logger.Logger
}

func NewAgent(client *http.Client, collector *RuntimeMetricsCollector, logger logger.Logger) *Agent {
	return &Agent{
		client:    client,
		collector: collector,
		logger:    logger,
	}
}

func (a *Agent) Start(context context.Context) {
	cfg := config.GetConfig()
	a.logger.Infof("Starting agent with server host: %s and agent poll interval: %s and agent report interval: %s", cfg.ServerHost, cfg.AgentPollInterval, cfg.AgentReportInterval)
	if a.collector == nil {

		a.logger.Error("Collector is nil")
		return
	}

	collectCounter := int(cfg.AgentReportInterval / cfg.AgentPollInterval)
	for {
		select {
		case <-context.Done():
			a.logger.Info("Context done")
			return
		default:
		}

		metrics := []model.Metrics{}
		for range collectCounter {
			select {
			case <-context.Done():
				a.logger.Info("Context done")
				return
			default:
			}
			metrics = model.CombineMetrics(metrics, a.collector.CollectRuntimeMetrics())
			time.Sleep(cfg.AgentPollInterval)
		}
		err := a.SendMetricsJSON(context, metrics)
		if err != nil {
			a.logger.Errorf("Error sending metrics: %v", err)
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
	for _, metric := range metrics {
		reqCtx, cancel := context.WithTimeout(ctx, config.HTTPRequestTimeout)

		jsonData, err := json.Marshal(metric)
		if err != nil {
			cancel()
			a.logger.Warnf("Error marshaling metric to JSON: %v. Skipping...", err)
			continue
		}

		compressedData, err := a.compressJSON(jsonData)
		if err != nil {
			cancel()
			a.logger.Warnf("Error compressing JSON data: %v. Skipping...", err)
			continue
		}

		stats := compression.GetCompressionStats(jsonData, compressedData)
		a.logger.Info("Sending metric via compressed JSON")
		a.logger.Debugf("Compression stats: original=%d bytes, compressed=%d bytes, ratio=%.2f",
			stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio)

		updateURL := a.getBaseURL() + config.UpdatePath
		cfg := config.GetConfig()

		response, err := retry.RetryWithBackoffHTTP(reqCtx, a.logger, func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, updateURL, bytes.NewBuffer(compressedData))
			if err != nil {
				return nil, fmt.Errorf("error creating request: %w", err)
			}

			req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
			req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

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

	compressedData, err := a.compressJSON(jsonData)
	if err != nil {
		return fmt.Errorf("error compressing JSON data: %w", err)
	}

	stats := compression.GetCompressionStats(jsonData, compressedData)
	a.logger.Infof("Sending batch of %d metrics via compressed JSON", len(metrics))
	a.logger.Debugf("Compression stats: original=%d bytes, compressed=%d bytes, ratio=%.2f",
		stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio)

	batchURL := a.getBatchURL()
	cfg := config.GetConfig()

	response, err := retry.RetryWithBackoffHTTP(reqCtx, a.logger, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, batchURL, bytes.NewBuffer(compressedData))
		if err != nil {
			return nil, fmt.Errorf("error creating batch request: %w", err)
		}

		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

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
