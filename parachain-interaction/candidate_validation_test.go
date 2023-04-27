package parachaininteraction

import (
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	parachaintypes "github.com/ChainSafe/gossamer/parachain-interaction/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
			PoVHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"),
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

	bd, err := scale.Marshal(BlockData{
		State: uint64(1),
		Add:   uint64(1),
	})
	require.NoError(t, err)

	pov := PoV{
		BlockData: bd, //[]byte{81, 201, 53, 192, 127, 19, 238, 98, 79, 205, 228, 10, 179, 251, 158, 207, 52, 216, 172, 73, 103, 67, 28, 78, 89, 243, 72, 167, 68, 55, 171, 48, 4, 243, 127, 239, 111, 183, 196, 107, 211, 117, 189, 178, 167, 60, 244, 232, 43, 127, 66, 173, 161, 133, 81, 249, 39, 150, 99, 231, 165, 239, 45, 157, 254, 43, 147, 43, 189, 160, 51, 247, 131, 81, 116, 140, 187, 94, 154, 130, 29, 160, 114, 135, 104, 45, 241, 56, 184, 121, 36, 58, 92, 104, 173, 26, 87, 0, 8, 133, 7, 4, 1, 0, 137, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 159, 218, 60, 92, 146, 103, 222, 40, 150, 224, 30, 70, 135, 2, 69, 126, 139, 7, 125, 213, 45, 212, 111, 38, 78, 224, 145, 14, 178, 25, 86, 105, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 1, 0, 0, 0, 219, 170, 117, 174, 158, 87, 148, 78, 73, 196, 180, 170, 79, 8, 133, 18, 53, 7, 127, 156, 20, 90, 144, 228, 136, 213, 65, 142, 34, 241, 63, 144, 0, 0, 0, 0, 20, 144, 0, 0, 32, 0, 0, 0, 16, 0, 8, 0, 0, 0, 0, 4, 0, 0, 0, 1, 0, 0, 5, 0, 0, 0, 5, 0, 0, 0, 6, 0, 0, 0, 6, 0, 0, 0, 9, 1, 63, 32, 6, 222, 61, 138, 84, 210, 126, 68, 169, 213, 206, 24, 150, 24, 242, 45, 180, 180, 157, 149, 50, 13, 144, 33, 153, 76, 133, 15, 37, 184, 227, 133, 122, 95, 83, 148, 165, 203, 236, 87, 253, 11, 60, 82, 36, 95, 199, 120, 54, 22, 228, 227, 110, 231, 204, 83, 94, 179, 154, 8, 200, 180, 89, 235, 200, 95, 12, 230, 120, 121, 157, 62, 255, 2, 66, 83, 185, 14, 132, 146, 124, 198, 128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 21, 1, 128, 3, 0, 128, 242, 238, 200, 199, 170, 3, 233, 7, 253, 11, 161, 204, 28, 109, 207, 110, 116, 240, 105, 181, 38, 64, 118, 111, 3, 222, 55, 184, 183, 81, 181, 61, 128, 223, 91, 79, 3, 86, 79, 50, 130, 149, 40, 193, 33, 8, 158, 35, 82, 137, 134, 195, 10, 60, 248, 193, 39, 236, 182, 225, 143, 110, 206, 62, 80, 169, 1, 159, 12, 182, 243, 110, 2, 122, 187, 32, 145, 207, 181, 17, 10, 181, 8, 127, 137, 0, 104, 95, 6, 21, 91, 60, 217, 168, 201, 229, 233, 162, 63, 213, 220, 19, 165, 237, 32, 0, 0, 0, 0, 0, 0, 0, 0, 104, 95, 8, 49, 108, 191, 143, 160, 218, 130, 42, 32, 172, 28, 85, 191, 27, 227, 32, 0, 0, 0, 0, 0, 0, 0, 0, 128, 254, 108, 203, 37, 75, 132, 240, 210, 32, 46, 181, 5, 189, 223, 159, 84, 203, 158, 189, 15, 178, 113, 144, 114, 233, 46, 229, 124, 29, 161, 216, 9, 0, 0, 20, 4, 2, 0, 193, 93, 60, 169, 1, 128, 60, 148, 0, 0, 0, 128, 209, 170, 86, 205, 21, 90, 223, 122, 141, 209, 101, 40, 181, 137, 244, 97, 74, 5, 224, 48, 181, 94, 70, 130, 207, 112, 41, 14, 110, 149, 215, 65, 128, 240, 132, 229, 221, 139, 121, 46, 20, 0, 224, 115, 50, 157, 45, 75, 164, 63, 23, 49, 233, 233, 131, 232, 2, 66, 220, 139, 78, 77, 181, 242, 55, 128, 44, 154, 180, 24, 159, 7, 154, 107, 131, 8, 30, 237, 18, 79, 129, 52, 159, 22, 194, 128, 244, 123, 56, 130, 106, 30, 246, 3, 188, 114, 20, 165, 0, 1, 1, 159, 6, 170, 57, 78, 234, 86, 48, 224, 124, 72, 174, 12, 149, 88, 206, 247, 48, 141, 80, 95, 14, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0, 76, 95, 6, 132, 160, 34, 163, 77, 216, 191, 162, 186, 175, 68, 241, 114, 183, 16, 4, 1, 0, 0, 0, 0, 200, 95, 10, 66, 243, 51, 35, 203, 92, 237, 59, 68, 221, 130, 95, 218, 159, 204, 128, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 92, 128, 144, 0, 0, 72, 94, 253, 108, 40, 131, 107, 154, 40, 82, 45, 201, 36, 17, 12, 244, 57, 4, 1, 244, 118, 71, 4, 181, 104, 210, 22, 103, 53, 106, 90, 5, 12, 17, 135, 70, 180, 222, 242, 92, 253, 166, 239, 58, 0, 0, 0, 0, 128, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 117, 5, 159, 9, 157, 136, 14, 198, 129, 121, 156, 12, 243, 14, 136, 134, 55, 29, 169, 93, 233, 128, 101, 208, 225, 22, 162, 251, 130, 181, 163, 187, 8, 132, 166, 45, 60, 251, 213, 213, 3, 226, 85, 58, 24, 65, 243, 191, 130, 90, 222, 225, 133, 241, 128, 34, 127, 74, 195, 148, 243, 74, 132, 78, 192, 170, 208, 186, 55, 251, 239, 89, 25, 232, 82, 112, 67, 156, 145, 124, 235, 61, 87, 91, 77, 106, 93, 128, 213, 158, 132, 188, 181, 221, 70, 181, 170, 183, 173, 183, 203, 120, 75, 162, 218, 140, 4, 72, 32, 18, 117, 181, 41, 207, 13, 156, 202, 27, 181, 21, 128, 102, 29, 206, 45, 146, 44, 68, 33, 71, 201, 247, 18, 252, 205, 95, 75, 73, 45, 230, 163, 93, 145, 21, 168, 65, 134, 208, 22, 166, 67, 169, 21, 128, 20, 41, 68, 33, 248, 131, 160, 35, 120, 253, 202, 223, 3, 7, 230, 107, 53, 173, 5, 150, 240, 60, 181, 162, 196, 35, 187, 33, 106, 255, 60, 234, 128, 251, 67, 138, 233, 52, 77, 203, 193, 235, 205, 49, 204, 239, 26, 76, 204, 84, 98, 110, 9, 31, 187, 130, 135, 164, 79, 177, 116, 176, 179, 236, 28, 128, 45, 154, 207, 199, 191, 39, 96, 0, 34, 235, 80, 226, 64, 15, 66, 12, 61, 175, 53, 2, 89, 247, 187, 117, 203, 104, 112, 155, 120, 116, 22, 241, 128, 194, 71, 245, 88, 116, 242, 64, 27, 124, 179, 52, 23, 124, 55, 79, 165, 90, 154, 49, 39, 255, 48, 104, 51, 43, 208, 167, 104, 53, 195, 188, 122, 128, 65, 26, 173, 102, 27, 123, 205, 192, 173, 69, 179, 56, 134, 219, 70, 86, 12, 190, 39, 49, 75, 212, 96, 204, 97, 162, 113, 54, 84, 28, 238, 56, 128, 148, 66, 156, 182, 70, 237, 254, 156, 14, 143, 119, 162, 11, 142, 235, 5, 74, 192, 20, 86, 18, 145, 23, 29, 244, 84, 98, 171, 231, 247, 100, 151, 168, 95, 9, 204, 233, 200, 136, 70, 155, 177, 160, 220, 234, 161, 41, 103, 46, 248, 96, 4, 88, 99, 117, 109, 117, 108, 117, 115, 45, 116, 101, 115, 116, 45, 112, 97, 114, 97, 99, 104, 97, 105, 110, 20, 128, 0, 132, 0, 0, 104, 129, 6, 40, 0, 0, 80, 92, 120, 116, 114, 105, 110, 115, 105, 99, 95, 105, 110, 100, 101, 120, 16, 0, 0, 0, 0, 148, 192, 64, 0, 0, 128, 143, 175, 68, 33, 248, 1, 167, 234, 82, 183, 240, 207, 133, 169, 98, 72, 214, 201, 140, 168, 141, 144, 225, 236, 222, 233, 162, 42, 80, 86, 251, 54, 160, 158, 20, 103, 160, 150, 188, 215, 26, 91, 106, 12, 129, 85, 226, 8, 16, 24, 0, 0, 80, 95, 14, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0, 92, 128, 1, 128, 72, 94, 140, 233, 97, 93, 224, 119, 90, 130, 248, 169, 77, 195, 210, 133, 161, 4, 1, 0, 132, 94, 46, 223, 59, 223, 56, 29, 235, 227, 49, 171, 116, 70, 173, 223, 220, 64, 0, 0, 100, 167, 179, 182, 224, 13, 0, 0, 0, 0, 0, 0, 0, 0, 148, 127, 0, 5, 50, 61, 247, 204, 71, 21, 11, 57, 48, 226, 102, 107, 10, 163, 19, 78, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 2, 0, 148, 127, 0, 0, 195, 101, 195, 207, 89, 214, 113, 235, 114, 218, 14, 122, 65, 19, 196, 78, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0},
	}
	ctrl := gomock.NewController(t)
	mockInstance := NewMockRuntimeInstance(ctrl)

	// doing all this because adder internally compares postState with bd.State in it's validate_block
	encoded_state, err := scale.Marshal(uint64(1))
	require.NoError(t, err)
	postState, err := common.Keccak256(encoded_state)
	require.NoError(t, err)

	hd, err := scale.Marshal(HeadData{
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

	mockPoVRequestor := NewMockPoVRequestor(ctrl)

	// fileContent, err := ioutil.ReadFile("test-validation-code.txt")
	// require.NoError(t, err)

	// validationCodeBytes, err := common.HexToBytes(strings.TrimSpace(string(fileContent)))
	// require.NoError(t, err)

	// TODO: put this in the
	runtimeFilePath := "./test_parachain_adder.wasm"
	validationCodeBytes, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	validationCode := parachaintypes.ValidationCode(validationCodeBytes)
	// validationCode := ValidationCode([]byte{1, 2, 3})

	mockInstance.EXPECT().ParachainHostPersistedValidationData(uint32(1000), gomock.Any()).Return(&persistedValidationData, nil)
	mockInstance.EXPECT().ParachainHostValidationCode(uint32(1000), gomock.Any()).Return(&validationCode, nil)

	mockPoVRequestor.EXPECT().RequestPoV(common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274")).Return(pov)
	// get PersistedValidationData and ValidationCode from polkadot test
	// candidateCommitment, persistedValidationData, err := ValidateFromChainState(mockInstance, candidateReceipt)
	_, _, err = ValidateFromChainState(mockInstance, mockPoVRequestor, candidateReceipt)
	require.NoError(t, err)

}

// encoded validation params [32, 28, 1, 3, 4, 5, 6, 7, 9, 24, 20, 1, 2, 3, 4, 5, 1, 0, 0, 0, 44, 253, 215, 195, 247, 110, 203, 34, 19, 93, 216, 224, 103, 29, 93, 184, 28, 116, 6, 85, 208, 180, 219, 158, 193, 113, 135, 4, 74, 83, 152, 228]
// sample validation code hash 0x9985e134020e8a1e2e211afcd5ac9ec6a2fd21dfb5e1c39b3b670f4415e90406

/*
validation_params.block_data: BlockData([20, 1, 2, 3, 4, 5])
validation_params.block_data.encode(): [24, 20, 1, 2, 3, 4, 5]
encoded: [32, 28, 1, 3, 4, 5, 6, 7, 9, 24, 20, 1, 2, 3, 4, 5, 1, 0, 0, 0, 49, 154, 0, 215, 46, 84, 192, 43, 175, 163, 128, 246, 208, 227, 135, 217, 129, 56, 180, 102, 170, 194, 185, 83, 102, 56, 35, 238, 186, 116, 193, 148]

validation_params encoded  []byte{81, 201, 53, 192, 127, 19, 238, 98, 79, 205, 228, 10, 179, 251, 158, 207, 52, 216, 172, 73, 103, 67, 28, 78, 89, 243, 72, 167, 68, 55, 171, 48, 4, 243, 127, 239, 111, 183, 196, 107, 211, 117, 189, 178, 167, 60, 244, 232, 43, 127, 66, 173, 161, 133, 81, 249, 39, 150, 99, 231, 165, 239, 45, 157, 254, 43, 147, 43, 189, 160, 51, 247, 131, 81, 116, 140, 187, 94, 154, 130, 29, 160, 114, 135, 104, 45, 241, 56, 184, 121, 36, 58, 92, 104, 173, 26, 87, 0, 8, 133, 7, 4, 1, 0, 137, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 159, 218, 60, 92, 146, 103, 222, 40, 150, 224, 30, 70, 135, 2, 69, 126, 139, 7, 125, 213, 45, 212, 111, 38, 78, 224, 145, 14, 178, 25, 86, 105, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 1, 0, 0, 0, 219, 170, 117, 174, 158, 87, 148, 78, 73, 196, 180, 170, 79, 8, 133, 18, 53, 7, 127, 156, 20, 90, 144, 228, 136, 213, 65, 142, 34, 241, 63, 144, 0, 0, 0, 0, 20, 144, 0, 0, 32, 0, 0, 0, 16, 0, 8, 0, 0, 0, 0, 4, 0, 0, 0, 1, 0, 0, 5, 0, 0, 0, 5, 0, 0, 0, 6, 0, 0, 0, 6, 0, 0, 0, 9, 1, 63, 32, 6, 222, 61, 138, 84, 210, 126, 68, 169, 213, 206, 24, 150, 24, 242, 45, 180, 180, 157, 149, 50, 13, 144, 33, 153, 76, 133, 15, 37, 184, 227, 133, 122, 95, 83, 148, 165, 203, 236, 87, 253, 11, 60, 82, 36, 95, 199, 120, 54, 22, 228, 227, 110, 231, 204, 83, 94, 179, 154, 8, 200, 180, 89, 235, 200, 95, 12, 230, 120, 121, 157, 62, 255, 2, 66, 83, 185, 14, 132, 146, 124, 198, 128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 21, 1, 128, 3, 0, 128, 242, 238, 200, 199, 170, 3, 233, 7, 253, 11, 161, 204, 28, 109, 207, 110, 116, 240, 105, 181, 38, 64, 118, 111, 3, 222, 55, 184, 183, 81, 181, 61, 128, 223, 91, 79, 3, 86, 79, 50, 130, 149, 40, 193, 33, 8, 158, 35, 82, 137, 134, 195, 10, 60, 248, 193, 39, 236, 182, 225, 143, 110, 206, 62, 80, 169, 1, 159, 12, 182, 243, 110, 2, 122, 187, 32, 145, 207, 181, 17, 10, 181, 8, 127, 137, 0, 104, 95, 6, 21, 91, 60, 217, 168, 201, 229, 233, 162, 63, 213, 220, 19, 165, 237, 32, 0, 0, 0, 0, 0, 0, 0, 0, 104, 95, 8, 49, 108, 191, 143, 160, 218, 130, 42, 32, 172, 28, 85, 191, 27, 227, 32, 0, 0, 0, 0, 0, 0, 0, 0, 128, 254, 108, 203, 37, 75, 132, 240, 210, 32, 46, 181, 5, 189, 223, 159, 84, 203, 158, 189, 15, 178, 113, 144, 114, 233, 46, 229, 124, 29, 161, 216, 9, 0, 0, 20, 4, 2, 0, 193, 93, 60, 169, 1, 128, 60, 148, 0, 0, 0, 128, 209, 170, 86, 205, 21, 90, 223, 122, 141, 209, 101, 40, 181, 137, 244, 97, 74, 5, 224, 48, 181, 94, 70, 130, 207, 112, 41, 14, 110, 149, 215, 65, 128, 240, 132, 229, 221, 139, 121, 46, 20, 0, 224, 115, 50, 157, 45, 75, 164, 63, 23, 49, 233, 233, 131, 232, 2, 66, 220, 139, 78, 77, 181, 242, 55, 128, 44, 154, 180, 24, 159, 7, 154, 107, 131, 8, 30, 237, 18, 79, 129, 52, 159, 22, 194, 128, 244, 123, 56, 130, 106, 30, 246, 3, 188, 114, 20, 165, 0, 1, 1, 159, 6, 170, 57, 78, 234, 86, 48, 224, 124, 72, 174, 12, 149, 88, 206, 247, 48, 141, 80, 95, 14, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0, 76, 95, 6, 132, 160, 34, 163, 77, 216, 191, 162, 186, 175, 68, 241, 114, 183, 16, 4, 1, 0, 0, 0, 0, 200, 95, 10, 66, 243, 51, 35, 203, 92, 237, 59, 68, 221, 130, 95, 218, 159, 204, 128, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 92, 128, 144, 0, 0, 72, 94, 253, 108, 40, 131, 107, 154, 40, 82, 45, 201, 36, 17, 12, 244, 57, 4, 1, 244, 118, 71, 4, 181, 104, 210, 22, 103, 53, 106, 90, 5, 12, 17, 135, 70, 180, 222, 242, 92, 253, 166, 239, 58, 0, 0, 0, 0, 128, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 117, 5, 159, 9, 157, 136, 14, 198, 129, 121, 156, 12, 243, 14, 136, 134, 55, 29, 169, 93, 233, 128, 101, 208, 225, 22, 162, 251, 130, 181, 163, 187, 8, 132, 166, 45, 60, 251, 213, 213, 3, 226, 85, 58, 24, 65, 243, 191, 130, 90, 222, 225, 133, 241, 128, 34, 127, 74, 195, 148, 243, 74, 132, 78, 192, 170, 208, 186, 55, 251, 239, 89, 25, 232, 82, 112, 67, 156, 145, 124, 235, 61, 87, 91, 77, 106, 93, 128, 213, 158, 132, 188, 181, 221, 70, 181, 170, 183, 173, 183, 203, 120, 75, 162, 218, 140, 4, 72, 32, 18, 117, 181, 41, 207, 13, 156, 202, 27, 181, 21, 128, 102, 29, 206, 45, 146, 44, 68, 33, 71, 201, 247, 18, 252, 205, 95, 75, 73, 45, 230, 163, 93, 145, 21, 168, 65, 134, 208, 22, 166, 67, 169, 21, 128, 20, 41, 68, 33, 248, 131, 160, 35, 120, 253, 202, 223, 3, 7, 230, 107, 53, 173, 5, 150, 240, 60, 181, 162, 196, 35, 187, 33, 106, 255, 60, 234, 128, 251, 67, 138, 233, 52, 77, 203, 193, 235, 205, 49, 204, 239, 26, 76, 204, 84, 98, 110, 9, 31, 187, 130, 135, 164, 79, 177, 116, 176, 179, 236, 28, 128, 45, 154, 207, 199, 191, 39, 96, 0, 34, 235, 80, 226, 64, 15, 66, 12, 61, 175, 53, 2, 89, 247, 187, 117, 203, 104, 112, 155, 120, 116, 22, 241, 128, 194, 71, 245, 88, 116, 242, 64, 27, 124, 179, 52, 23, 124, 55, 79, 165, 90, 154, 49, 39, 255, 48, 104, 51, 43, 208, 167, 104, 53, 195, 188, 122, 128, 65, 26, 173, 102, 27, 123, 205, 192, 173, 69, 179, 56, 134, 219, 70, 86, 12, 190, 39, 49, 75, 212, 96, 204, 97, 162, 113, 54, 84, 28, 238, 56, 128, 148, 66, 156, 182, 70, 237, 254, 156, 14, 143, 119, 162, 11, 142, 235, 5, 74, 192, 20, 86, 18, 145, 23, 29, 244, 84, 98, 171, 231, 247, 100, 151, 168, 95, 9, 204, 233, 200, 136, 70, 155, 177, 160, 220, 234, 161, 41, 103, 46, 248, 96, 4, 88, 99, 117, 109, 117, 108, 117, 115, 45, 116, 101, 115, 116, 45, 112, 97, 114, 97, 99, 104, 97, 105, 110, 20, 128, 0, 132, 0, 0, 104, 129, 6, 40, 0, 0, 80, 92, 120, 116, 114, 105, 110, 115, 105, 99, 95, 105, 110, 100, 101, 120, 16, 0, 0, 0, 0, 148, 192, 64, 0, 0, 128, 143, 175, 68, 33, 248, 1, 167, 234, 82, 183, 240, 207, 133, 169, 98, 72, 214, 201, 140, 168, 141, 144, 225, 236, 222, 233, 162, 42, 80, 86, 251, 54, 160, 158, 20, 103, 160, 150, 188, 215, 26, 91, 106, 12, 129, 85, 226, 8, 16, 24, 0, 0, 80, 95, 14, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0, 92, 128, 1, 128, 72, 94, 140, 233, 97, 93, 224, 119, 90, 130, 248, 169, 77, 195, 210, 133, 161, 4, 1, 0, 132, 94, 46, 223, 59, 223, 56, 29, 235, 227, 49, 171, 116, 70, 173, 223, 220, 64, 0, 0, 100, 167, 179, 182, 224, 13, 0, 0, 0, 0, 0, 0, 0, 0, 148, 127, 0, 5, 50, 61, 247, 204, 71, 21, 11, 57, 48, 226, 102, 107, 10, 163, 19, 78, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 2, 0, 148, 127, 0, 0, 195, 101, 195, 207, 89, 214, 113, 235, 114, 218, 14, 122, 65, 19, 196, 78, 123, 144, 18, 9, 107, 65, 196, 235, 58, 175, 148, 127, 110, 164, 41, 8, 0, 0}

*/
