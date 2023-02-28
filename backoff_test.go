package client

import (
	"testing"

	"github.com/cenkalti/backoff/v4"
)

// TestWithRandomizationFactor_ConfigureExponentialBackoff checks if the WithRandomizationFactor
// function correctly sets the randomization factor of an ExponentialBackoff instance
// when used with the ConfigureExponentialBackoff method.
func TestWithRandomizationFactor_ConfigureExponentialBackoff(t *testing.T) {
    bo := backoff.NewExponentialBackOff()

    rf := WithRandomizationFactor(0.5)
    rf.ConfigureExponentialBackoff(bo)

    if bo.RandomizationFactor != 0.5 {
        t.Errorf("RandomizationFactor not set properly. Expected %f, got %f", 0.5, bo.RandomizationFactor)
    }
}

// TestWithMultiplierConfigureExponentialBackoff ensures that the ConfigureExponentialBackoff
// function correctly configures the ExponentialBackoff object with the desired multipler value.
func TestWithMultiplierConfigureExponentialBackoff(t *testing.T) {
    bo := backoff.NewExponentialBackOff()

    w := WithMultiplier(2.0)
    w.ConfigureExponentialBackoff(bo)

    if bo.Multiplier != 2.0 {
        t.Errorf("Expected multiplier to be 2.0, but got %v", bo.Multiplier)
    }
}
