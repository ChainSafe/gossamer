package erasure

// #cgo CFLAGS: -I.
// #cgo LDFLAGS: -L. -lerasure
// #include "./rustlib.h"
import (
	"C"
)
import (
	"fmt"
	"unsafe"
)

// Obtain erasure-coded chunks, one for each validator.
//
// Works only up to 65536 validators, and `n_validators` must be non-zero.
//
// Accepts number of validators and scale encoded data.
func ObtainChunks(nValidators uint, data []byte) ([][]byte, error) {
	var cFlattenedChunks *C.uchar
	var cFlattenedChunksLen C.size_t

	cnValidators := C.size_t(nValidators)
	cData := (*C.uchar)(unsafe.Pointer(&data[0]))
	cLen := C.size_t(len(data))

	cErr := C.obtain_chunks(cnValidators, cData, cLen, &cFlattenedChunks, &cFlattenedChunksLen)
	var errStr string = C.GoString(cErr)
	C.free(unsafe.Pointer(cErr))

	if len(errStr) > 0 {
		return nil, fmt.Errorf(errStr)
	}

	resData := C.GoBytes(unsafe.Pointer(cFlattenedChunks), C.int(cFlattenedChunksLen))
	C.free(unsafe.Pointer(cFlattenedChunks))

	chunkSize := uint(len(resData)) / nValidators
	chunks := make([][]byte, nValidators)

	for i := uint(0); i < nValidators; i++ {
		start := i * chunkSize
		end := start + chunkSize
		chunks[i] = resData[start:end]
	}

	return chunks, nil
}

// Reconstruct decodable data from a set of chunks.
//
// Provide an iterator containing chunk data and the corresponding index.
// The indices of the present chunks must be indicated. If too few chunks
// are provided, recovery is not possible.
//
// Works only up to 65536 validators, and `n_validators` must be non-zero
func Reconstruct(nValidators uint, chunks [][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("Chunks can't be zero sized")
	}

	var cReconstructedData *C.uchar
	var cReconstructedDataLen C.size_t
	var flattenedChunks []byte

	for _, chunk := range chunks {
		flattenedChunks = append(flattenedChunks, chunk...)
	}

	cChunkSize := C.size_t(len(chunks[0]))
	cFlattenedChunks := (*C.uchar)(unsafe.Pointer(&flattenedChunks[0]))
	cFlattenedChunksLen := C.size_t(len(flattenedChunks))

	cErr := C.reconstruct(C.size_t(nValidators), cFlattenedChunks, cFlattenedChunksLen, cChunkSize, &cReconstructedData, &cReconstructedDataLen)
	errStr := C.GoString(cErr)
	C.free(unsafe.Pointer(cErr))

	if len(errStr) > 0 {
		return nil, fmt.Errorf(errStr)
	}

	res := C.GoBytes(unsafe.Pointer(cReconstructedData), C.int(cReconstructedDataLen))
	C.free(unsafe.Pointer(cReconstructedData))

	return res, nil
}

func main() {
}
