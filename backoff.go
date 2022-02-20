package client

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type BackoffGenerator func() backoff.BackOff

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

type WithInitialInterval time.Duration

func (w WithInitialInterval) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.InitialInterval = time.Duration(w)
}

type WithMaxElapsedTime time.Duration

func (w WithMaxElapsedTime) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.MaxElapsedTime = time.Duration(w)
}

type WithRandomizationFactor float64

func (w WithRandomizationFactor) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.RandomizationFactor = float64(w)
}

type WithMultiplier float64

func (w WithMultiplier) ConfigureExponentialBackoff(bo *backoff.ExponentialBackOff) {
	bo.Multiplier = float64(w)
}

func ConstantBackoffGenerator(d time.Duration) func() backoff.BackOff {
	return func() backoff.BackOff {
		return backoff.NewConstantBackOff(d)
	}
}

func NoBackoffGenerator() func() backoff.BackOff {
	return func() backoff.BackOff {
		return &backoff.ZeroBackOff{}
	}
}
