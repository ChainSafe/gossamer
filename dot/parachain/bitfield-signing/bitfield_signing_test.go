package bitfield_signing

import (
	"errors"
	"fmt"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestBitfieldOrderGuard(t *testing.T) {
	testcases := []struct {
		desc     string
		testcase []bitfieldData
		result   parachaintypes.BitVec
	}{
		{
			desc: "testcaes 1",
			testcase: []bitfieldData{
				{index: 1, data: true},
				{index: 0, data: false},
				{index: 5, data: true},
				{index: 4, data: true},
				{index: 2, data: true},
				{index: 3, data: false}},
			result: parachaintypes.NewBitVec([]bool{false, true, true, false, true, true}),
		},
		{
			desc: "testcaes 2",
			testcase: []bitfieldData{
				{index: 6, data: false},
				{index: 5, data: true},
				{index: 4, data: true},
				{index: 3, data: true},
				{index: 2, data: false},
				{index: 1, data: true}},
			result: parachaintypes.NewBitVec([]bool{true, false, true, true, true, false}),
		},
	}

	for _, tc := range testcases {
		r := bitfieldOrderGuard(tc.testcase)
		assert.Equal(t, tc.result, r, fmt.Sprintf("%s failed", tc.desc))
	}
}

func TestConstructAvailabilityBitfieldFailedParachainHostAvailabilityCores(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(nil, errors.New("something is off")).Times(1)

	testChan := make(chan any)
	bitfield, err := constructAvailabilityBitfield(runtimeMock, parachaintypes.ValidatorIndex(1), testChan)

	assert.Equal(t, bitfield, parachaintypes.BitVec{})
	assert.Error(t, err, "something is off")
}

func TestConstructAvailabilityBitfieldUnsupportType(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	cores := parachaintypes.NewAvailabilityCores()

	core1 := parachaintypes.CoreState{}
	err := core1.SetValue(1)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "unsupported type")

	cores = append(cores, core1)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(cores, nil).Times(1)

	testChan := make(chan any)
	bitfield, err := constructAvailabilityBitfield(runtimeMock, parachaintypes.ValidatorIndex(1), testChan)
	assert.Equal(t, bitfield, parachaintypes.BitVec{})
	assert.Error(t, scale.ErrUnsupportedVaryingDataTypeValue)
}

func TestConstructAvailabilityBitfield(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	cores := parachaintypes.NewAvailabilityCores()

	core1 := parachaintypes.CoreState{}
	err := core1.SetValue(parachaintypes.ScheduledCore{ParaID: 1})

	core2 := parachaintypes.CoreState{}
	err = core2.SetValue(parachaintypes.ScheduledCore{ParaID: 2})
	assert.Nil(t, err)

	core3 := parachaintypes.CoreState{}
	err = core3.SetValue(parachaintypes.OccupiedCore{CandidateHash: common.NewHash([]byte{1, 2, 3, 4, 5})})
	assert.Nil(t, err)

	core4 := parachaintypes.CoreState{}
	err = core4.SetValue(parachaintypes.Free{})
	assert.Nil(t, err)

	core5 := parachaintypes.CoreState{}
	err = core5.SetValue(parachaintypes.OccupiedCore{CandidateHash: common.NewHash([]byte{6, 7, 8, 9, 10})})

	cores = append(cores, core1, core2, core3, core4, core5)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(cores, nil).Times(1)

	subSystemToOverseerTestChan := make(chan any)
	go func() {
		for {
			request, ok := <-subSystemToOverseerTestChan
			if !ok {
				break
			}
			request.(availabilitystore.QueryChunkAvailability).Sender <- true
		}
	}()

	bitfield, err := constructAvailabilityBitfield(runtimeMock, parachaintypes.ValidatorIndex(1), subSystemToOverseerTestChan)

	close(subSystemToOverseerTestChan)

	assert.Nil(t, err)
	assert.Equal(t, parachaintypes.NewBitVec([]bool{false, false, true, false, true}), bitfield)
}
