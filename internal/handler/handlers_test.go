package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/mocks"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func setupTestRouter(handlers *Handlers) *chi.Mux {
	router := chi.NewRouter()
	router.Get(config.PingPath, handlers.PingHandler)
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.UpdatesPath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricsBatchHandler)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByURL)
			r.Post("/", handlers.GetValueHandlerByJSON)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.expectedStatusCode == http.StatusOK {
				mockService.EXPECT().UpdateMetric(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, nil)
			req.Header.Set(config.ContentTypeHeader, config.ContentTypeTextPlain)

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.method == http.MethodGet && test.expectedStatusCode == http.StatusOK {
				mockService.EXPECT().GetAllMetrics(gomock.Any()).Return([]*model.Metrics{}, nil).AnyTimes()
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
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

func TestGetValueHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		expectedStatusCode int
	}{
		{
			name:               "valid GET request",
			method:             http.MethodGet,
			path:               "/value/gauge/test_gauge",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid method - POST",
			method:             http.MethodPost,
			path:               "/value/gauge/test_gauge",
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "invalid path - missing parts",
			method:             http.MethodGet,
			path:               "/gauge/value/test",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid metric type",
			method:             http.MethodGet,
			path:               "/value/invalid/test_gauge",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid metric name",
			method:             http.MethodGet,
			path:               "/value/gauge/test",
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.method == http.MethodGet && test.expectedStatusCode == http.StatusNotFound {
				mockService.EXPECT().GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}

func TestUpdateMetricHandlerByJSON(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		contentType        string
		requestBody        string
		expectedStatusCode int
	}{
		{
			name:               "valid gauge JSON request",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test_gauge", "type": "gauge", "value": 3.14}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "valid counter JSON request",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test_counter", "type": "counter", "delta": 42}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "invalid method - GET",
			method:             http.MethodGet,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "gauge", "value": 1.0}`,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "invalid content type - text/plain",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeTextPlain,
			requestBody:        `{"id": "test", "type": "gauge", "value": 1.0}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid JSON - malformed",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "gauge", "value": 1.0`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid JSON - missing required fields",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test"}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid metric type in JSON",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "invalid", "value": 1.0}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "empty request body",
			method:             http.MethodPost,
			path:               "/update",
			contentType:        config.ContentTypeJSON,
			requestBody:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.expectedStatusCode == http.StatusOK {
				mockService.EXPECT().UpdateMetricByStruct(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.requestBody))
			req.Header.Set(config.ContentTypeHeader, test.contentType)

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}

func TestGetValueHandlerByJSON(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		contentType        string
		requestBody        string
		expectedStatusCode int
	}{
		{
			name:               "valid gauge JSON request",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test_gauge", "type": "gauge"}`,
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "valid counter JSON request",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test_counter", "type": "counter"}`,
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid method - GET",
			method:             http.MethodGet,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "gauge"}`,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "invalid content type - text/plain",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeTextPlain,
			requestBody:        `{"id": "test", "type": "gauge"}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid JSON - malformed",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "gauge"`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid JSON - missing required fields",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test"}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid metric type in JSON",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        `{"id": "test", "type": "invalid"}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "empty request body",
			method:             http.MethodPost,
			path:               "/value",
			contentType:        config.ContentTypeJSON,
			requestBody:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.expectedStatusCode == http.StatusNotFound {
				mockService.EXPECT().GetMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.requestBody))
			req.Header.Set(config.ContentTypeHeader, test.contentType)

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		pingError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful ping",
			method:         http.MethodGet,
			pingError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "ping failure",
			method:         http.MethodGet,
			pingError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Database unavailable\n",
		},
		{
			name:           "invalid method POST",
			method:         http.MethodPost,
			pingError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name:           "invalid method PUT",
			method:         http.MethodPut,
			pingError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if tt.method == http.MethodGet {
				mockService.EXPECT().Ping(gomock.Any()).Return(tt.pingError).Times(1)
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())

			req := httptest.NewRequest(tt.method, config.PingPath, nil)
			rr := httptest.NewRecorder()

			handlers.PingHandler(rr, req)

			require.Equal(t, tt.expectedStatus, rr.Code, "Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				require.Equal(t, tt.expectedBody, rr.Body.String(), "Expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestPingHandlerWithNilService(t *testing.T) {
	handlers := &Handlers{
		service: nil,
		logger:  zaptest.NewLogger(t).Sugar(),
	}

	req := httptest.NewRequest(http.MethodGet, config.PingPath, nil)
	rr := httptest.NewRecorder()

	handlers.PingHandler(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "Internal server error\n", rr.Body.String())
}

func TestUpdateMetricsBatchHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		contentType        string
		requestBody        string
		expectedStatusCode int
		setupMock          func(*mocks.MockService)
	}{
		{
			name:               "valid batch request",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `[{"id":"counter1","type":"counter","delta":10},{"id":"gauge1","type":"gauge","value":3.14}]`,
			expectedStatusCode: http.StatusOK,
			setupMock: func(mockService *mocks.MockService) {
				mockService.EXPECT().UpdateMetricsBatchByStruct(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:               "invalid method GET",
			method:             http.MethodGet,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `[]`,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "invalid content type",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        "text/plain",
			requestBody:        `[{"id":"counter1","type":"counter","delta":10}]`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid JSON",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `invalid json`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "empty batch",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `[]`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "service error",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `[{"id":"counter1","type":"counter","delta":10}]`,
			expectedStatusCode: http.StatusInternalServerError,
			setupMock: func(mockService *mocks.MockService) {
				mockService.EXPECT().UpdateMetricsBatchByStruct(gomock.Any(), gomock.Any()).Return(errors.New("service error")).Times(1)
			},
		},
		{
			name:               "invalid metric in batch",
			method:             http.MethodPost,
			path:               config.UpdatesPath,
			contentType:        config.ContentTypeJSON,
			requestBody:        `[{"id":"counter1","type":"counter"}]`,
			expectedStatusCode: http.StatusInternalServerError,
			setupMock: func(mockService *mocks.MockService) {
				mockService.EXPECT().UpdateMetricsBatchByStruct(gomock.Any(), gomock.Any()).Return(errors.New("validation failed")).Times(1)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			if test.setupMock != nil {
				test.setupMock(mockService)
			}

			handlers := NewHandlers(mockService, zaptest.NewLogger(t).Sugar())
			router := setupTestRouter(handlers)

			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.requestBody))
			req.Header.Set(config.ContentTypeHeader, test.contentType)

			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)
			require.Equal(t, test.expectedStatusCode, rr.Code, "Expected status code %d, got %d", test.expectedStatusCode, rr.Code)
		})
	}
}
