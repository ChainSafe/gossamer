package hexcodec

// Encode assumes its input is an array of nibbles (4bits).
func Encode(in []byte) (res []byte) {
	resI := 1
	if len(in)%2 == 1 { // Odd length
		res := make([]byte, (len(in)/2)+1)
		res[0] = in[0]

		for i := 1; i < len(in)-1; i += 2 {
			res[resI] = combineNibbles(in[i], in[i+1])
			resI++
		}
	} else { // Even length
		res := make([]byte, (len(in)/2)+1)
		res[0] = 0x0

		for i := 0; i < len(in)-1; i += 2 {
			res[resI] = combineNibbles(in[i], in[i+1])
			resI++
		}
	}
	return res
}

// combineNibbles concatenates the upper, most significant nibble with the lower, least significant nibble.
// Assumes nibles are the lower 4 bits of each of the inputs
func combineNibbles(ms byte, ls byte) byte {
	return byte(ms<<4 | (ls & 0xF))
}
