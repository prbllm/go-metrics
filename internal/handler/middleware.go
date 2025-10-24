package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prbllm/go-metrics/internal/config"
)

func supportsGzip(acceptEncoding string) bool {
	if acceptEncoding == "" {
		return false
	}

	encodings := strings.Split(acceptEncoding, ",")
	for _, encoding := range encodings {
		encoding = strings.TrimSpace(encoding)
		if encoding == "gzip" {
			return true
		}
	}
	return false
}

func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			config.GetLogger().Infof("HTTP Request: Method=%s, URL=%s, Status=%d, Size=%d bytes, Duration=%v, RemoteAddr=%s",
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

func GzipDecompressMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(config.ContentEncodingHeader) == config.ContentEncodingGzip {
				config.GetLogger().Debug("Decompressing gzip request body")

				gzReader, err := gzip.NewReader(r.Body)
				if err != nil {
					config.GetLogger().Errorf("Failed to create gzip reader: %v", err)
					http.Error(w, "Invalid gzip data", http.StatusBadRequest)
					return
				}
				defer gzReader.Close()

				decompressedBody, err := io.ReadAll(gzReader)
				if err != nil {
					config.GetLogger().Errorf("Failed to decompress gzip data: %v", err)
					http.Error(w, "Invalid gzip data", http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(decompressedBody))
				r.ContentLength = int64(len(decompressedBody))

				r.Header.Del(config.ContentEncodingHeader)

				config.GetLogger().Debugf("Successfully decompressed %d bytes", len(decompressedBody))
			}

			if supportsGzip(r.Header.Get(config.AcceptEncodingHeader)) {
				gzWriter := gzip.NewWriter(w)
				defer gzWriter.Close()

				wrappedWriter := &gzipResponseWriter{
					ResponseWriter: w,
					gzWriter:       gzWriter,
				}

				w.Header().Set(config.ContentEncodingHeader, config.ContentEncodingGzip)
				w.Header().Set(config.VaryHeader, config.AcceptEncodingHeader)

				config.GetLogger().Debug("Compressing response with gzip")

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
