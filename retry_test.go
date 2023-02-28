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
			actual := policy.IsErrorRetryable(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// TestMsgInRetryPatterns tests the msgInRetryPatterns fuction.
func TestMsgInRetryPatterns(t *testing.T) {
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
			assert.Equal(t, tc.expected, msgInRetryPatterns(tc.input))
		})
	}
}

// TestClientTrace tests the behavior of the Trace method of a client.
func TestClientTrace(t *testing.T) {
	srv := testutils.ServerFixture()
	defer srv.Close()

	// Modify the handler to handle TRACE requests
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodTrace, r.Method, "Unexpected HTTP method")

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("TRACE request received"))
		assert.NoError(t, err, "Error writing response body")
	})

	// Create a new client instance and make a TRACE request to the test server
	client := NewClient()

	resp, err := client.Trace(context.Background(), srv.URL)
	require.NoError(t, err, "Unexpected error")
	defer resp.Body.Close()

	// Verify that the response status code is as expected
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}

// TestClientOptions tests the behavior of the Options method of a client instance and
// ensures that the Options medthod of the Client instance behaves correctly when making
// an OPTIONS request to a server.
func TestClientOptions(t *testing.T) {
	srv := testutils.ServerFixture()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodOptions, r.Method, "Unexpected HTTP method")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OPTIONS request received"))
		require.NoError(t, err, "Error writing response body")
	})
	defer srv.Close()

	client := NewClient()

	resp, err := client.Options(context.Background(), srv.URL)
	require.NoError(t, err, "Unexpected error")
	defer resp.Body.Close()

	// Verify that the response status code is as expected
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}

// TestClientConnect tests the Connect method of the Client struct. It ensures
// that the Connect method works correctly and is able to make a successful CONNECT
// request to the server.
func TestClientConnect(t *testing.T) {
	srv := testutils.ServerFixture()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodConnect, r.Method, "Expected CONNECT method")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("CONNECT request received"))
		require.NoError(t, err)
	})
	defer srv.Close()

	client := NewClient()

	resp, err := client.Connect(context.Background(), srv.URL, nil)
	require.NoError(t, err, "Unexpected error")
	defer resp.Body.Close()

	// Verify that the response status code is as expected
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}

// TestClientDelete tests the Delete method of a HTTP client.
func TestClientDelete(t *testing.T) {
	srv := testutils.ServerFixture()

	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method, "Expected DELETE method")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("DELETE request received"))
		require.NoError(t, err)
	})

	defer srv.Close()

	client := NewClient()
	resp, err := client.Delete(context.Background(), srv.URL)
	require.NoError(t, err, "Unexpected error")
	defer resp.Body.Close()

	// Verify that the response status code is as expected
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
}
