// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package modules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMockStorageAPI(t *testing.T) {
	m := NewMockStorageAPI()
	require.NotNil(t, m)
}

func TestNewMockBlockAPI(t *testing.T) {
	m := NewMockBlockAPI()
	require.NotNil(t, m)
}

func TestNewMockCoreAPI(t *testing.T) {
	m := NewMockCoreAPI()
	require.NotNil(t, m)
}

func TestNewMockVersion(t *testing.T) {
	m := NewMockVersion()
	require.NotNil(t, m)
}
