// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net/http"

	"golang.org/x/oauth2"
)

// NewOAUTHWrapper returns a TransportWrapper which adds
// OAUTH2 authentication to a HTTP transport.
func NewOAUTHWrapper(opts ...OAUTHOption) *OAUTHWrapper {
	var cfg OAUTHConfig

	cfg.Option(opts...)

	return &OAUTHWrapper{
		transport: oauth2.Transport{
			Source: cfg.source,
		},
	}
}

type OAUTHWrapper struct {
	transport oauth2.Transport
}

func (w *OAUTHWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	return w.transport.RoundTrip(req)
}

func (w *OAUTHWrapper) Wrap(rt http.RoundTripper) http.RoundTripper {
	w.transport.Base = rt

	return w
}

type OAUTHConfig struct {
	source oauth2.TokenSource
}

func (c *OAUTHConfig) Option(opts ...OAUTHOption) {
	for _, opt := range opts {
		opt.ConfigureOAUTH(c)
	}
}

type OAUTHOption interface {
	ConfigureOAUTH(*OAUTHConfig)
}

// WithAccessToken configures a OAUTHWrapper with an OAUTH2 token
// used when making requests.
type WithAccessToken string

func (at WithAccessToken) ConfigureOAUTH(c *OAUTHConfig) {
	c.source = oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: string(at),
	})
}
