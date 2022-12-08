// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEquivocationProof(t *testing.T) {
	// To get these bytes run https://github.com/paritytech/substrate/blob/17c07af0b953b84dbe89341294e98e586f9b4591/frame/babe/src/tests.rs#L932
	exp := []byte{
		222, 241, 46, 66, 243, 228, 135, 233, 177, 64, 149, 170, 141, 92, 193, 106, 51, 73, 31, 27, 80, 218, 220, 248, 129, 29, 20, 128, 243, 250, 134, 39, 11, 0, 0, 0, 0, 0, 0, 0, 67, 253, 147, 84, 100, 171, 70, 100, 23, 162, 211, 181, 27, 117, 11, 48, 71, 172, 201, 71, 8, 170, 142, 105, 187, 1, 209, 158, 123, 168, 65, 244, 40, 206, 230, 49, 228, 215, 82, 164, 222, 129, 48, 67, 27, 99, 36, 109, 105, 93, 204, 135, 175, 136, 19, 22, 37, 27, 198, 211, 86, 81, 249, 80, 138, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 12, 6, 66, 65, 66, 69, 52, 2, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 4, 66, 65, 66, 69, 105, 2, 1, 12, 222, 241, 46, 66, 243, 228, 135, 233, 177, 64, 149, 170, 141, 92, 193, 106, 51, 73, 31, 27, 80, 218, 220, 248, 129, 29, 20, 128, 243, 250, 134, 39, 1, 0, 0, 0, 0, 0, 0, 0, 58, 61, 69, 220, 85, 181, 123, 245, 66, 244, 198, 255, 65, 175, 8, 14, 198, 117, 49, 127, 78, 213, 10, 225, 210, 113, 59, 249, 248, 146, 105, 45, 1, 0, 0, 0, 0, 0, 0, 0, 84, 199, 28, 35, 87, 115, 184, 33, 21, 240, 116, 66, 82, 54, 156, 19, 65, 79, 208, 232, 186, 211, 232, 254, 255, 70, 44, 106, 75, 181, 138, 15, 1, 0, 0, 0, 0, 0, 0, 0, 198, 233, 208, 44, 227, 141, 231, 178, 85, 56, 47, 128, 74, 100, 249, 188, 116, 170, 213, 89, 127, 81, 253, 230, 187, 83, 192, 184, 167, 108, 34, 186, 5, 66, 65, 66, 69, 1, 1, 88, 129, 117, 10, 97, 243, 99, 3, 71, 0, 51, 215, 169, 196, 213, 101, 78, 228, 209, 25, 131, 186, 115, 0, 140, 190, 74, 248, 224, 54, 30, 98, 177, 230, 123, 88, 35, 106, 66, 88, 241, 124, 238, 213, 61, 17, 226, 4, 82, 130, 56, 164, 18, 234, 182, 206, 52, 118, 233, 211, 235, 66, 193, 129, 67, 253, 147, 84, 100, 171, 70, 100, 23, 162, 211, 181, 27, 117, 11, 48, 71, 172, 201, 71, 8, 170, 142, 105, 187, 1, 209, 158, 123, 168, 65, 244, 40, 206, 230, 49, 228, 215, 82, 164, 222, 129, 48, 67, 27, 99, 36, 109, 105, 93, 204, 135, 175, 136, 19, 22, 37, 27, 198, 211, 86, 81, 249, 80, 138, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 12, 6, 66, 65, 66, 69, 52, 2, 0, 0, 0, 0, 10, 0, 0, 0, 0, 0, 0, 0, 4, 66, 65, 66, 69, 105, 2, 1, 12, 222, 241, 46, 66, 243, 228, 135, 233, 177, 64, 149, 170, 141, 92, 193, 106, 51, 73, 31, 27, 80, 218, 220, 248, 129, 29, 20, 128, 243, 250, 134, 39, 1, 0, 0, 0, 0, 0, 0, 0, 58, 61, 69, 220, 85, 181, 123, 245, 66, 244, 198, 255, 65, 175, 8, 14, 198, 117, 49, 127, 78, 213, 10, 225, 210, 113, 59, 249, 248, 146, 105, 45, 1, 0, 0, 0, 0, 0, 0, 0, 84, 199, 28, 35, 87, 115, 184, 33, 21, 240, 116, 66, 82, 54, 156, 19, 65, 79, 208, 232, 186, 211, 232, 254, 255, 70, 44, 106, 75, 181, 138, 15, 1, 0, 0, 0, 0, 0, 0, 0, 198, 233, 208, 44, 227, 141, 231, 178, 85, 56, 47, 128, 74, 100, 249, 188, 116, 170, 213, 89, 127, 81, 253, 230, 187, 83, 192, 184, 167, 108, 34, 186, 5, 66, 65, 66, 69, 1, 1, 228, 223, 106, 3, 77, 80, 87, 177, 234, 206, 45, 212, 145, 143, 35, 87, 197, 171, 4, 19, 97, 85, 150, 235, 238, 81, 41, 251, 15, 207, 20, 106, 8, 124, 139, 59, 101, 213, 95, 118, 235, 249, 26, 119, 80, 78, 51, 75, 233, 182, 163, 108, 184, 54, 173, 245, 140, 253, 23, 86, 177, 73, 182, 137,
	}
	dec := BabeEquivocationProof{
		FirstHeader:  *NewEmptyHeader(),
		SecondHeader: *NewEmptyHeader(),
	}

	err := scale.Unmarshal(exp, &dec)
	require.NoError(t, err)

	enc, err := scale.Marshal(dec)
	require.NoError(t, err)
	require.Equal(t, exp, enc)
}
