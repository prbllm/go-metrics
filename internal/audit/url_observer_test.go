package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestURLAuditObserver_Process(t *testing.T) {
	t.Run("send single event", func(t *testing.T) {
		var receivedEvent AuditEvent
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, config.ContentTypeJSON, r.Header.Get(config.ContentTypeHeader))

			var event AuditEvent
			err := json.NewDecoder(r.Body).Decode(&event)
			require.NoError(t, err)

			mu.Lock()
			receivedEvent = event
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric_1", "test_metric_2"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.Equal(t, event.MetricsIDs, receivedEvent.MetricsIDs)
		require.Equal(t, event.IPAddress, receivedEvent.IPAddress)
		require.NotZero(t, receivedEvent.Timestamp)
		mu.Unlock()
	})

	t.Run("send multiple events", func(t *testing.T) {
		var receivedEvents []AuditEvent
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event AuditEvent
			err := json.NewDecoder(r.Body).Decode(&event)
			require.NoError(t, err)

			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		events := []AuditEvent{
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric1"}, IPAddress: "192.168.1.1"},
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric2"}, IPAddress: "192.168.1.2"},
			{Timestamp: time.Now().Unix(), MetricsIDs: []string{"metric3"}, IPAddress: "192.168.1.3"},
		}

		for _, event := range events {
			observer.Process(ctx, event)
		}

		time.Sleep(300 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.Len(t, receivedEvents, 3)
		expectedMetrics := make(map[string]string)
		for _, event := range events {
			expectedMetrics[event.MetricsIDs[0]] = event.IPAddress
		}

		for _, received := range receivedEvents {
			require.Len(t, received.MetricsIDs, 1)
			metricID := received.MetricsIDs[0]
			expectedIP, exists := expectedMetrics[metricID]
			require.True(t, exists, "Metric %s not found in expected events", metricID)
			require.Equal(t, expectedIP, received.IPAddress)
		}
		mu.Unlock()
	})

	t.Run("auto-generate timestamp if zero", func(t *testing.T) {
		var receivedEvent AuditEvent
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event AuditEvent
			json.NewDecoder(r.Body).Decode(&event)

			mu.Lock()
			receivedEvent = event
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  0,
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.NotZero(t, receivedEvent.Timestamp)
		require.GreaterOrEqual(t, receivedEvent.Timestamp, time.Now().Unix()-1)
		mu.Unlock()
	})

	t.Run("drop event when observer context cancelled", func(t *testing.T) {
		requestCount := 0
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx, cancel := context.WithCancel(context.Background())
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)

		cancel()
		time.Sleep(50 * time.Millisecond)

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(context.Background(), event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.Equal(t, 0, requestCount)
		mu.Unlock()
	})

	t.Run("drop event when request context cancelled", func(t *testing.T) {
		requestCount := 0
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		reqCtx, cancel := context.WithCancel(context.Background())
		cancel()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(reqCtx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.Equal(t, 0, requestCount)
		mu.Unlock()
	})

	t.Run("handle server error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()
	})

	t.Run("handle bad request status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()
	})
}

func TestURLAuditObserver_Retry(t *testing.T) {
	t.Run("retry on retriable HTTP status 500", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(5 * time.Second)
		observer.Close()

		require.Equal(t, 3, attempts, "Should have retried 2 times (total 3 attempts)")
	})

	t.Run("retry on retriable HTTP status 502", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusBadGateway)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(2 * time.Second)
		observer.Close()

		require.Equal(t, 2, attempts, "Should have retried once (total 2 attempts)")
	})

	t.Run("retry exhausted on retriable status", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(6 * time.Second)
		observer.Close()

		require.GreaterOrEqual(t, attempts, 2, "Should have made at least 2 attempts before context timeout")
	})

	t.Run("no retry on non-retriable status 400", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		require.Equal(t, 1, attempts, "Should not retry on non-retriable status")
	})
}

func TestURLAuditObserver_Close(t *testing.T) {
	t.Run("close gracefully waits for pending events", func(t *testing.T) {
		var receivedEvents []AuditEvent
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event AuditEvent
			json.NewDecoder(r.Body).Decode(&event)

			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)

		for i := 0; i < 5; i++ {
			event := AuditEvent{
				Timestamp:  time.Now().Unix(),
				MetricsIDs: []string{fmt.Sprintf("metric%d", i)},
				IPAddress:  "127.0.0.1",
			}
			observer.Process(ctx, event)
		}

		time.Sleep(300 * time.Millisecond)
		observer.Close()

		mu.Lock()
		require.Len(t, receivedEvents, 5, "Expected 5 events, got %d", len(receivedEvents))
		mu.Unlock()
	})

	t.Run("close is idempotent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)

		observer.Close()
		observer.Close()
		observer.Close()
	})
}

func TestURLAuditObserver_NetworkError(t *testing.T) {
	t.Run("handle connection error gracefully", func(t *testing.T) {
		invalidURL := "http://localhost:99999/audit"

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, invalidURL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(300 * time.Millisecond)
		observer.Close()
	})
}

func TestURLAuditObserver_ContentType(t *testing.T) {
	t.Run("send correct content type header", func(t *testing.T) {
		var contentType string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get(config.ContentTypeHeader)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		require.Equal(t, config.ContentTypeJSON, contentType)
	})
}

func TestURLAuditObserver_RequestBody(t *testing.T) {
	t.Run("send valid JSON in request body", func(t *testing.T) {
		var requestBody string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			requestBody = string(body)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, server.URL, logger)
		require.NoError(t, err)
		defer observer.Close()

		event := AuditEvent{
			Timestamp:  time.Now().Unix(),
			MetricsIDs: []string{"test_metric"},
			IPAddress:  "127.0.0.1",
		}

		observer.Process(ctx, event)

		time.Sleep(200 * time.Millisecond)
		observer.Close()

		require.NotEmpty(t, requestBody)

		var parsedEvent AuditEvent
		err = json.Unmarshal([]byte(requestBody), &parsedEvent)
		require.NoError(t, err)
		require.Equal(t, event.MetricsIDs, parsedEvent.MetricsIDs)
		require.Equal(t, event.IPAddress, parsedEvent.IPAddress)
		require.True(t, strings.Contains(requestBody, "ts"))
		require.True(t, strings.Contains(requestBody, "metrics"))
		require.True(t, strings.Contains(requestBody, "ip_address"))
	})
}

func TestURLAuditObserver_Validation(t *testing.T) {
	t.Run("reject empty URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, "", logger)
		require.Error(t, err)
		require.Nil(t, observer)
		require.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("reject invalid URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, "not-a-valid-url", logger)
		require.Error(t, err)
		require.Nil(t, observer)
		require.True(t, strings.Contains(err.Error(), "invalid audit URL") || strings.Contains(err.Error(), "must use http or https"), "Error message: %s", err.Error())
	})

	t.Run("reject non-http URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, "ftp://example.com/audit", logger)
		require.Error(t, err)
		require.Nil(t, observer)
		require.Contains(t, err.Error(), "must use http or https")
	})

	t.Run("accept valid http URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, "http://localhost:8080/audit", logger)
		require.NoError(t, err)
		require.NotNil(t, observer)
		observer.Close()
	})

	t.Run("accept valid https URL", func(t *testing.T) {
		logger := zaptest.NewLogger(t).Sugar()
		ctx := context.Background()
		observer, err := NewURLAuditObserver(ctx, "https://example.com/audit", logger)
		require.NoError(t, err)
		require.NotNil(t, observer)
		observer.Close()
	})
}
