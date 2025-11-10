package retry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestIsRetriableHTTPError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context.DeadlineExceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "net.ErrClosed",
			err:      net.ErrClosed,
			expected: true,
		},
		{
			name:     "timeout error",
			err:      &net.OpError{Err: &timeoutError{}},
			expected: true,
		},
		{
			name:     "ECONNREFUSED",
			err:      &net.OpError{Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "ETIMEDOUT",
			err:      &net.OpError{Err: syscall.ETIMEDOUT},
			expected: true,
		},
		{
			name:     "EHOSTUNREACH",
			err:      &net.OpError{Err: syscall.EHOSTUNREACH},
			expected: true,
		},
		{
			name:     "ENETUNREACH",
			err:      &net.OpError{Err: syscall.ENETUNREACH},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetriableHTTPError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRetriableHTTPResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: http.StatusBadGateway,
			expected:   true,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "504 Gateway Timeout",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "501 Not Implemented",
			statusCode: http.StatusNotImplemented,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode}
			result := IsRetriableHTTPResponse(resp)
			require.Equal(t, tt.expected, result)
		})
	}

	t.Run("nil response", func(t *testing.T) {
		result := IsRetriableHTTPResponse(nil)
		require.False(t, result)
	})
}

func TestIsRetriablePostgresError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context.DeadlineExceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name: "ConnectionException",
			err: &pgconn.PgError{
				Code: pgerrcode.ConnectionException,
			},
			expected: true,
		},
		{
			name: "ConnectionDoesNotExist",
			err: &pgconn.PgError{
				Code: pgerrcode.ConnectionDoesNotExist,
			},
			expected: true,
		},
		{
			name: "ConnectionFailure",
			err: &pgconn.PgError{
				Code: pgerrcode.ConnectionFailure,
			},
			expected: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			err: &pgconn.PgError{
				Code: pgerrcode.SQLClientUnableToEstablishSQLConnection,
			},
			expected: true,
		},
		{
			name: "SQLServerRejectedEstablishmentOfSQLConnection",
			err: &pgconn.PgError{
				Code: pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
			},
			expected: true,
		},
		{
			name: "TransactionResolutionUnknown",
			err: &pgconn.PgError{
				Code: pgerrcode.TransactionResolutionUnknown,
			},
			expected: true,
		},
		{
			name: "ProtocolViolation",
			err: &pgconn.PgError{
				Code: pgerrcode.ProtocolViolation,
			},
			expected: true,
		},
		{
			name: "UniqueViolation (not retriable)",
			err: &pgconn.PgError{
				Code: pgerrcode.UniqueViolation,
			},
			expected: false,
		},
		{
			name: "IntegrityConstraintViolation (not retriable)",
			err: &pgconn.PgError{
				Code: pgerrcode.IntegrityConstraintViolation,
			},
			expected: false,
		},
		{
			name:     "generic error (not pgconn.PgError)",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetriablePostgresError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryWithBackoff_SuccessOnFirstAttempt(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := RetryWithBackoff(context.Background(), logger, IsRetriableHTTPError, fn)
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
}

func TestRetryWithBackoff_SuccessOnRetry(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return &net.OpError{Err: syscall.ECONNREFUSED}
		}
		return nil
	}

	err := RetryWithBackoff(context.Background(), logger, IsRetriableHTTPError, fn)
	require.NoError(t, err)
	require.Equal(t, 2, attempts)
}

func TestRetryWithBackoff_NonRetriableError(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	attempts := 0
	nonRetriableErr := errors.New("non-retriable error")
	fn := func() error {
		attempts++
		return nonRetriableErr
	}

	err := RetryWithBackoff(context.Background(), logger, IsRetriableHTTPError, fn)
	require.Error(t, err)
	require.Equal(t, nonRetriableErr, err)
	require.Equal(t, 1, attempts)
}

func TestRetryWithBackoff_AllRetriesExhausted(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	attempts := 0
	retriableErr := &net.OpError{Err: syscall.ECONNREFUSED}
	fn := func() error {
		attempts++
		return retriableErr
	}

	err := RetryWithBackoff(context.Background(), logger, IsRetriableHTTPError, fn)
	require.Error(t, err)
	require.Equal(t, retriableErr, err)
	require.Equal(t, 4, attempts)
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	retriableErr := &net.OpError{Err: syscall.ECONNREFUSED}
	fn := func() error {
		attempts++
		return retriableErr
	}

	err := RetryWithBackoff(ctx, logger, IsRetriableHTTPError, fn)
	require.Error(t, err)
	require.Equal(t, retriableErr, err)
	require.Equal(t, 1, attempts)
}

func TestRetryWithBackoffHTTP_SuccessOnFirstAttempt(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	attempts := 0
	fn := func() (*http.Response, error) {
		attempts++
		resp, err := http.Get(server.URL)
		return resp, err
	}

	resp, err := RetryWithBackoffHTTP(context.Background(), logger, fn)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, attempts)
	resp.Body.Close()
}

func TestRetryWithBackoffHTTP_RetriableStatus(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	fn := func() (*http.Response, error) {
		resp, err := http.Get(server.URL)
		return resp, err
	}

	resp, err := RetryWithBackoffHTTP(context.Background(), logger, fn)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 2, attempts)
	resp.Body.Close()
}

func TestRetryWithBackoffHTTP_NonRetriableStatus(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	attempts := 0
	fn := func() (*http.Response, error) {
		attempts++
		resp, err := http.Get(server.URL)
		return resp, err
	}

	resp, err := RetryWithBackoffHTTP(context.Background(), logger, fn)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, 1, attempts)
	resp.Body.Close()
}
