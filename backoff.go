// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// BackoffGenerator always returns a new instance of
// backoff.Backoff to ensure a fresh backoff state
// for repeated reqeusts
type BackoffGenerator func() backoff.BackOff

// ExponentialBackoffGenerator returns an ExponentialBackoff instance.
func ExponentialBackoffGenerator(opts ...ExponentialBackoffOption) func() backoff.BackOff {
	return func() backoff.BackOff {
		exp := backoff.NewExponentialBackOff()
		defer exp.Reset()

		for _, opt := range opts {
			opt.ConfigureExponentialBackoff(exp)
		}

		return exp
	}
}

type ExponentialBackoffOption interface {
	ConfigureExponentialBackoff(*backoff.ExponentialBackOff)
}

// WithInitialInterval sets the wait time between the first and second
// request attempts.
type WithInitialInterval time.Duration

func (w WithInitialInterval) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.InitialInterval = time.Duration(w)
}

// WithMaxElapsedTime sets the maximum cumulative time after which retries are no longer
// performed.
type WithMaxElapsedTime time.Duration

func (w WithMaxElapsedTime) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.MaxElapsedTime = time.Duration(w)
}

// WithRandomizationFactor sets the degree to which jitter is applied to successive retries.
type WithRandomizationFactor float64

func (w WithRandomizationFactor) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.RandomizationFactor = float64(w)
}

// WithMultiplier sets the exponential base used for increasing backoff.
type WithMultiplier float64

func (w WithMultiplier) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.Multiplier = float64(w)
}

// ConstantBackoffGenerator returns a backoff with constant intervals between retries
// as set with the parameter 'd'.
func ConstantBackoffGenerator(d time.Duration) func() backoff.BackOff {
	return func() backoff.BackOff {
		return backoff.NewConstantBackOff(d)
	}
}

// NoBackoffGenerator returns a backoff which has no time interval between retries.
func NoBackoffGenerator() func() backoff.BackOff {
	return func() backoff.BackOff {
		return &backoff.ZeroBackOff{}
	}
}
