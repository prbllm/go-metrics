package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func parseJSONLines(content string) []string {
	contentStr := strings.TrimSpace(content)
	if contentStr == "" {
		return nil
	}

	lines := strings.Split(contentStr, "\n")
	var jsonObjects []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var testEvent AuditEvent
		if err := json.Unmarshal([]byte(line), &testEvent); err == nil {
			jsonObjects = append(jsonObjects, line)
		}
	}

	return jsonObjects
}

func TestFileAuditObserver_Process(t *testing.T) {
	t.Run("write single event", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric_1", "test_metric_2"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(100 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)
		require.NotEmpty(t, content)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 1)

		var savedEvent AuditEvent
		err = json.Unmarshal([]byte(lines[0]), &savedEvent)
		require.NoError(t, err)
		require.Equal(t, event.MetricsIDs, savedEvent.MetricsIDs)
		require.Equal(t, event.IPAddress, savedEvent.IPAddress)
		require.NotZero(t, savedEvent.Timestamp)
	})

	t.Run("write multiple events", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)
		defer observer.Close()

		events := []AuditEvent{
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric1"}, IPAddress: "192.168.1.1"},
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric2"}, IPAddress: "192.168.1.2"},
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric3"}, IPAddress: "192.168.1.3"},
		}

		for _, event := range events {
			observer.Process(ctx, event)
		}

		time.Sleep(500 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)

		jsonObjects := parseJSONLines(string(content))
		require.Len(t, jsonObjects, 3, "Expected 3 events, got %d. Content: %s", len(jsonObjects), string(content))

		savedEvents := make([]AuditEvent, 0, 3)
		for _, obj := range jsonObjects {
			var savedEvent AuditEvent
			err = json.Unmarshal([]byte(obj), &savedEvent)
			require.NoError(t, err, "Failed to unmarshal JSON: %s", obj)
			savedEvents = append(savedEvents, savedEvent)
		}
		require.Len(t, savedEvents, 3)

		expectedMetrics := make(map[string]string)
		for _, event := range events {
			expectedMetrics[event.MetricsIDs[0]] = event.IPAddress
		}

		for _, savedEvent := range savedEvents {
			require.Len(t, savedEvent.MetricsIDs, 1)
			metricID := savedEvent.MetricsIDs[0]
			expectedIP, exists := expectedMetrics[metricID]
			require.True(t, exists, "Metric %s not found in expected events", metricID)
			require.Equal(t, expectedIP, savedEvent.IPAddress)
		}
	})

	t.Run("auto-generate timestamp if zero", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  0,
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(100 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)

		var savedEvent AuditEvent
		err = json.Unmarshal([]byte(strings.TrimSpace(string(content))), &savedEvent)
		require.NoError(t, err)
		require.NotZero(t, savedEvent.Timestamp)
		require.GreaterOrEqual(t, savedEvent.Timestamp, time.Now().Unix()-1)
	})

	t.Run("drop event when observer context cancelled", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx, cancel := context.WithCancel(context.Background())
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)

		cancel()
		time.Sleep(50 * time.Millisecond)

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(context.Background(), event)

		time.Sleep(100 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(string(content)))
	})

	t.Run("drop event when request context cancelled", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)
		defer observer.Close()

		reqCtx, cancel := context.WithCancel(context.Background())
		cancel()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(reqCtx, event)

		time.Sleep(100 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(string(content)))
	})
}

func TestFileAuditObserver_Close(t *testing.T) {
	t.Run("close gracefully waits for pending events", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)

		for i := 0; i < 5; i++ {
			event := AuditEvent{
				Timestamp:  time.Now().Unix(),
				MetricsIDs: []string{fmt.Sprintf("metric%d", i)},
				IPAddress:  "127.0.0.1",
			}
			observer.Process(ctx, event)
		}

		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)

		jsonObjects := parseJSONLines(string(content))
		require.Len(t, jsonObjects, 5, "Expected 5 events, got %d. Content: %s", len(jsonObjects), string(content))
	})

	t.Run("close is idempotent", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)

		observer.Close()
		observer.Close()
		observer.Close()
	})
}

func TestFileAuditObserver_AppendMode(t *testing.T) {
	t.Run("append to existing file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "audit_test_*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		initialEvent := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"initial_metric"},
			IPAddress:  "127.0.0.1",
		}
		initialData, _ := json.Marshal(initialEvent)
		os.WriteFile(tempFile.Name(), append(initialData, '\n'), 0644)
		tempFile.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer := NewFileAuditObserver(ctx, tempFile.Name(), logger)
		defer observer.Close()

		newEvent := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"new_metric"},
			IPAddress:  "192.168.1.1",
		}
		observer.Process(ctx, newEvent)

		time.Sleep(100 * time.Millisecond)
		observer.Close()

		content, err := os.ReadFile(tempFile.Name())
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 2)

		var firstEvent, secondEvent AuditEvent
		json.Unmarshal([]byte(lines[0]), &firstEvent)
		json.Unmarshal([]byte(lines[1]), &secondEvent)

		require.Equal(t, "initial_metric", firstEvent.MetricsIDs[0])
		require.Equal(t, "new_metric", secondEvent.MetricsIDs[0])
	})
}
