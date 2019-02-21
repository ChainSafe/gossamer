package trie

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

type node interface {
}

type (
	branch struct {
		Children [17]node //array of 17 elements, 16 for each nibble, and 17-th for the value
	}
	extension struct { //key-value pair
		Key []byte
		Val node
		terminator bool
	}
	leaf []byte
)

