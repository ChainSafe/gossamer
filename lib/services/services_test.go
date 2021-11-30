// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package services

import (
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/services/mocks"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistry_RegisterService(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))

	r.RegisterService(&mocks.Service{})
	r.RegisterService(&mocks.Service{})

	require.Len(t, r.services, 1)
}

func TestServiceRegistry_StartStopAll(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))

	m := new(mocks.Service)
	m.On("Start").Return(nil)
	m.On("Stop").Return(nil)

	r.RegisterService(m)

	r.StartAll()
	m.AssertCalled(t, "Start")

	r.StopAll()
	m.AssertCalled(t, "Stop")
}

func TestServiceRegistry_Get_Err(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))

	a := new(mocks.Service)
	a.On("Start").Return(nil)
	a.On("Stop").Return(nil)

	r.RegisterService(a)
	require.NotNil(t, r.Get(a))

	f := struct{}{}
	require.Nil(t, r.Get(f))
}
