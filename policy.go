package client

import (
	"net/http"
	"strings"
)

type RetryPolicy interface {
	IsErrorRetryable(error) bool
	IsStatusRetryableForMethod(string, int) bool
}

func NewDefaultRetryPolicy() DefaultRetryPolicy {
	return DefaultRetryPolicy{}
}

type DefaultRetryPolicy struct{}

func (p DefaultRetryPolicy) IsErrorRetryable(err error) bool {
	if err == nil {
		return true
	}

	switch msg := err.Error(); {
	case msgInRetryPatterns(msg):
		return true
	default:
		return false
	}
}

func (p DefaultRetryPolicy) IsStatusRetryableForMethod(method string, code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,    // 429
		http.StatusServiceUnavailable: // 503
		return true
	case http.StatusInternalServerError, // 500
		http.StatusBadGateway,     // 502
		http.StatusGatewayTimeout: // 504
		return isMethodIdempotent(method)
	default:
		return false
	}
}

func msgInRetryPatterns(msg string) bool {
	retryPatterns := []string{
		"connection refused",
		"connection reset",
		"EOF",
		"PROTOCOL_ERROR",
		"REFUSED_STREAM",
	}

	for _, pat := range retryPatterns {
		if !strings.Contains(msg, pat) {
			continue
		}

		return true
	}

	return false
}

func isMethodIdempotent(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch:
		return false
	default:
		return true
	}
}
