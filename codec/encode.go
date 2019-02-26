package codec

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"reflect"
	"fmt"
)

// Encoder is a wrapping around io.Writer
type Encoder struct {
	writer io.Writer
}

// Encode is the top-level function which performs SCALE encoding of b which may be of type []byte, int16, int32, int64,
// or bool
func (se *Encoder) Encode(b interface{}) (n int, err error) {
	switch v := b.(type) {
	case []byte:
		n, err = se.encodeByteArray(v)
	case *big.Int:
		n, err = se.encodeBigInteger(v)
	case int16:
		n, err = se.encodeInteger(int(v))
	case int32:
		n, err = se.encodeInteger(int(v))
	case int64:
		n, err = se.encodeInteger(int(v))
	case string:
		n, err = se.encodeByteArray([]byte(v))
	case bool:
		n, err = se.encodeBool(v)
	case interface{}:
		t := reflect.TypeOf(b).Kind()
		switch t {
		case reflect.Struct:
			n, err = se.encodeTuple(v)
		case reflect.Slice, reflect.Array:
			n, err = se.encodeVector(v)
		default:
			return 0, errors.New("unsupported type")
		}
	default:
		return 0, errors.New("unsupported type")
	}

	return n, err
}

// encodeByteArray performs the following:
// b -> [encodeInteger(len(b)) b]
// it writes to the buffer a byte array where the first byte is the length of b encoded with SCALE, followed by the
// byte array b itself
func (se *Encoder) encodeByteArray(b []byte) (bytesEncoded int, err error) {
	var n int
	n, err = se.encodeInteger(len(b))
	if err != nil {
		return 0, err
	}

	bytesEncoded = bytesEncoded + n
	n, err = se.writer.Write(b)
	return bytesEncoded + n, err
}

// encodeInteger performs the following on integer i:
// i  -> i^0...i^n where n is the length in bits of i
// note that the bit representation of i is in little endian; ie i^0 is the least significant bit of i,
// and i^n is the most significant bit
// if n < 2^6 write [00 i^2...i^8 ] [ 8 bits = 1 byte encoded ]
// if 2^6 <= n < 2^14 write [01 i^2...i^16] [ 16 bits = 2 byte encoded ]
// if 2^14 <= n < 2^30 write [10 i^2...i^32] [ 32 bits = 4 byte encoded ]
// if n >= 2^30 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (se *Encoder) encodeInteger(i int) (bytesEncoded int, err error) {
	if i < 1<<6 {
		err = binary.Write(se.writer, binary.LittleEndian, byte(i)<<2)
		return 1, err
	} else if i < 1<<14 {
		err = binary.Write(se.writer, binary.LittleEndian, uint16(i<<2)+1)
		return 2, err
	} else if i < 1<<30 {
		err = binary.Write(se.writer, binary.LittleEndian, uint32(i<<2)+2)
		return 4, err
	}

	o := make([]byte, 8)
	m := i
	var numBytes int

	// calculate the number of bytes needed to store i
	// the most significant byte cannot be zero
	// each iteration, shift by 1 byte until the number is zero
	// then break and save the numBytes needed
	for numBytes = 0; numBytes < 256 && m != 0; numBytes++ {
		m = m >> 8
	}

	topSixBits := uint8(numBytes - 4)
	lengthByte := topSixBits<<2 + 3

	err = binary.Write(se.writer, binary.LittleEndian, lengthByte)
	bytesEncoded++
	if err == nil {
		binary.LittleEndian.PutUint64(o, uint64(i))
		err = binary.Write(se.writer, binary.LittleEndian, o[0:numBytes])
		bytesEncoded += numBytes
	}

	return bytesEncoded, err
}

// encodeBigInteger performs the same encoding as encodeInteger, except on a big.Int.
// if 2^30 <= n < 2^536 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (se *Encoder) encodeBigInteger(i *big.Int) (bytesEncoded int, err error) {
	if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 6)) < 0 { // if i < 1<<6
		err = binary.Write(se.writer, binary.LittleEndian, uint8(i.Int64()<<2))
		return 1, err
	} else if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 14)) < 0 { // if i < 1<<14
		err = binary.Write(se.writer, binary.LittleEndian, uint16(i.Int64()<<2)+1)
		return 2, err
	} else if i.Cmp(new(big.Int).Lsh(big.NewInt(1), 30)) < 0 { //if i < 1<<30
		err = binary.Write(se.writer, binary.LittleEndian, uint32(i.Int64()<<2)+2)
		return 4, err
	}

	numBytes := len(i.Bytes())
	topSixBits := uint8(numBytes - 4)
	lengthByte := topSixBits<<2 + 3

	// write byte which encodes mode and length
	err = binary.Write(se.writer, binary.LittleEndian, lengthByte)
	if err == nil {
		// write integer itself
		err = binary.Write(se.writer, binary.LittleEndian, i.Bytes())
	}

	return numBytes + 1, err
}

// encodeBool performs the following:
// l = true -> write [1]
// l = false -> write [0]
func (se *Encoder) encodeBool(l bool) (bytesEncoded int, err error) {
	if l {
		bytesEncoded, err = se.writer.Write([]byte{0x01})
		return bytesEncoded, err
	}
	bytesEncoded, err = se.writer.Write([]byte{0x00})
	return bytesEncoded, err
}

// encodeTuple reads the number of fields in the struct and their types and writes to the buffer each of the struct fields
// encoded as their respective types
func (se *Encoder) encodeTuple(t interface{}) (bytesEncoded int, err error) {
	v := reflect.ValueOf(t)

	values := make([]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
	}

	for _, item := range values {
		n, err := se.Encode(item)
		if err != nil {
			return bytesEncoded, err
		}

		bytesEncoded += n
	}

	return bytesEncoded, nil
}

func (se *Encoder) encodeIntegerElements(arr []int) (bytesEncoded int, err error) {
	var n int
	n, err = se.encodeInteger(len(arr))
	bytesEncoded += n

	for _, elem := range arr {
		n, err = se.encodeInteger(int(elem))
		bytesEncoded += n
	}

	return bytesEncoded, err
}
// encodeVector encodes an interface where the underlying type is an array or slice
// it writes the encoded length of the vector to the Encoder, then encodes and writes each value in the vector
func (se *Encoder) encodeVector(t interface{}) (bytesEncoded int, err error) {
	fmt.Println(reflect.TypeOf(t))

	var n int
	switch arr := t.(type) {
	case []int:
		n, err = se.encodeIntegerElements(arr)
		bytesEncoded += n
	case []*big.Int:
		n, err = se.encodeInteger(len(arr))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeBigInteger(elem)
			bytesEncoded += n
		}		
	case []bool:
		n, err = se.encodeInteger(len(arr))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeBool(elem)
			bytesEncoded += n
		}
	case [][]byte:
		n, err = se.encodeInteger(len(arr))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeByteArray(elem)
			bytesEncoded += n
		}
	case [][]int:
		n, err = se.encodeInteger(len(arr))
		bytesEncoded += n

		for _, elem := range arr {
			n, err = se.encodeVector(elem)
			bytesEncoded += n
		}
	// TODO: case struct ?
	}

	return bytesEncoded, err
}