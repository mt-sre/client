// SPDX-FileCopyrightText: 2025 Red Hat, Inc. <sd-mt-sre@redhat.com>
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

// TestWithRandomizationFactor_ConfigureExponentialBackoff checks if the WithRandomizationFactor
// function correctly sets the randomization factor of an ExponentialBackoff instance
// when used with the ConfigureExponentialBackoff method.
func TestWithRandomizationFactor_ConfigureExponentialBackoff(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponentialBackOff()

	rf := WithRandomizationFactor(0.5)
	rf.ConfigureExponentialBackoff(bo)

	assert.Equal(t, 0.5, bo.RandomizationFactor, "RandomizationFactor not set properly")
}

// TestWithMultiplierConfigureExponentialBackoff ensures that the ConfigureExponentialBackoff
// function correctly configures the ExponentialBackoff object with the desired multipler value.
func TestWithMultiplierConfigureExponentialBackoff(t *testing.T) {
	t.Parallel()

	bo := backoff.NewExponentialBackOff()

	w := WithMultiplier(2.0)
	w.ConfigureExponentialBackoff(bo)

	assert.Equal(t, 2.0, bo.Multiplier, "Multiplier not set properly")
}
