// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_String(t *testing.T) {
	settings := Settings{
		ListeningAddress: "localhost:6600",
		BlockProfileRate: 1,
		MutexProfileRate: 2,
	}
	expected := "listening on localhost:6600 and setting block profile rate to 1, mutex profile rate to 2"
	assert.Equal(t, expected, settings.String())
}
