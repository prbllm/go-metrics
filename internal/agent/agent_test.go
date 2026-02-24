package agent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/encryption"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestAgentGenerateUrl(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	testData := []struct {
		metric      model.Metrics
		expectedURL string
		expectError bool
	}{
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
			expectedURL: "http://localhost:8080/update/gauge/test_metric/1.000000",
			expectError: false,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
			expectedURL: "http://localhost:8080/update/counter/test_metric/1",
			expectError: false,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Gauge},
			expectedURL: "",
			expectError: true,
		},
		{
			metric:      model.Metrics{ID: "test_metric", MType: model.Counter},
			expectedURL: "",
			expectError: true,
		},
	}

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)
	for _, test := range testData {
		url, err := agent.generateURL(test.metric)
		if test.expectError {
			require.Error(t, err, "Expected error")
		} else {
			require.NoError(t, err, "Failed to generate URL")
		}
		require.Equal(t, test.expectedURL, url, "URL is not equal to expected")
	}
}

func TestAgentSendMetrics(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.sendMetrics(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics")
}

func TestAgentSendMetricsJSON(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Failed to send metrics via JSON")
}

func TestAgentSendMetricsJSONWithNilClient(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)
	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.Error(t, err, "Expected error for nil client")
	require.Contains(t, err.Error(), "client is nil")
}

func TestAgentSendMetricsJSONSerialization(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	metrics := []model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &commonValue},
		{ID: "test_counter", MType: model.Counter, Delta: &commonDelta},
	}

	for _, metric := range metrics {
		jsonData, err := json.Marshal(metric)
		require.NoError(t, err, "Failed to marshal metric to JSON")
		require.NotEmpty(t, jsonData, "JSON data should not be empty")

		jsonStr := string(jsonData)
		require.Contains(t, jsonStr, metric.ID, "JSON should contain metric ID")
		require.Contains(t, jsonStr, metric.MType, "JSON should contain metric type")

		if metric.MType == model.Gauge && metric.Value != nil {
			require.Contains(t, jsonStr, "value", "JSON should contain value field for gauge")
		}
		if metric.MType == model.Counter && metric.Delta != nil {
			require.Contains(t, jsonStr, "delta", "JSON should contain delta field for counter")
		}
	}
}

func TestAgentSendMetricsBatchJSON(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}
	_ = agent.SendMetricsBatchJSON(context.Background(), metrics)
}

func TestAgentSendMetricsBatchJSONWithNilClient(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(nil, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
		{ID: "test_metric", MType: model.Counter, Delta: &commonDelta},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Expected error for nil client")
	require.Contains(t, err.Error(), "client is nil")
}

func TestAgentSendMetricsBatchJSONEmptyBatch(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	cfg := &config.Config{
		ServerHost: "localhost:8080",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Empty batch should not return error")
}

func TestAgentSendMetricsBatchJSONSerialization(t *testing.T) {
	commonValue := float64(1.0)
	commonDelta := int64(1)

	metrics := []model.Metrics{
		{ID: "test_gauge", MType: model.Gauge, Value: &commonValue},
		{ID: "test_counter", MType: model.Counter, Delta: &commonDelta},
	}

	jsonData, err := json.Marshal(metrics)
	require.NoError(t, err, "Failed to marshal metrics batch to JSON")
	require.NotEmpty(t, jsonData, "JSON data should not be empty")

	jsonStr := string(jsonData)
	require.Contains(t, jsonStr, "test_gauge", "JSON should contain first metric ID")
	require.Contains(t, jsonStr, "test_counter", "JSON should contain second metric ID")
	require.Contains(t, jsonStr, "gauge", "JSON should contain gauge type")
	require.Contains(t, jsonStr, "counter", "JSON should contain counter type")
}

func TestAgentSendMetricsJSON_RetryOnRetriableHTTPStatus(t *testing.T) {
	commonValue := float64(1.0)
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
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should succeed after retry")
	require.Equal(t, 3, attempts, "Should have retried 2 times (total 3 attempts)")
}

func TestAgentSendMetricsJSON_RetryExhausted(t *testing.T) {
	commonValue := float64(1.0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should not return error (skips metric after retry exhaustion)")
	require.Equal(t, 4, attempts, "Should have exhausted all retries (1 initial + 3 retries)")
}

func TestAgentSendMetricsJSON_RetryOnDifferentRetriableStatuses(t *testing.T) {
	commonValue := float64(1.0)
	testCases := []struct {
		name        string
		statusCode  int
		shouldRetry bool
	}{
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"404 Not Found", http.StatusNotFound, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts == 1 {
					w.WriteHeader(tc.statusCode)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()

			logger := zaptest.NewLogger(t).Sugar()
			serverURL, _ := url.Parse(server.URL)
			cfg := &config.Config{
				ServerHost: serverURL.Host,
			}
			config.SetConfig(cfg, logger)
			agent := NewAgent(http.DefaultClient, nil, logger)

			metrics := []model.Metrics{
				{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
			}

			err := agent.SendMetricsJSON(context.Background(), metrics)
			if tc.shouldRetry {
				require.NoError(t, err, "Should succeed after retry for retriable status")
				require.Equal(t, 2, attempts, "Should have retried once")
			} else {
				require.NoError(t, err, "Should not return error (skips non-retriable error)")
				require.Equal(t, 1, attempts, "Should not retry for non-retriable status")
			}
		})
	}
}

func TestAgentSendMetricsBatchJSON_RetryOnRetriableHTTPStatus(t *testing.T) {
	commonValue := float64(1.0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should succeed after retry")
	require.Equal(t, 2, attempts, "Should have retried once (total 2 attempts)")
}

func TestAgentSendMetricsBatchJSON_RetryExhausted(t *testing.T) {
	commonValue := float64(1.0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Should return error after retry exhaustion")
	require.Contains(t, err.Error(), "error sending metrics batch")
	require.Equal(t, 4, attempts, "Should have exhausted all retries (1 initial + 3 retries)")
}

func TestAgentSendMetricsBatchJSON_NoRetryOnNonRetriableStatus(t *testing.T) {
	commonValue := float64(1.0)
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.Error(t, err, "Should return error for non-retriable status")
	require.Contains(t, err.Error(), "unexpected status code")
	require.Equal(t, 1, attempts, "Should not retry for non-retriable status")
}

func TestAgentSendMetricsJSON_ContextTimeout(t *testing.T) {
	commonValue := float64(1.0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(ctx, metrics)
	require.NoError(t, err, "Should not return error (skips metric on timeout)")
}

func TestAgentSendMetricsJSON_WithHashHeader(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-key-123"
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics successfully")
	require.NotEmpty(t, receivedHash, "HashSHA256 header should be present when key is set")
}

func TestAgentSendMetricsJSON_WithoutHashHeader(t *testing.T) {
	commonValue := float64(1.0)
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        "",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics successfully")
	require.Empty(t, receivedHash, "HashSHA256 header should not be present when key is empty")
}

func TestAgentSendMetricsJSON_HashComputation(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-key-456"
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics successfully")
	require.NotEmpty(t, receivedHash, "HashSHA256 header should be present")

	jsonData, err := json.Marshal(metrics[0])
	require.NoError(t, err, "Should marshal metric to JSON")
	expectedHash := hash.ComputeHash(testKey, jsonData)
	require.Equal(t, expectedHash, receivedHash, "Hash should be computed correctly based on key and original JSON body")
}

func TestAgentSendMetricsBatchJSON_WithHashHeader(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-key-789"
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics batch successfully")
	require.NotEmpty(t, receivedHash, "HashSHA256 header should be present when key is set")
}

func TestAgentSendMetricsBatchJSON_WithoutHashHeader(t *testing.T) {
	commonValue := float64(1.0)
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        "",
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics batch successfully")
	require.Empty(t, receivedHash, "HashSHA256 header should not be present when key is empty")
}

func TestAgentSendMetricsBatchJSON_HashComputation(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-key-batch-123"
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send metrics batch successfully")
	require.NotEmpty(t, receivedHash, "HashSHA256 header should be present")

	jsonData, err := json.Marshal(metrics)
	require.NoError(t, err, "Should marshal metrics batch to JSON")
	expectedHash := hash.ComputeHash(testKey, jsonData)
	require.Equal(t, expectedHash, receivedHash, "Hash should be computed correctly based on key and original JSON body")
}

func TestAgentSendMetricsJSON_WithCryptoKey_EncryptedPayloadAndHash(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-crypto-key"

	pubPath, privPath := generateRSAKeyPairFilesForAgent(t)

	var receivedBody []byte
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)

		var payload struct {
			Key  string `json:"key"`
			Data string `json:"data"`
		}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		encKey, err := base64.StdEncoding.DecodeString(payload.Key)
		require.NoError(t, err)
		cipherData, err := base64.StdEncoding.DecodeString(payload.Data)
		require.NoError(t, err)

		plaintext, err := encryption.DecryptHybrid(privPath, encKey, cipherData)
		require.NoError(t, err)
		receivedBody = plaintext

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
		CryptoKey:  pubPath,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send encrypted metrics successfully")
	require.NotEmpty(t, receivedBody, "Decrypted body should be captured")

	expectedHash := hash.ComputeHash(testKey, receivedBody)
	require.Equal(t, expectedHash, receivedHash, "Hash should be computed from original JSON body")
}

func TestAgentSendMetricsBatchJSON_WithCryptoKey_EncryptedPayloadAndHash(t *testing.T) {
	commonValue := float64(1.0)
	testKey := "test-crypto-key-batch"

	pubPath, privPath := generateRSAKeyPairFilesForAgent(t)

	var receivedBody []byte
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash = r.Header.Get(config.HashSHA256Header)

		var payload struct {
			Key  string `json:"key"`
			Data string `json:"data"`
		}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		encKey, err := base64.StdEncoding.DecodeString(payload.Key)
		require.NoError(t, err)
		cipherData, err := base64.StdEncoding.DecodeString(payload.Data)
		require.NoError(t, err)

		plaintext, err := encryption.DecryptHybrid(privPath, encKey, cipherData)
		require.NoError(t, err)
		receivedBody = plaintext

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t).Sugar()
	serverURL, _ := url.Parse(server.URL)
	cfg := &config.Config{
		ServerHost: serverURL.Host,
		Key:        testKey,
		CryptoKey:  pubPath,
	}
	config.SetConfig(cfg, logger)
	agent := NewAgent(http.DefaultClient, nil, logger)

	metrics := []model.Metrics{
		{ID: "test_metric", MType: model.Gauge, Value: &commonValue},
	}

	err := agent.SendMetricsBatchJSON(context.Background(), metrics)
	require.NoError(t, err, "Should send encrypted batch successfully")
	require.NotEmpty(t, receivedBody, "Decrypted batch body should be captured")

	expectedHash := hash.ComputeHash(testKey, receivedBody)
	require.Equal(t, expectedHash, receivedHash, "Hash should be computed from original JSON batch body")
}

func generateRSAKeyPairFilesForAgent(t *testing.T) (pubPath, privPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	tempDir, err := os.MkdirTemp("", "agent-crypto-*")
	require.NoError(t, err)

	privPath = tempDir + "/private.pem"
	pubPath = tempDir + "/public.pem"

	require.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(privBlock), 0600))
	require.NoError(t, os.WriteFile(pubPath, pem.EncodeToMemory(pubBlock), 0644))

	return pubPath, privPath
}
