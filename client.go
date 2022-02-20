package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

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

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodGet, url)
}

func (c *Client) Head(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodHead, url)
}

func (c *Client) Post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPost, url, nil)
}

func (c *Client) Put(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPut, url, nil)
}

func (c *Client) Patch(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodPatch, url, nil)
}

func (c *Client) Delete(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodDelete, url)
}

func (c *Client) Connect(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	return c.requestWithBody(ctx, http.MethodConnect, url, nil)
}

func (c *Client) Options(ctx context.Context, url string) (*http.Response, error) {
	return c.requestWithoutBody(ctx, http.MethodOptions, url)
}

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

type WithTransport struct{ http.RoundTripper }

func (t WithTransport) ConfigureClient(c *ClientConfig) {
	c.Transport = t.RoundTripper
}

type WithWrapper struct{ TransportWrapper }

func (ww WithWrapper) ConfigureClient(c *ClientConfig) {
	c.Wrappers = append(c.Wrappers, ww.TransportWrapper)
}

type TransportWrapper interface {
	Wrap(http.RoundTripper) http.RoundTripper
}
