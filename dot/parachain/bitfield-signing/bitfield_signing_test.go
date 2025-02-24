package bitfield_signing

import (
	"context"
	"errors"
	"fmt"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
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
	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1), testChan)

	assert.Equal(t, bitfield, parachaintypes.BitVec{})
	assert.Error(t, err, "something is off")
}

func TestConstructAvailabilityBitfieldFailedUnsupportedType(t *testing.T) {
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
	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1), testChan)
	assert.Equal(t, bitfield, parachaintypes.BitVec{})
	assert.Error(t, scale.ErrUnsupportedVaryingDataTypeValue)
}

func TestConstructAvailabilityBitfieldSuccess(t *testing.T) {
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

	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1), subSystemToOverseerTestChan)

	close(subSystemToOverseerTestChan)

	assert.Nil(t, err)
	assert.Equal(t, parachaintypes.NewBitVec([]bool{false, false, true, false, true}), bitfield)
}

func TestProcessActiveLeavesUpdateSignalActivatedLeafIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testSignal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated:   nil,
		Deactivated: []common.Hash{{1}, {2}, {3}, {4}, {5}},
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	// activatedLeaf == nil
	err = testBitfieldSigningSubsystem.ProcessActiveLeavesUpdateSignal(testSignal)
	assert.Nil(t, err)
}

func TestProcessActiveLeavesUpdateSignalGetRuntimeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testActiveLeaves := &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash{1, 2, 3, 4, 5},
		Number: 1,
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2, 3, 4, 5}).Return(nil, errors.New("something is off")).Times(1)

	// get runtime error
	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)
	assert.EqualError(t, err, "getting runtime: something is off")
}

func TestProcessActiveLeavesUpdateSignalGetParachainHostValidatorsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testActiveLeaves := &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash{1, 2, 3, 4, 5},
		Number: 1,
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2, 3, 4, 5}).Return(runtimeMock, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostValidators().Return(nil, errors.New("something is off with validators")).Times(1)

	// get validators error
	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)
	assert.EqualError(t, err, "getting validators: something is off with validators")
}

func TestProcessActiveLeavesUpdateSignalNotValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testActiveLeaves := &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash{1, 2, 3, 4, 5},
		Number: 1,
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2, 3, 4, 5}).Return(runtimeMock, nil).Times(1)

	// validatorIDs is empty
	runtimeMock.EXPECT().ParachainHostValidators().Return([]parachaintypes.ValidatorID{}, nil).Times(1)

	// current node is not a validator
	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)
	assert.Nil(t, err)
}

func TestProcessActiveLeavesUpdateSignalConstructAvailabilityBitfieldError(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testActiveLeaves := &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash{1, 2, 3, 4, 5},
		Number: 1,
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2, 3, 4, 5}).Return(runtimeMock, nil).Times(1)

	// validatorIDs has Alice
	runtimeMock.EXPECT().ParachainHostValidators().Return([]parachaintypes.ValidatorID{
		parachaintypes.ValidatorID(aliceKeypair.Public().Encode()),
	}, nil).Times(1)

	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(1), nil).Times(1)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(nil, errors.New("something is off")).Times(1)

	// ConstructAvailabilityBitfield error
	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)
	assert.EqualError(t, err, "construct availabilityBitfield: querying availability cores: something is off")
}

func TestProcessActiveLeavesUpdateSignalSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)
	blockAPIMock := mocks.NewMockBlockAPI(ctrl)

	testActiveLeaves := &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash{1, 2, 3, 4, 5},
		Number: 1,
	}

	testSubSystemToOverseerChan := make(chan any)

	testKs := keystore.NewBasicKeystore("test", crypto.Sr25519Type)
	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)

	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	err = testKs.Insert(aliceKeypair)
	assert.Nil(t, err)

	testBitfieldSigningSubsystem := NewBitfieldSigning(testSubSystemToOverseerChan, testKs, blockAPIMock)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{1, 2, 3, 4, 5}).Return(runtimeMock, nil).Times(1)

	// validatorIDs has Alice
	runtimeMock.EXPECT().ParachainHostValidators().Return([]parachaintypes.ValidatorID{
		parachaintypes.ValidatorID(aliceKeypair.Public().Encode()),
	}, nil).Times(1)

	cores := parachaintypes.NewAvailabilityCores()

	core1 := parachaintypes.CoreState{}
	err = core1.SetValue(parachaintypes.ScheduledCore{ParaID: 1})

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

	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(1), nil).Times(1)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(cores, nil).Times(1)

	go func() {
		for {
			request, ok := <-testSubSystemToOverseerChan
			if !ok {
				break
			}
			switch request.(type) {
			case availabilitystore.QueryChunkAvailability:
				request.(availabilitystore.QueryChunkAvailability).Sender <- true
			case parachaintypes.DistributeBitfield:
				a := request.(parachaintypes.DistributeBitfield)
				assert.Equal(t, common.Hash{1, 2, 3, 4, 5}, a.RelayParent)
				assert.Equal(t, parachaintypes.ValidatorIndex(0), a.Bitfield.ValidatorIndex) // only alice is in the validator set now
				assert.Equal(t, parachaintypes.NewBitVec([]bool{false, false, true, false, true}), a.Bitfield.Payload)
				assert.EqualValues(t, 64, len(a.Bitfield.Signature)) // signature is not empty
			}
		}
	}()

	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)

	assert.Nil(t, err)

	// wait for the DistributeBitfield content checks
	time.Sleep(1 * time.Second)

	close(testSubSystemToOverseerChan)
}
