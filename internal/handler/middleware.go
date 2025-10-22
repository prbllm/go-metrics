package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prbllm/go-metrics/internal/config"
)

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
