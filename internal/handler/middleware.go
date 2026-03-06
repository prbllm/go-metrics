package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/encryption"
	"github.com/prbllm/go-metrics/internal/hash"
	"github.com/prbllm/go-metrics/internal/logger"
)

// LoggingMiddleware создает middleware для логирования HTTP запросов.
func LoggingMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			logger.Infof("HTTP Request: Method=%s, URL=%s, Status=%d, Size=%d bytes, Duration=%v, RemoteAddr=%s",
				r.Method,
				r.URL.String(),
				ww.Status(),
				ww.BytesWritten(),
				duration,
				r.RemoteAddr,
			)
		})
	}
}

// DecryptCryptoMiddleware создает middleware для расшифровки зашифрованного тела запроса.
func DecryptCryptoMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	type encryptedPayload struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.GetConfig()
			if cfg.CryptoKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}
			if len(bodyBytes) == 0 {
				http.Error(w, "Empty encrypted body", http.StatusBadRequest)
				return
			}

			var payload encryptedPayload
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				http.Error(w, "Invalid encrypted payload", http.StatusBadRequest)
				return
			}

			encKey, err := base64.StdEncoding.DecodeString(payload.Key)
			if err != nil {
				http.Error(w, "Invalid encrypted key", http.StatusBadRequest)
				return
			}

			cipherData, err := base64.StdEncoding.DecodeString(payload.Data)
			if err != nil {
				http.Error(w, "Invalid encrypted data", http.StatusBadRequest)
				return
			}

			plaintext, err := encryption.DecryptHybrid(cfg.CryptoKey, encKey, cipherData)
			if err != nil {
				http.Error(w, "Invalid crypto", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))
			r.Header.Del(config.ContentEncodingHeader)

			next.ServeHTTP(w, r)
		})
	}
}

// GzipDecompressMiddleware создает middleware для распаковки gzip запросов и сжатия ответов.
func GzipDecompressMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debugf("GzipDecompressMiddleware: Method=%s, URL=%s, Path=%s", r.Method, r.URL.String(), r.URL.Path)

			contentEncoding := r.Header.Get(config.ContentEncodingHeader)
			if contentEncoding == config.ContentEncodingGzip {
				logger.Debug("Decompressing gzip request body")

				decompressedBody, err := compression.DecompressReader(r.Body)
				if err != nil {
					http.Error(w, "Invalid gzip data", http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(decompressedBody))
				r.ContentLength = int64(len(decompressedBody))

				r.Header.Del(config.ContentEncodingHeader)

				logger.Debugf("Successfully decompressed %d bytes", len(decompressedBody))
				logger.Debugf("Decompressed request body: %s", string(decompressedBody))
			} else {
				logger.Debugf("Request does not contain gzip encoding (Content-Encoding: %s), skipping decompression", contentEncoding)
			}

			if compression.SupportsGzip(r.Header.Get(config.AcceptEncodingHeader)) {
				gzWriter := gzip.NewWriter(w)
				defer gzWriter.Close()

				wrappedWriter := &gzipResponseWriter{
					ResponseWriter: w,
					gzWriter:       gzWriter,
				}

				w.Header().Set(config.ContentEncodingHeader, config.ContentEncodingGzip)
				w.Header().Set(config.VaryHeader, config.AcceptEncodingHeader)

				logger.Debug("Compressing response with gzip")

				next.ServeHTTP(wrappedWriter, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzWriter *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gzWriter.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// HashValidationMiddleware создает middleware для проверки и добавления хеша SHA256 в запросы и ответы.
func HashValidationMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hashHeader := r.Header.Get(config.HashSHA256Header)
			if hashHeader != "" {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read request body", http.StatusInternalServerError)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				hashActual := hash.ComputeHash(config.GetConfig().Key, bodyBytes)
				if hashActual != hashHeader {
					http.Error(w, "Invalid hash", http.StatusBadRequest)
					return
				}
			}

			bufferingRecorder := newBufferingRecorder()
			next.ServeHTTP(bufferingRecorder, r)

			respBytes := bufferingRecorder.body.Bytes()
			respHash := hash.ComputeHash(config.GetConfig().Key, respBytes)
			bufferingRecorder.Header().Set(config.HashSHA256Header, respHash)
			bufferingRecorder.FlushTo(w)
		})
	}
}

// TrustedSubnetMiddleware создает middleware для проверки доверенной подсети клиента по заголовку X-Real-IP.
// Если в конфигурации не задана trusted_subnet, запросы проходят без ограничений.
func TrustedSubnetMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.GetConfig()

			if cfg.TrustedSubnet == "" {
				next.ServeHTTP(w, r)
				return
			}

			ipStr := r.Header.Get(config.RealIPHeader)
			if ipStr == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ip := net.ParseIP(ipStr)
			if ip == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			_, ipNet, err := net.ParseCIDR(cfg.TrustedSubnet)
			if err != nil {
				logger.Errorf("invalid trusted subnet configuration %q: %v", cfg.TrustedSubnet, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if !ipNet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type bufferingRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferingRecorder() *bufferingRecorder {
	return &bufferingRecorder{
		header: make(http.Header),
	}
}

func (br *bufferingRecorder) Header() http.Header {
	return br.header
}

func (br *bufferingRecorder) WriteHeader(code int) {
	br.status = code
}

func (br *bufferingRecorder) Write(p []byte) (int, error) {
	return br.body.Write(p)
}

func (br *bufferingRecorder) FlushTo(w http.ResponseWriter) error {
	for k, vv := range br.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(br.status)
	_, err := w.Write(br.body.Bytes())
	return err
}
