package agent

import (
	"context"
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
	config.GetLogger().Info("Starting agent")
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
		err := a.sendMetrics(metrics)
		if err != nil {
			config.GetLogger().Errorf("Error sending metrics: ", err)
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
			config.GetLogger().Warnf("Error generating url: ", err, ". Skipping...")
			continue
		}
		config.GetLogger().Debugf("Sending metric: ", metric.String(), "to url: ", url)
		response, err := a.client.Post(url, "text/plain", strings.NewReader(""))
		if err != nil {
			config.GetLogger().Errorf("Error sending metric: ", err, ". Skipping...")
			continue
		}
		config.GetLogger().Debugf("Response: ", response.Status)
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
