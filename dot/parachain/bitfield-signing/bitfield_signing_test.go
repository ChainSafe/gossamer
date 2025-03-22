// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfieldsigning

import (
	"context"
	"errors"
	"testing"
	"time"

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
)

func TestConstructAvailabilityBitfieldFailedParachainHostAvailabilityCores(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(nil, errors.New("something is off")).Times(1)

	testChan := make(chan any)
	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1),
		testChan)

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
	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1),
		testChan)
	assert.Equal(t, bitfield, parachaintypes.BitVec{})
	assert.Equal(t, err, scale.ErrUnsupportedVaryingDataTypeValue)
}

func TestConstructAvailabilityBitfieldSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtimeMock := NewMockInstance(ctrl)

	cores := parachaintypes.NewAvailabilityCores()

	core1 := parachaintypes.CoreState{}
	err := core1.SetValue(parachaintypes.ScheduledCore{ParaID: 1})
	assert.Nil(t, err)

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
	assert.Nil(t, err)

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

	bitfield, err := constructAvailabilityBitfield(context.Background(), runtimeMock, parachaintypes.ValidatorIndex(1),
		subSystemToOverseerTestChan)

	close(subSystemToOverseerTestChan)

	assert.Nil(t, err)

	expectedBitfield, err := parachaintypes.NewBitVec([]bool{false, false, true, false, true})
	assert.Nil(t, err)

	assert.True(t, bitfield.IsEqual(&expectedBitfield))
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
	assert.Nil(t, err)

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
	assert.Nil(t, err)

	cores = append(cores, core1, core2, core3, core4, core5)

	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(1), nil).Times(1)

	runtimeMock.EXPECT().ParachainHostAvailabilityCores().Return(cores, nil).Times(1)

	go func() {
		for {
			request, ok := <-testSubSystemToOverseerChan
			if !ok {
				break
			}
			switch request := request.(type) {
			case availabilitystore.QueryChunkAvailability:
				request.Sender <- true
			case parachaintypes.DistributeBitfield:
				assert.Equal(t, common.Hash{1, 2, 3, 4, 5}, request.RelayParent)
				assert.Equal(t, parachaintypes.ValidatorIndex(0), request.Bitfield.ValidatorIndex) // only alice here

				expectedBitfield, err := parachaintypes.NewBitVec([]bool{false, false, true, false, true})
				assert.Nil(t, err)

				assert.True(t, request.Bitfield.Payload.IsEqual(&expectedBitfield))
				assert.EqualValues(t, 64, len(request.Bitfield.Signature)) // signature is not empty
			}
		}
	}()

	err = handleActiveLeavesUpdate(context.Background(), testBitfieldSigningSubsystem, testActiveLeaves)
	assert.Nil(t, err)

	// wait for the DistributeBitfield content checks
	time.Sleep(1 * time.Second)

	close(testSubSystemToOverseerChan)
}
