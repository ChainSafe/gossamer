package trie

import "fmt"

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

func keybytesReturnNib(str []byte) []byte {
	l := len(str)*2 + 1
	fmt.Println(l, str)
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

