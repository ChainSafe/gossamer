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
func keyToNibbles(in []byte) []byte {
	if len(in) == 0 {
		return []byte{}
	}

	l := len(in) * 2
	res := make([]byte, l)
	for i, b := range in {
		res[2*i] = b / 16
		res[2*i+1] = b % 16
	}

	if res[l-1] == 0 {
		res = res[:l-1]
	}

	return res
}

func nibblesToKey(in []byte) (res []byte) {
	if len(in) % 2 == 0 {
		res = make([]byte, len(in)/2)
		for i := 0; i < len(in); i += 2 {
			res[i/2] = (in[i] << 4 & 0xf0) | (in[i+1] & 0xf)
		}
	} else {
		res = make([]byte, len(in)/2 + 1)
		for i := 0; i < len(in); i += 2 {
			if i < len(in) - 1 {
				res[i/2] = (in[i] << 4 & 0xf0) | (in[i+1] & 0xf)
			} else {
				res[i/2] = (in[i] << 4 & 0xf0)
			}
		}
	}

	return res
}