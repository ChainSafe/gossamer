package codec

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"reflect"
)

type Decoder struct {
	reader io.Reader
}

func (sd *Decoder) Decode(t interface{}) (out interface{}, err error) {
	//v := reflect.ValueOf(t).Elem()

	switch reflect.TypeOf(t).Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out, err = sd.DecodeInteger() // assign decoded value
	// case reflect.Struct:
	// 	b := make([]byte, reflect.TypeOf(t).Size())
	// 	sd.reader.Read(b)
	// 	//ptr := v.Addr().Interface()
	// 	fmt.Println(t)
	// 	fmt.Println(&t)
	// 	out, err = DecodeTuple(b, t)		

	case reflect.Ptr:
		out, err = sd.DecodeTuple(t)		
		//ptr = out
	}

	return out, err
}

// ReadByte reads the one byte from the buffer
func (sd *Decoder) ReadByte() (byte, error) {
	b := make([]byte, 1) // make buffer
	_, err := sd.reader.Read(b) // read what's in the Decoder's underlying buffer to our new buffer b
	return b[0], err
}

// decodeSmallInt is used in the DecodeInteger and DecodeBigInteger functions when the mode is <= 2
// need to pass in the first byte, since we assume it's already been read
func (sd *Decoder) decodeSmallInt(firstByte byte) (o int64, err error) {
	mode := firstByte & 3
	if mode == 0 { // 1 byte mode
		return int64(firstByte >> 2), nil
	} else if mode == 1 { // 2 byte mode
		c, err := sd.ReadByte()
		if err != nil {
			return 0, err
		}
		o := binary.LittleEndian.Uint16([]byte{firstByte,c}) >> 2
		return int64(o), nil
	} else if mode == 2 { // 4 byte mode
		c := make([]byte, 3)
		_, err := sd.reader.Read(c)
		if err != nil {
			return 0, err
		}
		o := binary.LittleEndian.Uint32(append([]byte{firstByte}, c...)) >> 2
		return int64(o), nil
	}

	return 0, errors.New("could not decode small int: mode not <= 2")
}

// DecodeInteger accepts a byte array representing a SCALE encoded integer and performs SCALE decoding of the int
// if the encoding is valid, it then returns (o, bytesDecoded, err) where o is the decoded integer, bytesDecoded is the
// number of input bytes decoded, and err is nil
// otherwise, it returns 0, 0, and error
func (sd *Decoder) DecodeInteger() (o int64, err error) {
	//b := make([]byte, 1) // make buffer
	//bytesDecoded, err = sd.reader.Read(b) // read what's in the Decoder's underlying buffer to our new buffer b

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

	c := make([]byte, byteLen)
	_, err = sd.reader.Read(c)
	if err != nil {
		return 0, err
	}

	if err == nil {
		if byteLen == 4 {
			o = int64(binary.LittleEndian.Uint32(c))
		} else if byteLen > 4 && byteLen < 8 {
			d := make([]byte, 8)
			copy(d, c)
			o = int64(binary.LittleEndian.Uint64(d))
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

	// if len(b) < int(byteLen)+1 {
	// 	err = errors.New("could not decode invalid integer")
	// }

	c := make([]byte, byteLen)
	_, err = sd.reader.Read(c)
	if err != nil {
		return nil, errors.New("could not decode invalid big.Int")
	}

	o := reverseBytes(c)
	output = new(big.Int).SetBytes(o)

	return output, nil
}

// DecodeByteArray accepts a byte array representing a SCALE encoded byte array and performs SCALE decoding
// of the byte array
// if the encoding is valid, it then returns the decoded byte array, the total number of input bytes decoded, and nil
// otherwise, it returns nil, 0, and error
func (sd *Decoder) DecodeByteArray() (o []byte, bytesDecoded int64, err error) {
	//var length int64
	b := make([]byte, 1) // make buffer
	sd.reader.Read(b) // read what's in the Decoder's underlying buffer to our new buffer b

	// check mode of encoding, stored at 2 least significant bits
	mode := b[0] & 0x03
	if mode == 0 { // encoding of length: 1 byte mode
		//length, _, err = sd.DecodeInteger([]byte{b[0]})
		length, err := sd.DecodeInteger()
		if err == nil {
			if length == 0 || length > 1<<6 || int64(len(b)) < length+1 {
				err = errors.New("could not decode invalid byte array")
			} else {
				o = b[1 : length+1]
				bytesDecoded = length + 1
			}
		}
	} else if mode == 1 { // encoding of length: 2 byte mode
		// pass first two bytes of byte array to decode length
		//length, _, err = sd.DecodeInteger(b[0:2])
		length, err := sd.DecodeInteger()

		if err == nil {
			if length < 1<<6 || length > 1<<14 || int64(len(b)) < length+2 {
				err = errors.New("could not decode invalid byte array")
			} else {
				o = b[2 : length+2]
				bytesDecoded = length + 2
			}
		}
	} else if mode == 2 { // encoding of length: 4 byte mode
		// pass first four bytes of byte array to decode length
		//length, _, err = sd.DecodeInteger(b[0:4])
		length, err := sd.DecodeInteger()

		if err == nil {
			if length < 1<<14 || length > 1<<30 || int64(len(b)) < length+4 {
				err = errors.New("could not decode invalid byte array")
			} else {
				o = b[4 : length+4]
				bytesDecoded = length + 4
			}
		}
	} else if mode == 3 { // encoding of length: big-integer mode
		//length, _, err = sd.DecodeInteger(b)
		length, err := sd.DecodeInteger()

		if err == nil {
			// get the length of the encoded length
			topSixBits := (binary.LittleEndian.Uint16(b) & 0xff) >> 2
			byteLen := topSixBits + 4

			if length < 1<<30 || int64(len(b)) < (length+int64(byteLen))+1 {
				err = errors.New("could not decode invalid byte array")
			} else {
				o = b[int64(byteLen)+1 : length+int64(byteLen)+1]
				bytesDecoded = int64(byteLen) + length + 1
			}
		}
	}

	return o, bytesDecoded, err
}

// DecodeBool accepts a byte array representing a SCALE encoded bool and performs SCALE decoding
// of the bool then returns it. if invalid, return false and an error
func (sd *Decoder) DecodeBool() (bool, error) {
	b := make([]byte, 1)
	sd.reader.Read(b)

	if b[0] == 1 {
		return true, nil
	} else if b[0] == 0 {
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

	var bytesDecoded int64
	var byteLen int64
	var err error
	var o interface{}

	val := reflect.Indirect(reflect.ValueOf(t))

	// iterate through each field in the struct
	for i := 0; i < v.NumField(); i++ {

		// b := make([]byte, reflect.TypeOf(v.Field(i)).Size()) // make buffer
		// fmt.Println(reflect.TypeOf(v.Field(i).Interface()).Size())
		// sd.reader.Read(b) // read what's in the Decoder's underlying buffer to our new buffer b

		// get the field value at i
		fieldValue := val.Field(i)

		switch v.Field(i).Interface().(type) {
		case []byte:
			//b := make([]byte, sizeof(v))
			//o, byteLen, err = sd.DecodeByteArray(b[bytesDecoded:])
			o, byteLen, err = sd.DecodeByteArray()
			if err != nil {
				break
			}
			// get the pointer to the value and set the value
			ptr := fieldValue.Addr().Interface().(*[]byte)
			*ptr = o.([]byte)
		case int8, int16, int32, int64:
			//o, byteLen, err = sd.DecodeInteger(b[bytesDecoded:])
			o, byteLen, err = sd.DecodeByteArray()

			if err != nil {
				break
			}
			// get the pointer to the value and set the value
			ptr := fieldValue.Addr().Interface().(*int64)
			*ptr = o.(int64)
		case bool:
			//o, err = sd.DecodeBool(b[bytesDecoded])
			o, byteLen, err = sd.DecodeByteArray()

			if err != nil {
				break
			}
			// get the pointer to the value and set the value
			ptr := fieldValue.Addr().Interface().(*bool)
			*ptr = o.(bool)
			byteLen = 1
		}

		// if len(b) < int(bytesDecoded)+1 {
		// 	err = errors.New("could not decode invalid byte array into tuple")
		// }
		bytesDecoded = bytesDecoded + byteLen
		if err != nil {
			break
		}
	}

	return t, err
}
