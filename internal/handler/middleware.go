package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prbllm/go-metrics/internal/compression"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
)

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

func GzipDecompressMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(config.ContentEncodingHeader) == config.ContentEncodingGzip {
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
