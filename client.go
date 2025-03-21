// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// NewClient returns an opionanted HTTP client which can be
// optionally augmented with TransportWrappers which add
// features such as retries with exponential backoff.
func NewClient(opts ...ClientOption) *Client {
	var cfg ClientConfig

	cfg.Option(opts...)
	cfg.Default()

	var client http.Client

	cfg.Wrap(&client)

	return &Client{
		cfg:    cfg,
		client: &client,
	}
}

type Client struct {
	cfg    ClientConfig
	client *http.Client
}

// Get performs a HTTP GET request against the provided URL.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodGet, url)
}

// Head performs a HTTP HEAD request against the provided URL.
func (c *Client) Head(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodHead, url)
}

// Post performs a HTTP POST request against the provided URL with the given body.
func (c *Client) Post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPost, url, nil)
}

// Put performs a HTTP PUT request against the provided URL with the given body.
func (c *Client) Put(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPut, url, nil)
}

// Patch performs a HTTP PATCH request against the provided URL with the given body.
func (c *Client) Patch(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPatch, url, nil)
}

// Delete performs a HTTP DELETE request against the provided URL.
func (c *Client) Delete(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodDelete, url)
}

// Connect performs a HTTP CONNECT request against the provided URL with the given body.
func (c *Client) Connect(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodConnect, url, nil)
}

// Options performs a HTTP OPTIONS request against the provided URL.
func (c *Client) Options(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodOptions, url)
}

// Trace performs a HTTP TRACE request against the provided URL.
func (c *Client) Trace(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodTrace, url)
}

func (c *Client) requestWithoutBody(ctx context.Context, method, url string) (*http.Response, error) {
	return c.requestWithBody(ctx, method, url, nil)
}

func (c *Client) requestWithBody(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("constructing request: %w", err)
	}

	return c.client.Do(req)
}

type ClientConfig struct {
	Transport http.RoundTripper
	Wrappers  []TransportWrapper
}

func (c *ClientConfig) Option(opts ...ClientOption) {
	for _, opt := range opts {
		opt.ConfigureClient(c)
	}
}

func (c *ClientConfig) Default() {
	if c.Transport == nil {
		c.Transport = http.DefaultTransport
	}
}

func (c *ClientConfig) Wrap(client *http.Client) {
	tp := c.Transport

	for _, w := range c.Wrappers {
		w.Wrap(tp)
	}

	client.Transport = tp
}

type ClientOption interface {
	ConfigureClient(*ClientConfig)
}

// WithTransport configures a Client instance with the given
// http.RoundTripper instance.
type WithTransport struct{ http.RoundTripper }

func (t WithTransport) ConfigureClient(c *ClientConfig) {
	c.Transport = t.RoundTripper
}

// WithWrapper configures a Client instance with the given
// TransportWrapper. This option can be provided multiple
// times to apply several TransportWrappers. The order in
// which the TransportWrappers is applied is important!
type WithWrapper struct{ TransportWrapper }

func (ww WithWrapper) ConfigureClient(c *ClientConfig) {
	c.Wrappers = append(c.Wrappers, ww.TransportWrapper)
}

// TransportWrapper adds functionality to a http.RoundTripper
// by adding pre and post call execution steps.
type TransportWrapper interface {
	// Wrap return a http.RoundTripper which has been
	// wrapped with the pre and post functionality the
	// TransportWrapper provides.
	Wrap(http.RoundTripper) http.RoundTripper
}
