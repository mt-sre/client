package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-logr/logr"
)

var errTemporary = errors.New("temporary error occurred")

func NewRetryWrapper(opts ...RetryWrapperOption) *RetryWrapper {
	var cfg RetryWrapperConfig

	cfg.Option(opts...)

	cfg.Default()

	if cfg.maxRetries > 0 {
		inner := cfg.GenerateBackoff

		cfg.GenerateBackoff = func() backoff.BackOff {
			return backoff.WithMaxRetries(inner(), cfg.maxRetries)
		}
	}

	return &RetryWrapper{
		cfg: cfg,
	}
}

type RetryWrapper struct {
	cfg RetryWrapperConfig
	rt  http.RoundTripper
}

func (w *RetryWrapper) Wrap(rt http.RoundTripper) http.RoundTripper {
	w.rt = rt

	return w
}

func (w *RetryWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	log := w.cfg.Logger.WithValues(
		"method", req.Method,
		"host", req.URL.Host,
		"path", req.URL.Path,
	)

	// preserve request body so that each request can be made with a readable body
	copy, err := copyRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("copying request body: %w", err)
	}

	retries := 0

	var res *http.Response

	roundtrip := func() error {
		if retries > 0 {
			log.Info("retrying request",
				"retries", retries,
			)
		}

		if copy != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(copy))
		}

		// drain open response body so that existing connections may be reused
		if res != nil {
			drainResponseBody(w.cfg.Logger.V(1), res)
		}

		var err error
		res, err = w.rt.RoundTrip(req)
		if err != nil {
			if !w.cfg.Policy.IsErrorRetryable(err) {
				// exit with error if request failed before a response was received
				return backoff.Permanent(err)
			}

			return errTemporary
		}

		log.Info("received response",
			"responseStatus", res.StatusCode,
		)

		if !w.cfg.Policy.IsStatusRetryableForMethod(req.Method, res.StatusCode) {
			// exit with no error if HTTP status code does not permit retry
			return nil
		}

		retries++

		// exit with temporary error to retry request
		return errTemporary
	}

	bo := backoff.WithContext(w.cfg.GenerateBackoff(), req.Context())

	if err := backoff.Retry(roundtrip, bo); err != nil {
		if !errors.Is(err, errTemporary) && !errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("permanent error encountered: %w", err)
		}
	}

	return res, nil
}

func copyRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	copy, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	if err := req.Body.Close(); err != nil {
		return nil, fmt.Errorf("closing request body: %w", err)
	}

	return copy, nil
}

func drainResponseBody(logger logr.Logger, res *http.Response) {
	defer res.Body.Close()

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		logger.Info("unable to discard response body",
			"error", err,
		)
	}
}

type RetryWrapperConfig struct {
	Logger          logr.Logger
	GenerateBackoff func() backoff.BackOff
	Policy          RetryPolicy
	maxRetries      uint64
}

func (c *RetryWrapperConfig) Option(opts ...RetryWrapperOption) {
	for _, opt := range opts {
		opt.ConfigureRetryWrapper(c)
	}
}

func (c *RetryWrapperConfig) Default() {
	if c.Logger == nil {
		c.Logger = logr.Discard()
	}

	if c.GenerateBackoff == nil {
		c.GenerateBackoff = ExponentialBackoffGenerator()
	}

	if c.Policy == nil {
		c.Policy = NewDefaultRetryPolicy()
	}
}

type RetryWrapperOption interface {
	ConfigureRetryWrapper(*RetryWrapperConfig)
}

type WithLogger struct{ logr.Logger }

func (l WithLogger) ConfigureRetryWrapper(c *RetryWrapperConfig) {
	c.Logger = l.Logger
}

type WithBackoffGenerator func() backoff.BackOff

func (bg WithBackoffGenerator) ConfigureRetryWrapper(c *RetryWrapperConfig) {
	c.GenerateBackoff = bg
}

type WithMaxRetries uint64

func (mr WithMaxRetries) ConfigureRetryWrapper(c *RetryWrapperConfig) {
	c.maxRetries = uint64(mr)
}
