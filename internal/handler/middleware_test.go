package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware(t *testing.T) {
	router := chi.NewRouter()
	router.Use(LoggingMiddleware())

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMiddlewareWithHandlers(t *testing.T) {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := NewHandlers(metricsService)

	router := chi.NewRouter()
	router.Use(LoggingMiddleware())
	router.Get("/", handlers.GetAllMetricsHandlerByUrl)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestGzipDecompressMiddleware(t *testing.T) {
	metric := model.Metrics{
		ID:    "TestMetric",
		MType: model.Gauge,
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	jsonData, err := json.Marshal(metric)
	require.NoError(t, err, "Failed to marshal metric to JSON")

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err = gzWriter.Write(jsonData)
	require.NoError(t, err, "Failed to write to gzip writer")
	err = gzWriter.Close()
	require.NoError(t, err, "Failed to close gzip writer")
	compressedData := buf.Bytes()

	router := chi.NewRouter()
	router.Use(GzipDecompressMiddleware())

	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get(config.ContentEncodingHeader), "Content-Encoding header should be removed")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "Failed to read request body")

		require.Equal(t, jsonData, body, "Decompressed data should match original JSON")

		require.Equal(t, int64(len(jsonData)), r.ContentLength, "Content-Length should match decompressed size")

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(compressedData))
	req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
	req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)
	req.ContentLength = int64(len(compressedData))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "Expected status 200")
}

func TestSupportsGzip(t *testing.T) {
	testCases := []struct {
		name           string
		acceptEncoding string
		expected       bool
	}{
		{"Empty header", "", false},
		{"Only gzip", "gzip", true},
		{"Gzip with deflate", "gzip, deflate", true},
		{"Deflate with gzip", "deflate, gzip", true},
		{"Only deflate", "deflate", false},
		{"Identity", "identity", false},
		{"Gzip with spaces", " gzip , deflate ", true},
		{"Multiple gzip", "gzip, gzip, deflate", true},
		{"Case sensitive", "GZIP", false}, // gzip должен быть в нижнем регистре
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := supportsGzip(tc.acceptEncoding)
			require.Equal(t, tc.expected, result,
				"supportsGzip(%q) should return %v", tc.acceptEncoding, tc.expected)
		})
	}
}

func TestGzipDecompressMiddlewareWithAcceptEncoding(t *testing.T) {
	responseData := []byte(`{"message":"Hello, World!","data":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95,96,97,98,99,100],"description":"This is a large JSON response that should be compressed effectively by gzip compression algorithm"}`)

	router := chi.NewRouter()
	router.Use(GzipDecompressMiddleware())

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseData)
	})

	testCases := []struct {
		name           string
		acceptEncoding string
		shouldCompress bool
	}{
		{"Gzip only", "gzip", true},
		{"Gzip with deflate", "gzip, deflate", true},
		{"Deflate with gzip", "deflate, gzip", true},
		{"Only deflate", "deflate", false},
		{"Identity", "identity", false},
		{"Empty", "", false},
		{"Gzip with spaces", " gzip , deflate ", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tc.acceptEncoding)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "Expected status 200")

			if tc.shouldCompress {
				require.Equal(t, config.ContentEncodingGzip, rr.Header().Get(config.ContentEncodingHeader),
					"Response should have gzip encoding header")
				require.Equal(t, config.AcceptEncodingHeader, rr.Header().Get(config.VaryHeader),
					"Response should have Vary header")
				require.Less(t, len(rr.Body.Bytes()), len(responseData),
					"Compressed response should be smaller than original")
			} else {
				require.Empty(t, rr.Header().Get(config.ContentEncodingHeader),
					"Response should not have gzip encoding header")
				require.Equal(t, responseData, rr.Body.Bytes(),
					"Response should not be compressed")
			}
		})
	}
}
