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

package sync

import (
	"github.com/stretchr/testify/mock"

	. "github.com/ChainSafe/gossamer/dot/sync/mocks"
)

func NewMockFinalityGadget() *MockFinalityGadget {
	m := new(MockFinalityGadget)
	// using []uint8 instead of []byte: https://github.com/stretchr/testify/pull/969
	m.On("VerifyBlockJustification", mock.AnythingOfType("[]uint8")).Return(nil)
	return m
}

func NewMockVerifier() *MockVerifier {
	m := new(MockVerifier)
	m.On("VerifyBlock", mock.AnythingOfType("*types.Header")).Return(nil)
	return m
}

func NewBlockProducer() *MockBlockProducer {
	m := new(MockBlockProducer)
	m.On("Pause").Return(nil)
	m.On("Resume").Return(nil)
	m.On("SetRuntime", mock.AnythingOfType("runtime.Instance"))

	return m
}
