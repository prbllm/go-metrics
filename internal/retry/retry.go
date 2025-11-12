package retry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
)

func IsRetriableHTTPError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			if opErr.Err != nil {
				var errno syscall.Errno
				if errors.As(opErr.Err, &errno) {
					switch errno {
					case syscall.ECONNREFUSED, syscall.ETIMEDOUT, syscall.EHOSTUNREACH, syscall.ENETUNREACH:
						return true
					}
				}
			}
		}

		if errors.Is(err, net.ErrClosed) {
			return true
		}

		if netErr.Timeout() {
			return true
		}
	}

	return false
}

func IsRetriableHTTPResponse(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	statusCode := resp.StatusCode
	return statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

func IsRetriablePostgresError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	pgErrCode := pgErr.Code

	return pgErrCode == pgerrcode.ConnectionException ||
		pgErrCode == pgerrcode.ConnectionDoesNotExist ||
		pgErrCode == pgerrcode.ConnectionFailure ||
		pgErrCode == pgerrcode.SQLClientUnableToEstablishSQLConnection ||
		pgErrCode == pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection ||
		pgErrCode == pgerrcode.TransactionResolutionUnknown ||
		pgErrCode == pgerrcode.ProtocolViolation
}

func RetryWithBackoff(ctx context.Context, logger logger.Logger, isRetriable func(error) bool, fn func() error) error {
	var lastErr error

	err := fn()
	if err == nil {
		return nil
	}

	if !isRetriable(err) {
		return err
	}

	lastErr = err
	logger.Warnf("Retriable error occurred, starting retry: %v", err)

	delays := []time.Duration{config.Delay1, config.Delay3, config.Delay5}

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		delay := delays[attempt]
		logger.Debugf("Retry attempt %d/%d after %v delay", attempt+1, config.MaxRetries, delay)

		select {
		case <-ctx.Done():
			logger.Warnf("Context cancelled during retry delay, returning last error")
			return lastErr
		case <-time.After(delay):
		}

		err := fn()
		if err == nil {
			logger.Infof("Retry succeeded after %d attempts", attempt+1)
			return nil
		}

		if !isRetriable(err) {
			logger.Errorf("Non-retriable error occurred during retry: %v", err)
			return err
		}

		lastErr = err
		logger.Warnf("Retry attempt %d/%d failed: %v", attempt+1, config.MaxRetries, err)
	}

	logger.Errorf("All retry attempts exhausted, last error: %v", lastErr)
	return lastErr
}

func RetryWithBackoffHTTP(
	ctx context.Context,
	logger logger.Logger,
	fn func() (*http.Response, error),
) (*http.Response, error) {
	var lastResp *http.Response

	httpFn := func() error {
		resp, err := fn()
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return err
		}

		if IsRetriableHTTPResponse(resp) {
			lastResp = resp
			return &httpError{statusCode: resp.StatusCode}
		}

		lastResp = resp
		return nil
	}

	isRetriableHTTP := func(err error) bool {
		if err == nil {
			return false
		}
		if IsRetriableHTTPError(err) {
			return true
		}
		var httpErr *httpError
		return errors.As(err, &httpErr)
	}

	err := RetryWithBackoff(ctx, logger, isRetriableHTTP, httpFn)
	if err != nil {
		if lastResp != nil && lastResp.Body != nil {
			lastResp.Body.Close()
		}
		return nil, err
	}

	return lastResp, nil
}

type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return http.StatusText(e.statusCode)
}
