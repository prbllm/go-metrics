package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
)

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

			req := httptest.NewRequest(test.method, test.path, nil)
			req.Header.Set("Content-Type", "text/plain")

			rr := httptest.NewRecorder()

			handlers.UpdateHandler(rr, req)
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
