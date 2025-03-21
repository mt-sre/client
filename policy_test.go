// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/mt-sre/client/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryPolicyInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(RetryPolicy), new(DefaultRetryPolicy))
}

func TestDefaultRetryPolicy(t *testing.T) {
	t.Parallel()

	const numRetries = 1

	type testCase struct {
		Method      string
		StatusCode  int
		ShouldRetry bool
	}

	cases := make(map[string]testCase)

	for _, method := range idempotentHTTPMethods() {
		for _, code := range nonRetryableCodes() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: false,
			}
		}

		for _, code := range retryableCodes() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: true,
			}
		}

		for _, code := range retryableCodesIdempotentOnly() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: true,
			}
		}
	}

	for _, method := range nonIdempotentHTTPMethods() {
		for _, code := range nonRetryableCodes() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: false,
			}
		}

		for _, code := range retryableCodes() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: true,
			}
		}

		for _, code := range retryableCodesIdempotentOnly() {
			name := fmt.Sprintf("%s/%d", method, code)

			cases[name] = testCase{
				Method:      method,
				StatusCode:  code,
				ShouldRetry: false,
			}
		}
	}

	for name, tc := range cases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := testutils.MockRequest(t, tc.Method, nil)

			calls := 1

			if tc.ShouldRetry {
				calls += numRetries
			}

			var mrt testutils.MockRoundTripper

			for i := 0; i < calls; i++ {
				req.Body = io.NopCloser(bytes.NewBuffer([]byte{}))

				mrt.
					On("RoundTrip", req).
					Return(&http.Response{
						StatusCode: tc.StatusCode,
						Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
					}, nil)
			}

			retry := NewRetryWrapper(
				WithBackoffGenerator(NoBackoffGenerator()),
				WithMaxRetries(numRetries),
			)

			var client http.Client
			client.Transport = retry.Wrap(&mrt)

			_, err := client.Do(req)
			require.NoError(t, err)

			mrt.AssertExpectations(t)
		})
	}
}

func retryableCodes() []int {
	return []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
	}
}

func retryableCodesIdempotentOnly() []int {
	return []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusGatewayTimeout,
	}
}

func nonRetryableCodes() []int {
	return []int{
		http.StatusContinue,
		http.StatusSwitchingProtocols,
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusMultipleChoices,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusUnsupportedMediaType,
		http.StatusNotImplemented,
		http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		http.StatusInsufficientStorage,
		http.StatusLoopDetected,
		http.StatusNotExtended,
		http.StatusNetworkAuthenticationRequired,
	}
}

func idempotentHTTPMethods() []string {
	return []string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPut,
		http.MethodTrace,
	}
}

func nonIdempotentHTTPMethods() []string {
	return []string{
		http.MethodPatch,
		http.MethodPost,
	}
}
