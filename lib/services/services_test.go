// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package services

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/services/mocks"
	"github.com/stretchr/testify/require"
)

func TestServiceRegistry_RegisterService(t *testing.T) {
	r := NewServiceRegistry()

	r.RegisterService(&mocks.Service{})
	r.RegisterService(&mocks.Service{})

	require.Len(t, r.services, 1)
}

func TestServiceRegistry_StartStopAll(t *testing.T) {
	r := NewServiceRegistry()

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
	r := NewServiceRegistry()

	a := new(mocks.Service)
	a.On("Start").Return(nil)
	a.On("Stop").Return(nil)

	r.RegisterService(a)
	require.NotNil(t, r.Get(a))

	f := struct{}{}
	require.Nil(t, r.Get(f))
}
