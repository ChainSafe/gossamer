package hexcodec

// Encode assumes its input is an array of nibbles (4bits), and produces Hex-Encoded output.
// HexEncoded: For PK = (k_1,...,k_n), Enc_hex(PK) :=
// (0, k_1 + k_2 * 16,...) for even length
// (k_1, k_2 + k_3 * 16,...) for odd length
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

// combineNibbles concatenates two nibble to make a byte.
// Assumes nibbles are the lower 4 bits of each of the inputs
func combineNibbles(ms byte, ls byte) byte {
	return byte(ms<<4 | (ls & 0xF))
}
