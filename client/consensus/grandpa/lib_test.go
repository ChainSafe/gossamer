// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// TODO fix test
//func Test_checkMessageSignature(t *testing.T) {
//	kp, err := ed25519.GenerateKeypair()
//	require.NoError(t, err)
//
//	// TODO Empty value until message is exported
//	message := finalityGrandpa.Message[string, uint]{}
//
//	msg := messageData[string, uint]{
//		1,
//		2,
//		message,
//	}
//
//	encMsg, err := scale.Marshal(msg)
//	require.NoError(t, err)
//
//	sig, err := kp.Sign(encMsg)
//	require.NoError(t, err)
//
//	valid, err := checkMessageSignature[string, uint](message, kp.Public().(*ed25519.PublicKey), sig, 1, 2)
//	require.NoError(t, err)
//	require.True(t, valid)
//}
