// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package availabilitydistribution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewAvailabilityDistribution(t *testing.T) {
	ctrl := gomock.NewController(t)
	netMock := NewMockNetwork(ctrl)
	blockStateMock := NewMockBlockState(ctrl)

	overseerCh := make(chan any)

	ad := NewAvailabilityDistribution(overseerCh, netMock, blockStateMock)

	assert.NotNil(t, ad)
}
