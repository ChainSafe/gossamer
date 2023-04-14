package parachaininteraction

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	parachaintypes "github.com/ChainSafe/gossamer/parachain-interaction/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestValidateFromChainState(t *testing.T) {
	// https://polkadot.js.org/apps/?rpc=wss%3A%2F%2Frpc.polkadot.io#/explorer/query/0xebf6e4c13a92e4c92cfa9129ad7f4f86d469ca189e5eefefbf7df609023648fd

	collatorID, err := sr25519.NewPublicKey(common.MustHexToBytes("0xd0e5980907942337b78520a0def9b13d805f54979bfd2729a68fc84be8d5ca04"))
	require.NoError(t, err)

	b, err := common.HexToBytes("0x200cf5b73430a579ac99ecd4c71f45919b0ab2f836521b4897cd0ec2d18717698e761b312187d8b38610d91f568e60a1fc49d4e22fecb5623f81ec7092c9908f")
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], b)
	candidateReceipt := CandidateReceipt{
		descriptor: CandidateDescriptor{
			ParaID:                      uint32(1000),
			RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"),
			Collator:                    *collatorID,
			PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"),
			PovHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"),
			ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"),
			Signature:                   collatorSignature(signature),
			ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"),
			ValidationCodeHash:          validationCodeHash(common.MustHexToHash("0x9985e134020e8a1e2e211afcd5ac9ec6a2fd21dfb5e1c39b3b670f4415e90406")),
		},

		// TODO: we might have to change this value
		commitmentsHash: common.MustHexToHash("0xa54a8dce5fd2a27e3715f99e4241f674a48f4529f77949a4474f5b283b823535"),

		// {
		// 	descriptor: {
		// 	  paraId: 1,000
		// 	  relayParent: 0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0
		// 	  collatorId: 0xd0e5980907942337b78520a0def9b13d805f54979bfd2729a68fc84be8d5ca04
		// 	  persistedValidationDataHash: 0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37
		// 	  povHash: 0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274
		// 	  erasureRoot: 0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1
		// 	  signature: 0x200cf5b73430a579ac99ecd4c71f45919b0ab2f836521b4897cd0ec2d18717698e761b312187d8b38610d91f568e60a1fc49d4e22fecb5623f81ec7092c9908f
		// 	  paraHead: 0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5
		// 	  validationCodeHash: 0x9985e134020e8a1e2e211afcd5ac9ec6a2fd21dfb5e1c39b3b670f4415e90406
		// 	}
		// 	commitments: {
		// 	  upwardMessages: []
		// 	  horizontalMessages: []
		// 	  newValidationCode: null
		// 	  headData: 0xfccb8b3b969d7fe3b059ef2aa1bf8b920ea9ea96c5a324f3c1b52a2a2ffa32b19652da00b1ee1efee107780b1fc76b0784a122f4182d555b35675b8dabe0232a06ff9f305008ea9609ef2a4814a663cad90e67ac55c2da7c85b0162dfe67b3a89368c0f2080661757261209959590800000000056175726101011062ae92e8323126eec6a49d626e8976d7465eee05f103ddaea3360449461a70108fad1874460ceffcbdde1e97bbeb2d545b0fa9547ddb07ba7701e1743bc602
		// 	  processedDownwardMessages: 0
		// 	  hrmpWatermark: 14,987,615
		// 	}
		//   }
	}

	ctrl := gomock.NewController(t)
	mockInstance := NewMockRuntimeInstance(ctrl)

	persistedValidationData := parachaintypes.PersistedValidationData{
		ParentHead:             headData([]byte{7, 8, 9}),
		RelayParentNumber:      uint32(0),
		RelayParentStorageRoot: common.MustHexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		MaxPovSize:             uint32(1024),
	}

	fileContent, err := ioutil.ReadFile("test-validation-code.txt")
	require.NoError(t, err)

	validationCodeBytes, err := common.HexToBytes(strings.TrimSpace(string(fileContent)))
	require.NoError(t, err)

	validationCode := parachaintypes.ValidationCode(validationCodeBytes)
	// validationCode := ValidationCode([]byte{1, 2, 3})

	mockInstance.EXPECT().ParachainHostPersistedValidationData(uint32(1000), gomock.Any()).Return(&persistedValidationData, nil)
	mockInstance.EXPECT().ParachainHostValidationCode(uint32(1000), gomock.Any()).Return(&validationCode, nil)

	// get PersistedValidationData and ValidationCode from polkadot test
	// candidateCommitment, persistedValidationData, err := ValidateFromChainState(mockInstance, candidateReceipt)
	_, _, err = ValidateFromChainState(mockInstance, candidateReceipt)
	require.NoError(t, err)

}

// sample validation code hash 0x9985e134020e8a1e2e211afcd5ac9ec6a2fd21dfb5e1c39b3b670f4415e90406
