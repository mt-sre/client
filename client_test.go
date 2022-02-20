package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/mt-sre/client/internal/testutils"
)

// TestNewClient ensures a working http client is returned
// by NewClient.
func TestNewClient(t *testing.T) {
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

	client.Get(context.Background(), "")

	mrt.AssertExpectations(t)
}
