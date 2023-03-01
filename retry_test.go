package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/mt-sre/client/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryWrapperInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(http.RoundTripper), new(RetryWrapper))

	require.Implements(t, new(TransportWrapper), new(RetryWrapper))
}

func TestRoundTripResponseBody(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		StatusCode int
		Body       []byte
	}{
		"empty body no retry": {
			StatusCode: http.StatusOK,
			Body:       []byte(""),
		},
		"empty body single retry": {
			StatusCode: http.StatusInternalServerError,
			Body:       []byte(""),
		},
		"non-empty body no retry": {
			StatusCode: http.StatusOK,
			Body:       []byte("test"),
		},
		"non-empty body single retry": {
			StatusCode: http.StatusInternalServerError,
			Body:       []byte("test"),
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := testutils.MockRequest(t, http.MethodGet, nil)

			var mrt testutils.MockRoundTripper
			mrt.
				On("RoundTrip", req).
				Return(&http.Response{
					StatusCode: tc.StatusCode,
					Body: io.NopCloser(
						bytes.NewBuffer(tc.Body),
					),
				}, nil).
				Once()
			mrt.
				On("RoundTrip", req).
				Return(&http.Response{
					StatusCode: tc.StatusCode,
					Body: io.NopCloser(
						bytes.NewBuffer(tc.Body),
					),
				}, nil).
				Maybe()

			retry := NewRetryWrapper(
				WithBackoffGenerator(NoBackoffGenerator()),
				WithMaxRetries(1),
			)

			var client http.Client
			client.Transport = retry.Wrap(&mrt)

			res, err := client.Do(req)
			require.NoError(t, err)

			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.Body, body)

			mrt.AssertExpectations(t)
		})
	}
}

// TestRoundTripWithContext ensures that cancelled requests
// are not retried, but still return the last response received.
func TestRoundTripWithContext(t *testing.T) {
	t.Parallel()

	const (
		delay                 = 10 * time.Millisecond
		requestsBeforeTimeout = 3
	)

	ctx, cancel := context.WithTimeout(context.Background(), delay*requestsBeforeTimeout-1*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	require.NoError(t, err)

	var mrt testutils.MockRoundTripper

	mrt.
		On("RoundTrip", req).
		Return(&http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
		}, nil).
		Times(requestsBeforeTimeout)

	retry := NewRetryWrapper(
		WithBackoffGenerator(
			ConstantBackoffGenerator(delay),
		),
	)

	var client http.Client
	client.Transport = retry.Wrap(&mrt)

	res, err := client.Do(req)
	require.NoError(t, err)

	assert.NotNil(t, res)
}

// TestRoundTripConcurrencySafety ensures that individual
// requests are not using the same backoff instance which
// would cause all requests to stop retrying after the first
// "backoff.Stop" condition is met.
func TestRoundTripConcurrencySafety(t *testing.T) {
	t.Parallel()

	srv := testutils.ServerFixture()
	defer func() {
		srv.CloseClientConnections()
		srv.Close()
	}()

	const initialDelay = 20 * time.Microsecond

	retry := NewRetryWrapper(
		WithBackoffGenerator(
			ExponentialBackoffGenerator(
				WithInitialInterval(initialDelay),
				WithMaxElapsedTime(100*time.Microsecond),
			),
		),
	)

	client := srv.Client()
	client.Transport = retry.Wrap(http.DefaultTransport)

	const sessions = 100

	var wg sync.WaitGroup
	wg.Add(sessions)

	type result struct {
		Delay time.Duration
		Err   error
		Res   *http.Response
	}

	results := make(chan result)

	for i := 0; i < sessions; i++ {
		go func() {
			start := time.Now()

			res, err := client.Get(srv.URL + "/status?code=429")

			results <- result{
				Delay: time.Since(start),
				Err:   err,
				Res:   res,
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		require.NoError(t, res.Err)

		assert.GreaterOrEqual(t, res.Delay, initialDelay, res.Res)
	}
}

// TestDefaultRetryPolicy_IsErrorRetryable ensures that the IsErrorRetryable
// method of DefaultRetryPolicy behaves as expected, correctly identifying
// retryable and non-retryable errors.
func TestDefaultRetryPolicy_IsErrorRetryable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    error
		expected bool
	}{
		{
			name:     "nil error",
			input:    nil,
			expected: true,
		},
		{
			name:     "retryable error",
			input:    errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			input:    errors.New("unknown error"),
			expected: false,
		},
	}

	policy := DefaultRetryPolicy{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := policy.IsErrorRetryable(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestMsgInRetryPatterns tests the msgInRetryPatterns fuction.
func TestMsgInRetryPatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Connection refused",
			input:    "connection refused",
			expected: true,
		},
		{
			name:     "Unknown error",
			input:    "unknown error",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, msgInRetryPatterns(tc.input))
		})
	}
}

// TestWithLogger_configureRetryWrapper checks that the method correctly sets
// the Logger field of a RetryWrapperConfig instance to the logger instance
// sepcified by the WithLogger instance.
func TestWithLogger_ConfigureRetryWrapper(t *testing.T) {
	t.Parallel()

	logger := logr.Discard()

	withLogger := WithLogger{Logger: logger}

	config := &RetryWrapperConfig{}

	withLogger.ConfigureRetryWrapper(config)

	// check that the Logger field is set to the logger instance
	require.Equal(t, logger, config.Logger, "Logger field is not set correctly")
}
