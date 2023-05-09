package parachains

import (
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachains/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func createTestCandidateReceiptAndValidationCode(t *testing.T) (CandidateReceipt, parachaintypes.ValidationCode) {
	// this wasm was achieved by building polkadot's adder test parachain
	runtimeFilePath := "./test-data/test_parachain_adder.wasm"
	validationCodeBytes, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	validationCode := parachaintypes.ValidationCode(validationCodeBytes)

	validationCodeHashV := common.MustBlake2bHash(validationCodeBytes)

	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	collatorID, err := sr25519.NewPublicKey(collatorKeypair.Public().Encode())
	require.NoError(t, err)

	candidateReceipt := CandidateReceipt{
		descriptor: CandidateDescriptor{
			ParaID:                      uint32(1000),
			RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"),
			Collator:                    *collatorID,
			PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"),
			PoVHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"),
			ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"),
			ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"),
			ValidationCodeHash:          validationCodeHash(validationCodeHashV),
		},

		commitmentsHash: common.MustHexToHash("0xa54a8dce5fd2a27e3715f99e4241f674a48f4529f77949a4474f5b283b823535"),
	}

	payload, err := candidateReceipt.descriptor.createSignaturePayload()
	require.NoError(t, err)

	signatureBytes, err := collatorKeypair.Sign(payload)
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], signatureBytes)

	candidateReceipt.descriptor.Signature = collatorSignature(signature)

	return candidateReceipt, validationCode
}

func TestValidateFromChainState(t *testing.T) {
	// https://polkadot.js.org/apps/?rpc=wss%3A%2F%2Frpc.polkadot.io#/explorer/query/0xebf6e4c13a92e4c92cfa9129ad7f4f86d469ca189e5eefefbf7df609023648fd

	candidateReceipt, validationCode := createTestCandidateReceiptAndValidationCode(t)

	bd, err := scale.Marshal(BlockDataInAdderParachain{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)

	pov := PoV{
		BlockData: bd,
	}

	// NOTE: adder parachain internally compares postState with bd.State in it's validate_block,
	// so following is necessary.
	encoded_state, err := scale.Marshal(uint64(1))
	require.NoError(t, err)
	postState, err := common.Keccak256(encoded_state)
	require.NoError(t, err)

	hd, err := scale.Marshal(HeadDataInAdderParachain{
		Number:     uint64(1),
		ParentHash: common.MustHexToHash("0x0102030405060708090001020304050607080900010203040506070809000102"),
		PostState:  postState,
	})
	require.NoError(t, err)

	persistedValidationData := parachaintypes.PersistedValidationData{
		ParentHead:             hd,
		RelayParentNumber:      uint32(1),
		RelayParentStorageRoot: common.MustHexToHash("0x50c969706800c0e9c3c4565dc2babb25e4a73d1db0dee1bcf7745535a32e7ca1"),
		MaxPovSize:             uint32(2048),
	}

	ctrl := gomock.NewController(t)

	mockInstance := NewMockRuntimeInstance(ctrl)
	mockInstance.EXPECT().ParachainHostPersistedValidationData(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).Return(&persistedValidationData, nil)
	mockInstance.EXPECT().ParachainHostValidationCode(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.OccupiedCoreAssumption{})).Return(&validationCode, nil)
	mockInstance.EXPECT().ParachainHostCheckValidationOutputs(uint32(1000), gomock.AssignableToTypeOf(parachaintypes.CandidateCommitments{})).Return(true, nil)

	mockPoVRequestor := NewMockPoVRequestor(ctrl)
	mockPoVRequestor.EXPECT().RequestPoV(common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274")).Return(pov)

	_, _, _, err = ValidateFromChainState(mockInstance, mockPoVRequestor, candidateReceipt)
	require.NoError(t, err)

}

type HeadDataInAdderParachain struct {
	Number     uint64
	ParentHash [32]byte
	PostState  [32]byte
}

type BlockDataInAdderParachain struct {
	State uint64
	Add   uint64
}
