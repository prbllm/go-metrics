package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(handlers *Handlers) *chi.Mux {
	router := chi.NewRouter()
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandler)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandler)
		})
	})
	return router
}

func TestUpdateHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		expectedStatusCode int
	}{
		{
			name:               "valid counter request",
			method:             http.MethodPost,
			path:               "/update/counter/test_counter/42",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "valid gauge request",
			method:             http.MethodPost,
			path:               "/update/gauge/test_gauge/3.14",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "invalid method request",
			method:             http.MethodGet,
			path:               "/update/counter/test_counter/42",
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "invalid path - missing parts",
			method:             http.MethodPost,
			path:               "/update/counter/test",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid metric type",
			method:             http.MethodPost,
			path:               "/update/invalid/test/42",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid counter value",
			method:             http.MethodPost,
			path:               "/update/counter/test/abc",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid gauge value",
			method:             http.MethodPost,
			path:               "/update/gauge/test/invalid",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlers := NewHandlers(&service.MockMetricsService{})
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, nil)
			req.Header.Set("Content-Type", "text/plain")

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	handlers := NewHandlers(&service.MockMetricsService{})
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()

	handlers.NotFoundHandler(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code, "Expected status code %d, got %d", http.StatusNotFound, rr.Code)
}

func TestGetAllMetricsHandler(t *testing.T) {
	tests := []struct {
		name                string
		method              string
		path                string
		expectedStatusCode  int
		expectedContentType string
	}{
		{
			name:                "valid GET request",
			method:              http.MethodGet,
			path:                "/",
			expectedStatusCode:  http.StatusOK,
			expectedContentType: "text/html; charset=utf-8",
		},
		{
			name:                "invalid method - POST",
			method:              http.MethodPost,
			path:                "/",
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedContentType: "",
		},
		{
			name:                "invalid method - PUT",
			method:              http.MethodPut,
			path:                "/",
			expectedStatusCode:  http.StatusMethodNotAllowed,
			expectedContentType: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlers := NewHandlers(&service.MockMetricsService{})
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)

			if test.expectedContentType != "" {
				require.Equal(t, test.expectedContentType, rr.Header().Get("Content-Type"), "Expected content type %s, got %s", test.expectedContentType, rr.Header().Get("Content-Type"))
			}
		})
	}
}
