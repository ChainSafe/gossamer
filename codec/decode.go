package codec

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"reflect"
)

// Decoder is a wrapping around io.Reader
type Decoder struct {
	reader io.Reader
}

// Decode is the high level function wrapping the specific type decoding functions
func (sd *Decoder) Decode(t interface{}) (out interface{}, err error) {
	switch t.(type) {
	case *big.Int:
		out, err = sd.DecodeBigInt()
	case int8, int16, int32, int64:
		out, err = sd.DecodeInteger()
	case []byte:
		out, err = sd.DecodeByteArray()
	case bool:
		out, err = sd.DecodeBool()
	case interface{}:
		out, err = sd.DecodeTuple(t)
	default:
		return nil, errors.New("decode error: unsupported type")
	}
	return out, err
}

// ReadByte reads the one byte from the buffer
func (sd *Decoder) ReadByte() (byte, error) {
	b := make([]byte, 1)        // make buffer
	_, err := sd.reader.Read(b) // read what's in the Decoder's underlying buffer to our new buffer b
	return b[0], err
}

// decodeSmallInt is used in the DecodeInteger and DecodeBigInteger functions when the mode is <= 2
// need to pass in the first byte, since we assume it's already been read
func (sd *Decoder) decodeSmallInt(firstByte byte) (o int64, err error) {
	mode := firstByte & 3
	if mode == 0 { // 1 byte mode
		o = int64(firstByte >> 2)
	} else if mode == 1 { // 2 byte mode
		buf, err := sd.ReadByte()
		if err == nil {
			o = int64(binary.LittleEndian.Uint16([]byte{firstByte, buf}) >> 2)
		}
	} else if mode == 2 { // 4 byte mode
		buf := make([]byte, 3)
		_, err := sd.reader.Read(buf)
		if err == nil {
			o = int64(binary.LittleEndian.Uint32(append([]byte{firstByte}, buf...)) >> 2)
		}
	} else {
		err = errors.New("could not decode small int: mode not <= 2")
	}

	return o, err
}

// DecodeInteger accepts a byte array representing a SCALE encoded integer and performs SCALE decoding of the int
// if the encoding is valid, it then returns (o, bytesDecoded, err) where o is the decoded integer, bytesDecoded is the
// number of input bytes decoded, and err is nil
// otherwise, it returns 0, 0, and error
func (sd *Decoder) DecodeInteger() (o int64, err error) {
	b, err := sd.ReadByte()
	if err != nil {
		return 0, err
	}

	// check mode of encoding, stored at 2 least significant bits
	mode := b & 3
	if mode <= 2 {
		return sd.decodeSmallInt(b)
	}

	// >4 byte mode
	topSixBits := b >> 2
	byteLen := int(topSixBits) + 4 

	buf := make([]byte, byteLen)
	_, err = sd.reader.Read(buf)
	if err != nil {
		return 0, err
	}

	if err == nil {
		if byteLen == 4 {
			o = int64(binary.LittleEndian.Uint32(buf))
		} else if byteLen > 4 && byteLen < 8 {
			tmp := make([]byte, 8)
			copy(tmp, buf)
			o = int64(binary.LittleEndian.Uint64(tmp))
		}

		if o == 0 {
			err = errors.New("could not decode invalid integer")
		}
	}

	return o, err
}

// DecodeBigInt decodes a SCALE encoded byte array into a *big.Int
// Works for all integers, including ints > 2**64
func (sd *Decoder) DecodeBigInt() (output *big.Int, err error) {
	b, err := sd.ReadByte()
	if err != nil {
		return nil, err
	}

	// check mode of encoding, stored at 2 least significant bits
	mode := b & 0x03
	if mode <= 2 {
		tmp, err := sd.decodeSmallInt(b)
		if err != nil {
			return nil, err
		}
		return new(big.Int).SetInt64(tmp), nil
	}

	// >4 byte mode
	topSixBits := b >> 2
	byteLen := int(topSixBits) + 4

	buf := make([]byte, byteLen)
	_, err = sd.reader.Read(buf)
	if err == nil {
		o := reverseBytes(buf)
		output = new(big.Int).SetBytes(o)
	} else {
		err = errors.New("could not decode invalid big.Int: reached early EOF")
	}

	return output, err
}

// DecodeByteArray accepts a byte array representing a SCALE encoded byte array and performs SCALE decoding
// of the byte array
// if the encoding is valid, it then returns the decoded byte array, the total number of input bytes decoded, and nil
// otherwise, it returns nil, 0, and error
func (sd *Decoder) DecodeByteArray() (o []byte, err error) {
	length, err := sd.DecodeInteger()
	if err != nil {
		return nil, err
	}

	b := make([]byte, length)
	_, err = sd.reader.Read(b)
	if err != nil {
		return nil, errors.New("could not decode invalid byte array: reached early EOF")
	}

	return b, nil
}

// DecodeBool accepts a byte array representing a SCALE encoded bool and performs SCALE decoding
// of the bool then returns it. if invalid, return false and an error
func (sd *Decoder) DecodeBool() (bool, error) {
	b, err := sd.ReadByte()
	if err != nil {
		return false, err
	}

	if b == 1 {
		return true, nil
	} else if b == 0 {
		return false, nil
	}

	return false, errors.New("cannot decode invalid boolean")
}

// DecodeTuple accepts a byte array representing the SCALE encoded tuple and an interface. This interface should be a pointer
// to a struct which the encoded tuple should be marshalled into. If it is a valid encoding for the struct, it returns the
// decoded struct, otherwise error,
// Note that we return the same interface that was passed to this function; this is because we are writing directly to the
// struct that is passed in, using reflect to get each of the fields.
func (sd *Decoder) DecodeTuple(t interface{}) (interface{}, error) {
	v := reflect.ValueOf(t).Elem()

	var err error
	var o interface{}

	val := reflect.Indirect(reflect.ValueOf(t))

	// iterate through each field in the struct
	for i := 0; i < v.NumField(); i++ {
		// get the field value at i
		fieldValue := val.Field(i)

		switch v.Field(i).Interface().(type) {
		case []byte:
			o, err = sd.DecodeByteArray()
			if err != nil {
				break
			}

			// get the pointer to the value and set the value
			ptr := fieldValue.Addr().Interface().(*[]byte)
			*ptr = o.([]byte)
		case int8, int16, int32, int64:
			o, err = sd.DecodeInteger()
			if err != nil {
				break
			}

			ptr := fieldValue.Addr().Interface().(*int64)
			*ptr = o.(int64)
		case bool:
			o, err = sd.DecodeBool()
			if err != nil {
				break
			}

			ptr := fieldValue.Addr().Interface().(*bool)
			*ptr = o.(bool)
		}

		if err != nil {
			break
		}
	}

	return t, err
}
