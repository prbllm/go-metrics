package agent

import (
	"fmt"
	"math/rand/v2"
	"runtime"

	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type RuntimeMetricsCollector struct {
	logger logger.Logger
}

func NewRuntimeMetricsCollector(logger logger.Logger) *RuntimeMetricsCollector {
	return &RuntimeMetricsCollector{
		logger: logger,
	}
}

func (c *RuntimeMetricsCollector) CollectRuntimeMetrics() []model.Metrics {
	c.logger.Debug("Collecting runtime metrics...")
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pollCount := int64(1)
	randomValue := rand.Float64()
	metrics := []model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Alloc)},
		{ID: "BuckHashSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.BuckHashSys)},
		{ID: "Frees", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Frees)},
		{ID: "GCCPUFraction", MType: model.Gauge, Value: c.ToFloatPointer(memStats.GCCPUFraction)},
		{ID: "GCSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.GCSys)},
		{ID: "HeapAlloc", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapAlloc)},
		{ID: "HeapIdle", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapIdle)},
		{ID: "HeapInuse", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapInuse)},
		{ID: "HeapObjects", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapObjects)},
		{ID: "HeapReleased", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapReleased)},
		{ID: "HeapSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.HeapSys)},
		{ID: "LastGC", MType: model.Gauge, Value: c.ToFloatPointer(memStats.LastGC)},
		{ID: "Lookups", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Lookups)},
		{ID: "MCacheInuse", MType: model.Gauge, Value: c.ToFloatPointer(memStats.MCacheInuse)},
		{ID: "MCacheSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.MCacheSys)},
		{ID: "MSpanInuse", MType: model.Gauge, Value: c.ToFloatPointer(memStats.MSpanInuse)},
		{ID: "MSpanSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.MSpanSys)},
		{ID: "Mallocs", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Mallocs)},
		{ID: "NextGC", MType: model.Gauge, Value: c.ToFloatPointer(memStats.NextGC)},
		{ID: "NumForcedGC", MType: model.Gauge, Value: c.ToFloatPointer(memStats.NumForcedGC)},
		{ID: "NumGC", MType: model.Gauge, Value: c.ToFloatPointer(memStats.NumGC)},
		{ID: "OtherSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.OtherSys)},
		{ID: "PauseTotalNs", MType: model.Gauge, Value: c.ToFloatPointer(memStats.PauseTotalNs)},
		{ID: "StackInuse", MType: model.Gauge, Value: c.ToFloatPointer(memStats.StackInuse)},
		{ID: "StackSys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.StackSys)},
		{ID: "Sys", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Sys)},
		{ID: "TotalAlloc", MType: model.Gauge, Value: c.ToFloatPointer(memStats.TotalAlloc)},
		{ID: "PollCount", MType: model.Counter, Delta: &pollCount},
		{ID: "RandomValue", MType: model.Gauge, Value: &randomValue},
	}

	return metrics
}

func (c *RuntimeMetricsCollector) CollectGopsutilMetrics() ([]model.Metrics, error) {
	c.logger.Debug("Collecting gopsutil metrics...")
	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	cpuStats, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}

	metrics := []model.Metrics{
		{ID: "TotalMemory", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Total)},
		{ID: "FreeMemory", MType: model.Gauge, Value: c.ToFloatPointer(memStats.Free)},
	}

	for i, cpuStat := range cpuStats {
		metrics = append(metrics, model.Metrics{ID: fmt.Sprintf("CPUutilization%d", i), MType: model.Gauge, Value: c.ToFloatPointer(cpuStat)})
	}

	return metrics, nil
}

func (c *RuntimeMetricsCollector) ToFloatPointer(number any) *float64 {
	switch v := number.(type) {
	case float64:
		return &v
	case uint64:
		f := float64(v)
		return &f
	case uint32:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case int32:
		f := float64(v)
		return &f
	default:
		zero := 0.0
		return &zero
	}
}
