// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package testutils

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func ServerFixture() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/status", statusHandler)

	return httptest.NewServer(handler)
}

func statusHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)

		return
	}

	code, err := strconv.Atoi(req.FormValue("code"))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse code: %v", err), http.StatusBadRequest)

		return
	}

	w.WriteHeader(code)
}

func MockRequest(t *testing.T, method string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, "", body)
	require.NoError(t, err)

	return req
}

type MockRoundTripper struct {
	mock.Mock
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)

	return args.Get(0).(*http.Response), args.Error(1)
}
