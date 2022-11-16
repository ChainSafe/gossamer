// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	settings := Settings{}

	server := New(settings)

	expectedServer := &Server{
		settings: Settings{
			ListeningAddress: "localhost:6060",
		},
	}
	assert.Equal(t, expectedServer, server)
}

func Test_Service_StartStop(t *testing.T) {
	t.Parallel()

	server := New(Settings{ListeningAddress: "127.0.0.1:0"})

	err := server.Start()

	require.NoError(t, err)

	err = server.Stop()

	require.NoError(t, err)
}
