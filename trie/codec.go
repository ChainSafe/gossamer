package trie

// keyEncodeByte swaps the two nibbles of a byte to result in 'LE'
func keyEncodeByte(b byte) byte {
	b1 := (uint(b) & 240) >> 4
	b2 := (uint(b) & 15) << 4

	return byte(b1 | b2)
}

// KeyEncode encodes the key by placing in in little endian nibble format
func KeyEncode(k []byte) []byte {
	result := make([]byte, len(k))
	// Encode each byte
	for i, b := range k {
		result[i] = keyEncodeByte(b)
	}
	return result
}

// keyToHex turns bytes into nibbles
func keyToHex(in []byte) []byte {
	l := len(in) * 2
	res := make([]byte, l)
	for i, b := range in {
		res[2*i] = b / 16
		res[2*i+1] = b % 16
	}

	return res
}

// hexToKey performs the opposite of keyToHex; turns nibbles back into bytes
// removes last byte if length of input is odd (set to 16 if using keyToHex)
func hexToKey(in []byte) []byte {
	l := len(in) / 2
	res := make([]byte, l)
	for i := 0; i < len(in)-1; i = i + 2 {
		res[i/2] = in[i+1] | in[i]<<4
	}
	return res
}

// bigKeySize returns the node type's BigKeySize
// BigKeySize is 125 if node is extension, 126 if node is leaf
// func bigKeySize(n node) int {
// 	switch n.(type) {
// 	case *leaf:
// 		return 126
// 	default:
// 		return -1
// 	}
// }

// // getPrefix returns the node type's prefix, used for encoding the node
// func getPrefix(n node) (prefix byte) {
// 	switch n := n.(type) {
// 	case *leaf:
// 		return 1
// 	case *branch:
// 		if n.value == nil {
// 			// branch without value
// 			return 254
// 		}
// 		// branch with value
// 		return 255
// 	default:
// 		return 0
// 	}
// }

// func uint16ToBytes(in uint16) (out []byte) {
// 	out = make([]byte, 2)
// 	out[0] = byte(in & 0x00ff)
// 	out[1] = byte(in >> 8 & 0x00ff)
// 	return out
// }
