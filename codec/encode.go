package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"reflect"
)

// Encoder is a wrapping around io.Writer
type Encoder struct {
	writer io.Writer
}

// Encode is the top-level function which performs SCALE encoding of b which may be of type []byte, int16, int32, int64,
// or bool
func Encode(b interface{}) (encodedItem []byte, err error) {
	var buffer = bytes.Buffer{}
	se := Encoder{&buffer}

	switch v := b.(type) {
	case []byte:
		_, err = se.encodeByteArray(v)
	case *big.Int:
		_, err = se.encodeBigInteger(v)
	case int16:
		_, err = se.encodeInteger(int(v))
	case int32:
		_, err = se.encodeInteger(int(v))
	case int64:
		_, err = se.encodeInteger(int(v))
	// case string:
	// 	return encodeByteArray([]byte(v))
	case bool:
		_, err = se.encodeBool(v)
	case interface{}:
		return encodeTuple(v)
	default:
		return nil, errors.New("unsupported type")
	}

	return buffer.Bytes(), err
}

// encodeByteArray performs the following:
// b -> [encodeInteger(len(b)) b]
// it returns a byte array where the first byte is the length of b encoded with SCALE, followed by the byte array b itself
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
// if n < 2^6 return [00 i^2...i^8 ] [ 8 bits = 1 byte output ]
// if 2^6 <= n < 2^14 return [01 i^2...i^16] [ 16 bits = 2 byte output ]
// if 2^14 <= n < 2^30 return [10 i^2...i^32] [ 32 bits = 4 byte output ]
// if n >= 2^30 return [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (se *Encoder) encodeInteger(i int) (bytesEncoded int, err error) {
	if i < 1<<6 {
		err = binary.Write(se.writer, binary.LittleEndian, uint8(byte(i) << 2))
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
	binary.LittleEndian.PutUint64(o, uint64(i))
	err = binary.Write(se.writer, binary.LittleEndian, o[0:numBytes])

	return numBytes + 1, err
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
// l = true -> return [1]
// l = false -> return [0]
func (se *Encoder) encodeBool(l bool) (bytesEncoded int, err error) {
	if l {
		se.writer.Write([]byte{0x01})
		return 1, nil
	}
	se.writer.Write([]byte{0x00})
	return 1, nil
}

func encodeTuple(t interface{}) ([]byte, error) {
	v := reflect.ValueOf(t)

	values := make([]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
	}

	o := []byte{}
	for _, item := range values {
		encodedItem, err := Encode(item)
		if err != nil {
			return nil, err
		}

		o = append(o, encodedItem...)
	}

	return o, nil
}