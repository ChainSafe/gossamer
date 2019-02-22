package trie

import "fmt"

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
