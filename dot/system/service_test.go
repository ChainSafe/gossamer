// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package system

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
)

func TestService_ChainName(t *testing.T) {
	svc := newTestService()

	name := svc.ChainName()
	require.Equal(t, "gssmr", name)
}

func TestService_SystemName(t *testing.T) {
	svc := newTestService()

	name := svc.SystemName()
	require.Equal(t, "gossamer", name)
}

func TestService_SystemVersion(t *testing.T) {
	svc := newTestService()
	ver := svc.SystemVersion()
	require.Equal(t, "0.0.1", ver)
}

func TestService_Properties(t *testing.T) {
	expected := map[string]interface{}(nil)

	svc := newTestService()
	props := svc.Properties()
	require.Equal(t, expected, props)
}

func TestService_Start(t *testing.T) {
	svc := newTestService()
	err := svc.Start()
	require.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	svc := newTestService()
	err := svc.Stop()
	require.NoError(t, err)
}

func newTestService() *Service {

	sysInfo := &types.SystemInfo{
		SystemName:    "gossamer",
		SystemVersion: "0.0.1",
	}
	genData := &genesis.Data{
		Name: "gssmr",
	}
	return NewService(sysInfo, genData)
}
