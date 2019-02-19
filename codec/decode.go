package codec

import (
	"encoding/binary"
	"errors"
)

// DecodeInteger accepts a byte array representing a SCALE encoded integer and performs SCALE decoding of the
// int, then returns it
func DecodeInteger(b []byte) (int64, error) {
	if len(b) == 1 {
		return int64(b[0] >> 2), nil
	} else if len(b) == 2 {
		o := binary.LittleEndian.Uint16(b) >> 2
		return int64(o), nil
	} else if len(b) == 4 {
		o := binary.LittleEndian.Uint32(b) >> 2
		return int64(o), nil
	} 

	topSixBits := (binary.LittleEndian.Uint16(b) & 0xff) >> 2
	byteLen := topSixBits + 4

	if byteLen == 4 {
		return int64(binary.LittleEndian.Uint32(b[1 : byteLen+1])), nil
	} else if byteLen > 4 && byteLen < 8 {
		upperBytes := make([]byte, 8-byteLen)
		upperBytes = append(b[5:byteLen+1], upperBytes...)
		upper := int64(binary.LittleEndian.Uint32(upperBytes)) << 32
		lower := int64(binary.LittleEndian.Uint32(b[1:5]))
		return upper + lower, nil
	}
	
	return int64(0), nil
}

// DecodeByteArray accepts a byte array representing a SCALE encoded byte array and performs SCALE decoding
// of the byte array, then returns it.  If it is invalid, return nil and error
func DecodeByteArray(b []byte) ([]byte, error) {
	if b[0] & 0x03 == 0 { // encoding of length: 1 byte mode
		length, err := DecodeInteger([]byte{b[0]})
		if err != nil {
			return nil, err
		}

		if length == 0 || length > 1 << 6 || int64(len(b)) < length + 1 {
			return nil, errors.New("could not decode invalid byte array")
		}

		return b[1:length+1], nil
	} else if b[0] & 0x03 == 1 { // encoding of length: 2 byte mode
		// pass first two bytes of byte array to decode length
		length, err := DecodeInteger(b[0:2]) 
		if err != nil {
			return nil, err
		}

		if length < 1 << 6 || length > 1 << 14 || int64(len(b)) < length + 2 { 
			return nil, errors.New("could not decode invalid byte array")
		}

		return b[2:length+2], nil
	} else if b[0] & 0x03 == 2 { // encoding of length: 4 byte mode
		// pass first four bytes of byte array to decode length
		length, err := DecodeInteger(b[0:4]) 
		if err != nil {
			return nil, err
		}

		if length < 1 << 14 || length > 1 << 30 || int64(len(b)) < length + 4 {
			return nil, errors.New("could not decode invalid byte array")
		}

		return b[4:length+4], nil
	} else if b[0] & 0x03 == 3 { // encoding of length: big-integer mode
		length, err := DecodeInteger(b)
		if err != nil {
			return nil, err
		}

		// get the length of the encoded length
		topSixBits := (binary.LittleEndian.Uint16(b) & 0xff) >> 2
		byteLen := topSixBits + 4

		if length < 1 << 30  || int64(len(b)) < length + int64(byteLen) {
			return nil, errors.New("could not decode invalid byte array")
		}

		return b[int64(byteLen):length+int64(byteLen)], nil
	}

	return nil, errors.New("could not decode invalid byte array")
}

// DecodeBool accepts a byte array representing a SCALE encoded bool and performs SCALE decoding
// of the bool then returns it. if invalid, return false and an error
func DecodeBool(b byte) (bool, error) {
	if b == 1 {
		return true, nil
	} else if b == 0 {
		return false, nil
	}

	return false, errors.New("cannot decode invalid boolean")
}