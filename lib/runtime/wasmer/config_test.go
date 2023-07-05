// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Config_SetTestVersion(t *testing.T) {
	t.Run("panics_with_nil_*testing.T", func(t *testing.T) {
		var c Config
		assert.PanicsWithValue(t,
			"*testing.T argument cannot be nil. Please don't use this function outside of Go tests.",
			func() {
				c.SetTestVersion(nil, runtime.Version{})
			})
	})

	t.Run("set_test_version", func(t *testing.T) {
		var c Config
		testVersion := runtime.Version{
			StateVersion: 1,
		}

		c.SetTestVersion(t, testVersion)

		require.NotNil(t, c.testVersion)
		assert.Equal(t, testVersion, *c.testVersion)
	})
}
