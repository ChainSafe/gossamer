// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package services

import (
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestServiceRegistry_RegisterService(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))

	r.RegisterService(&MockService{})
	r.RegisterService(&MockService{})

	require.Len(t, r.services, 1)
}

func TestServiceRegistry_StartStopAll(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))
	ctrl := gomock.NewController(t)
	m := NewMockService(ctrl)
	m.EXPECT().Start().Return(nil)
	m.EXPECT().Pause().Return(nil)
	m.EXPECT().Stop().Return(nil)

	r.RegisterService(m)

	r.StartAll()
	r.StopAll()
}

func TestServiceRegistry_Get_Err(t *testing.T) {
	r := NewServiceRegistry(log.New(log.SetWriter(io.Discard)))

	a := NewMockService(nil)

	r.RegisterService(a)
	require.NotNil(t, r.Get(a))

	f := struct{}{}
	require.Nil(t, r.Get(f))
}
