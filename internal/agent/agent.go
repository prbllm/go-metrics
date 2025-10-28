package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
)

type Agent struct {
	client         *http.Client
	collector      *RuntimeMetricsCollector
	route          string
	pollInterval   time.Duration
	reportInterval time.Duration
	logger         logger.Logger
}

func NewAgent(client *http.Client, collector *RuntimeMetricsCollector, route string, pollInterval time.Duration, reportInterval time.Duration, logger logger.Logger) *Agent {
	return &Agent{
		client:         client,
		collector:      collector,
		route:          route,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		logger:         logger,
	}
}

func (a *Agent) Start(context context.Context) {
	a.logger.Infof("Starting agent with route: %s and agent poll interval: %s and agent report interval: %s", a.route, a.pollInterval, a.reportInterval)
	if a.collector == nil {

		a.logger.Error("Collector is nil")
		return
	}

	collectCounter := int(a.reportInterval / a.pollInterval)
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
			metrics = a.collector.Collect()
			time.Sleep(a.pollInterval)
		}
		err := a.SendMetricsJSON(metrics)
		if err != nil {
			a.logger.Errorf("Error sending metrics: %v", err)
		}
	}
}

func (a *Agent) sendMetrics(metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}

	for _, metric := range metrics {
		url, err := a.generateURL(metric)
		if err != nil {
			a.logger.Warnf("Error generating url: %v. Skipping...", err)
			continue
		}
		a.logger.Debugf("Sending metric: %s to url: %s", metric.String(), url)
		response, err := a.client.Post(url, config.ContentTypeTextPlain, strings.NewReader(""))
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

	url := a.route
	if url[len(url)-1] != '/' {
		url += "/"
	}
	return fmt.Sprintf("%s%s/%s/%s", url, metric.MType, metric.ID, value), nil
}

func (a *Agent) compressJSON(jsonData []byte) ([]byte, error) {
	return compression.CompressData(jsonData)
}

func (a *Agent) SendMetricsJSON(metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}
	for _, metric := range metrics {
		jsonData, err := json.Marshal(metric)
		if err != nil {
			a.logger.Warnf("Error marshaling metric to JSON: %v. Skipping...", err)
			continue
		}

		compressedData, err := a.compressJSON(jsonData)
		if err != nil {
			a.logger.Warnf("Error compressing JSON data: %v. Skipping...", err)
			continue
		}

		stats := compression.GetCompressionStats(jsonData, compressedData)
		a.logger.Info("Sending metric via compressed JSON")
		a.logger.Debugf("Compression stats: original=%d bytes, compressed=%d bytes, ratio=%.2f",
			stats.OriginalSize, stats.CompressedSize, stats.CompressionRatio)

		req, err := http.NewRequest(http.MethodPost, a.route, bytes.NewBuffer(compressedData))
		if err != nil {
			a.logger.Errorf("Error creating request: %v. Skipping...", err)
			continue
		}

		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

		response, err := a.client.Do(req)
		if err != nil {
			a.logger.Errorf("Error sending metric via JSON: %v. Skipping...", err)
			continue
		}

		a.logger.Debugf("JSON Response: %s", response.Status)
		response.Body.Close()
	}
	return nil
}
