package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
)

type Agent struct {
	client         *http.Client
	collector      *RuntimeMetricsCollector
	route          string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewAgent(client *http.Client, collector *RuntimeMetricsCollector, route string, pollInterval time.Duration, reportInterval time.Duration) *Agent {
	return &Agent{
		client:         client,
		collector:      collector,
		route:          route,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

func (a *Agent) Start(context context.Context) {
	config.GetLogger().Infof("Starting agent with route: %s and agent poll interval: %s and agent report interval: %s", a.route, a.pollInterval, a.reportInterval)
	if a.collector == nil {

		config.GetLogger().Error("Collector is nil")
		return
	}

	collectCounter := int(a.reportInterval / a.pollInterval)
	for {
		select {
		case <-context.Done():
			config.GetLogger().Info("Context done")
			return
		default:
		}

		metrics := []model.Metrics{}
		for range collectCounter {
			select {
			case <-context.Done():
				config.GetLogger().Info("Context done")
				return
			default:
			}
			metrics = a.collector.Collect()
			time.Sleep(a.pollInterval)
		}
		err := a.SendMetricsJSON(metrics)
		if err != nil {
			config.GetLogger().Errorf("Error sending metrics: %v", err)
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
			config.GetLogger().Warnf("Error generating url: %v. Skipping...", err)
			continue
		}
		config.GetLogger().Debugf("Sending metric: %s to url: %s", metric.String(), url)
		response, err := a.client.Post(url, config.ContentTypeTextPlain, strings.NewReader(""))
		if err != nil {
			config.GetLogger().Errorf("Error sending metric: %v. Skipping...", err)
			continue
		}
		config.GetLogger().Debugf("Response: %s", response.Status)
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
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	_, err := gzWriter.Write(jsonData)
	if err != nil {
		gzWriter.Close()
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (a *Agent) SendMetricsJSON(metrics []model.Metrics) error {
	if a.client == nil {
		return fmt.Errorf("client is nil")
	}
	for _, metric := range metrics {
		jsonData, err := json.Marshal(metric)
		if err != nil {
			config.GetLogger().Warnf("Error marshaling metric to JSON: %v. Skipping...", err)
			continue
		}

		compressedData, err := a.compressJSON(jsonData)
		if err != nil {
			config.GetLogger().Warnf("Error compressing JSON data: %v. Skipping...", err)
			continue
		}

		config.GetLogger().Info("Sending metric via compressed JSON")
		config.GetLogger().Debugf("Original size: %d bytes, Compressed size: %d bytes", len(jsonData), len(compressedData))

		req, err := http.NewRequest(http.MethodPost, a.route, bytes.NewBuffer(compressedData))
		if err != nil {
			config.GetLogger().Errorf("Error creating request: %v. Skipping...", err)
			continue
		}

		req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
		req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

		response, err := a.client.Do(req)
		if err != nil {
			config.GetLogger().Errorf("Error sending metric via JSON: %v. Skipping...", err)
			continue
		}

		config.GetLogger().Debugf("JSON Response: %s", response.Status)
		response.Body.Close()
	}
	return nil
}
