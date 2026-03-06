package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/encryption"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestLoggingMiddleware(t *testing.T) {
	router := chi.NewRouter()
	router.Use(LoggingMiddleware(zaptest.NewLogger(t).Sugar()))

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
	logger := zaptest.NewLogger(t).Sugar()
	storage := repository.NewMemStorage(logger)
	metricsService := service.NewMetricsService(storage)
	handlers := NewHandlers(metricsService, logger)

	router := chi.NewRouter()
	router.Use(LoggingMiddleware(logger))
	router.Get("/", handlers.GetAllMetricsHandlerByURL)

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

	compressedData, err := compression.CompressData(jsonData)
	require.NoError(t, err, "Failed to compress data")

	router := chi.NewRouter()
	router.Use(GzipDecompressMiddleware(zaptest.NewLogger(t).Sugar()))

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
			result := compression.SupportsGzip(tc.acceptEncoding)
			require.Equal(t, tc.expected, result,
				"supportsGzip(%q) should return %v", tc.acceptEncoding, tc.expected)
		})
	}
}

func TestGzipDecompressMiddlewareWithAcceptEncoding(t *testing.T) {
	responseData := []byte(`{"message":"Hello, World!","data":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95,96,97,98,99,100],"description":"This is a large JSON response that should be compressed effectively by gzip compression algorithm"}`)

	router := chi.NewRouter()
	router.Use(GzipDecompressMiddleware(zaptest.NewLogger(t).Sugar()))

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

func TestHashValidationMiddleware(t *testing.T) {
	router := chi.NewRouter()
	router.Use(HashValidationMiddleware(zaptest.NewLogger(t).Sugar()))

	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	testCases := []struct {
		name           string
		key            string
		emptyHeader    bool
		expectedStatus int
	}{
		{
			name:           "Valid",
			key:            "test-key",
			emptyHeader:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid header",
			emptyHeader:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Another key",
			key:            "another-key",
			emptyHeader:    false,
			expectedStatus: http.StatusBadRequest,
		},
	}

	config.SetConfig(&config.Config{
		Key: "test-key",
	}, zaptest.NewLogger(t).Sugar())

	jsonData := []byte(`{"id": "test", "type": "gauge", "value": 1.0}`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(jsonData))
			if !tc.emptyHeader {
				req.Header.Set(config.HashSHA256Header, hash.ComputeHash(tc.key, jsonData))
			}
			req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			require.Equal(t, tc.expectedStatus, rr.Code, "Expected status %d", tc.expectedStatus)
		})
	}
}

func TestTrustedSubnetMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	t.Run("no trusted subnet configured - passes without header", func(t *testing.T) {
		config.SetConfig(config.GetConfig(), logger)

		router := chi.NewRouter()
		router.Use(TrustedSubnetMiddleware(logger))
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("trusted subnet configured - header in subnet", func(t *testing.T) {
		cfg := *config.GetConfig()
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		router := chi.NewRouter()
		router.Use(TrustedSubnetMiddleware(logger))
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(config.RealIPHeader, "127.0.0.1")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("trusted subnet configured - header outside subnet", func(t *testing.T) {
		cfg := *config.GetConfig()
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		router := chi.NewRouter()
		router.Use(TrustedSubnetMiddleware(logger))
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(config.RealIPHeader, "10.0.0.1")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("trusted subnet configured - no header", func(t *testing.T) {
		cfg := *config.GetConfig()
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		router := chi.NewRouter()
		router.Use(TrustedSubnetMiddleware(logger))
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("trusted subnet configured - invalid IP header", func(t *testing.T) {
		cfg := *config.GetConfig()
		cfg.TrustedSubnet = "127.0.0.0/8"
		config.SetConfig(&cfg, logger)

		router := chi.NewRouter()
		router.Use(TrustedSubnetMiddleware(logger))
		router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(config.RealIPHeader, "not-an-ip")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusForbidden, rr.Code)
	})
}

func TestDecryptCryptoMiddleware_NoCryptoKeyConfigured(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	config.SetConfig(&config.Config{
		CryptoKey: "",
	}, logger)

	router := chi.NewRouter()
	router.Use(DecryptCryptoMiddleware(logger))
	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("plain-body"), body)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("plain-body")))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestDecryptCryptoMiddleware_ValidEncryptedPayload(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	pubPath, privPath := func() (string, string) {
		t.Helper()
		return encryptionTestKeyPairFiles(t)
	}()

	config.SetConfig(&config.Config{
		CryptoKey: privPath,
	}, logger)

	type encryptedPayload struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	originalBody := []byte(`{"foo":"bar"}`)

	encKey, ciphertext, err := encryption.EncryptHybrid(pubPath, originalBody)
	require.NoError(t, err)

	payload := encryptedPayload{
		Key:  base64.StdEncoding.EncodeToString(encKey),
		Data: base64.StdEncoding.EncodeToString(ciphertext),
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	router := chi.NewRouter()
	router.Use(DecryptCryptoMiddleware(logger))
	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, originalBody, body)
		require.Empty(t, r.Header.Get(config.ContentEncodingHeader))
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(payloadBytes))
	req.Header.Set(config.ContentTypeHeader, config.ContentTypeJSON)
	req.Header.Set(config.ContentEncodingHeader, config.ContentEncodingGzip)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestDecryptCryptoMiddleware_InvalidEncryptedPayload(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	_, privPath := encryptionTestKeyPairFiles(t)
	config.SetConfig(&config.Config{
		CryptoKey: privPath,
	}, logger)

	router := chi.NewRouter()
	router.Use(DecryptCryptoMiddleware(logger))
	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("not-json")))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	badPayload := map[string]string{
		"key":  "!!!not-base64!!!",
		"data": "!!!not-base64!!!",
	}
	badBytes, err := json.Marshal(badPayload)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(badBytes))
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusBadRequest, rr2.Code)
}

func encryptionTestKeyPairFiles(t *testing.T) (pubPath, privPath string) {
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

	tempDir := t.TempDir()
	privPath = tempDir + "/private.pem"
	pubPath = tempDir + "/public.pem"

	require.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(privBlock), 0600))
	require.NoError(t, os.WriteFile(pubPath, pem.EncodeToMemory(pubBlock), 0644))

	return pubPath, privPath
}
