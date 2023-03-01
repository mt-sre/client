package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/mt-sre/client/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient ensures a working http client is returned
// by NewClient.
func TestNewClient(t *testing.T) {
	t.Parallel()

	mrt := &testutils.MockRoundTripper{}

	req := testutils.MockRequest(t, http.MethodGet, nil)

	mrt.
		On("RoundTrip", req).
		Return(&http.Response{
			StatusCode: http.StatusOK,
		}, nil)

	client := NewClient(
		WithTransport{RoundTripper: mrt},
	)

	_, err := client.Get(context.Background(), "")
	require.NoError(t, err)

	mrt.AssertExpectations(t)
}

// TestClientConfig_default ensures that the transport field
// is set correctly.
func TestClientConfig_Default(t *testing.T) {
	t.Parallel()

	cfg := &ClientConfig{
		Transport: nil,
	}
	cfg.Default()

	require.Equal(t, http.DefaultTransport, cfg.Transport, "Transport is not set to http.DefaultTransport")
}

// TestClientTrace tests the behavior of the Trace method of a client.
func TestClientTrace(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

// TestHead ensures the Head method method of the client sends a HEAD request to the
// specified URL and that the response contains the expected status code and an empty body.
func TestClientHead(t *testing.T) {
	t.Parallel()

	srv := testutils.ServerFixture()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Test")
	})

	defer srv.Close()

	client := NewClient()
	resp, err := client.Head(context.Background(), srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify that the response body is empty.
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

// TestPatch function tests the Patch method of the Client struct to ensure that the status code is HTTP 200
// OK and that the response body matches the expected value "test\n".
func TestClientPatch(t *testing.T) {
	t.Parallel()

	srv := testutils.ServerFixture()
	defer srv.Close()

	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Empty(t, body)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test")

	})

	client := NewClient()
	resp, err := client.Patch(context.Background(), srv.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify that the response body matches the expected value.
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "test\n", string(body))
}
