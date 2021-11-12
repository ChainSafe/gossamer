// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

// keyToNibbles turns bytes into nibbles
// does not rearrange the nibbles; assumes they are already ordered in LE
func keyToNibbles(in []byte) []byte {
	if len(in) == 0 {
		return []byte{}
	} else if len(in) == 1 && in[0] == 0 {
		return []byte{0, 0}
	}

	l := len(in) * 2
	res := make([]byte, l)
	for i, b := range in {
		res[2*i] = b / 16
		res[2*i+1] = b % 16
	}

	return res
}

// nibblesToKey turns a slice of nibbles w/ length k into a big endian byte array
// if the length of the input is odd, the result is [ in[1] in[0] | ... | 0000 in[k-1] ]
// otherwise, res = [ in[1] in[0] | ... | in[k-1] in[k-2] ]
func nibblesToKey(in []byte) (res []byte) {
	if len(in)%2 == 0 {
		res = make([]byte, len(in)/2)
		for i := 0; i < len(in); i += 2 {
			res[i/2] = (in[i] & 0xf) | (in[i+1] << 4 & 0xf0)
		}
	} else {
		res = make([]byte, len(in)/2+1)
		for i := 0; i < len(in); i += 2 {
			if i < len(in)-1 {
				res[i/2] = (in[i] & 0xf) | (in[i+1] << 4 & 0xf0)
			} else {
				res[i/2] = (in[i] & 0xf)
			}
		}
	}

	return res
}

// nibblesToKey turns a slice of nibbles w/ length k into a little endian byte array
// assumes nibbles are already LE, does not rearrange nibbles
// if the length of the input is odd, the result is [ 0000 in[0] | in[1] in[2] | ... | in[k-2] in[k-1] ]
// otherwise, res = [ in[0] in[1] | ... | in[k-2] in[k-1] ]
func nibblesToKeyLE(in []byte) (res []byte) {
	if len(in)%2 == 0 {
		res = make([]byte, len(in)/2)
		for i := 0; i < len(in); i += 2 {
			res[i/2] = (in[i] << 4 & 0xf0) | (in[i+1] & 0xf)
		}
	} else {
		res = make([]byte, len(in)/2+1)
		res[0] = in[0]
		for i := 2; i < len(in); i += 2 {
			res[i/2] = (in[i-1] << 4 & 0xf0) | (in[i] & 0xf)
		}
	}

	return res
}
