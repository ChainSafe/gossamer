package scale

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	"reflect"
)

func Marshal(v interface{}) (b []byte, err error) {
	es := encodeState{
		fieldScaleIndicesCache: cache,
	}
	err = es.marshal(v)
	if err != nil {
		return
	}
	b = es.Bytes()
	return
}

type encodeState struct {
	bytes.Buffer
	*fieldScaleIndicesCache
}

func (es *encodeState) marshal(in interface{}) (err error) {
	switch in := in.(type) {
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		err = es.encodeFixedWidthInt(in)
	case *big.Int:
		err = es.encodeBigInt(in)
	case *Uint128:
		err = es.encodeUint128(in)
	case []byte:
		err = es.encodeBytes(in)
	case string:
		err = es.encodeBytes([]byte(in))
	case bool:
		err = es.encodeBool(in)
	default:
		switch reflect.TypeOf(in).Kind() {
		case reflect.Ptr:
			// Assuming that anything that is a pointer is an Option to capture {nil, T}
			elem := reflect.ValueOf(in).Elem()
			switch elem.IsValid() {
			case false:
				err = es.WriteByte(0)
			default:
				err = es.WriteByte(1)
				if err != nil {
					return
				}
				err = es.marshal(elem.Interface())
			}
		case reflect.Struct:
			err = es.encodeStruct(in)
		case reflect.Array:
			err = es.encodeArray(in)
		case reflect.Slice:
			t := reflect.TypeOf(in)
			// check if this is a convertible to VaryingDataType, if so encode using encodeVaryingDataType
			switch t.ConvertibleTo(reflect.TypeOf(VaryingDataType{})) {
			case true:
				invdt := reflect.ValueOf(in).Convert(reflect.TypeOf(VaryingDataType{}))
				switch in := invdt.Interface().(type) {
				case VaryingDataType:
					err = es.encodeVaryingDataType(in)
				default:
					log.Panicf("this should never happen")
				}
			case false:
				err = es.encodeSlice(in)
			}
		default:
			_, ok := in.(VaryingDataTypeValue)
			switch ok {
			case true:
				t := reflect.TypeOf(in)
				switch t.Kind() {
				// TODO: support more primitive types.  Do we need to support arrays and slices as well?
				case reflect.Int:
					in = reflect.ValueOf(in).Convert(reflect.TypeOf(int(1))).Interface()
				case reflect.Int16:
					in = reflect.ValueOf(in).Convert(reflect.TypeOf(int16(1))).Interface()
				}
				err = es.marshal(in)
			default:
				err = fmt.Errorf("unsupported type: %T", in)
			}

		}
	}
	return
}

func (es *encodeState) encodeVaryingDataType(values VaryingDataType) (err error) {
	err = es.encodeLength(len(values))
	if err != nil {
		return
	}
	for _, t := range values {
		// encode type.Index (idx) for varying data type
		err = es.WriteByte(byte(t.Index()))
		if err != nil {
			return
		}
		err = es.marshal(t)
	}
	return
}

func (es *encodeState) encodeSlice(t interface{}) (err error) {
	switch arr := t.(type) {
	// int is the only case that handles encoding differently.
	// all other cases can recursively call es.marshal()
	case []int:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.encodeUint(uint(elem))
			if err != nil {
				return
			}
		}
	// the cases below are to avoid using the reflect library
	// for commonly used slices in gossamer
	case []*big.Int:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.marshal(elem)
			if err != nil {
				return
			}
		}
	case []bool:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.marshal(elem)
			if err != nil {
				return
			}
		}
	case [][]byte:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.marshal(elem)
			if err != nil {
				return
			}
		}
	case [][]int:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.marshal(elem)
			if err != nil {
				return
			}
		}
	case []string:
		err = es.encodeLength(len(arr))
		if err != nil {
			return
		}
		for _, elem := range arr {
			err = es.marshal(elem)
			if err != nil {
				return
			}
		}
	default:
		// use reflect package for any other cases
		s := reflect.ValueOf(t)
		err = es.encodeUint(uint(s.Len()))
		if err != nil {
			return
		}
		for i := 0; i < s.Len(); i++ {
			err = es.marshal(s.Index(i).Interface())
			if err != nil {
				return
			}
		}
	}
	return
}

// encodeArray encodes an interface where the underlying type is an array
// it writes the encoded length of the Array to the Encoder, then encodes and writes each value in the Array
func (es *encodeState) encodeArray(in interface{}) (err error) {
	v := reflect.ValueOf(in)
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		switch elem := elem.(type) {
		case int:
			// an array of unsized integers needs to be encoded using scale length encoding
			err = es.encodeUint(uint(elem))
			if err != nil {
				return
			}
		default:
			err = es.marshal(v.Index(i).Interface())
			if err != nil {
				return
			}
		}
	}
	return
}

// encodeBigInt performs the same encoding as encodeInteger, except on a big.Int.
// if 2^30 <= n < 2^536 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (es *encodeState) encodeBigInt(i *big.Int) (err error) {
	switch {
	case i == nil:
		err = fmt.Errorf("nil *big.Int")
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 6)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint8(i.Int64()<<2))
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 14)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint16(i.Int64()<<2)+1)
	case i.Cmp(new(big.Int).Lsh(big.NewInt(1), 30)) < 0:
		err = binary.Write(es, binary.LittleEndian, uint32(i.Int64()<<2)+2)
	default:
		numBytes := len(i.Bytes())
		topSixBits := uint8(numBytes - 4)
		lengthByte := topSixBits<<2 + 3

		// write byte which encodes mode and length
		err = binary.Write(es, binary.LittleEndian, lengthByte)
		if err == nil {
			// write integer itself
			err = binary.Write(es, binary.LittleEndian, reverseBytes(i.Bytes()))
		}
	}
	return
}

// encodeBool performs the following:
// l = true -> write [1]
// l = false -> write [0]
func (es *encodeState) encodeBool(l bool) (err error) {
	switch l {
	case true:
		_, err = es.Write([]byte{0x01})
	case false:
		_, err = es.Write([]byte{0x00})
	}
	return
}

// encodeByteArray performs the following:
// b -> [encodeInteger(len(b)) b]
// it writes to the buffer a byte array where the first byte is the length of b encoded with SCALE, followed by the
// byte array b itself
func (es *encodeState) encodeBytes(b []byte) (err error) {
	err = es.encodeUint(uint(len(b)))
	if err != nil {
		return
	}

	_, err = es.Write(b)
	return
}

// encodeFixedWidthInt encodes an int with size < 2**32 by putting it into little endian byte format
func (es *encodeState) encodeFixedWidthInt(i interface{}) (err error) {
	switch i := i.(type) {
	case int8:
		err = binary.Write(es, binary.LittleEndian, byte(i))
	case uint8:
		err = binary.Write(es, binary.LittleEndian, i)
	case int16:
		err = binary.Write(es, binary.LittleEndian, uint16(i))
	case uint16:
		err = binary.Write(es, binary.LittleEndian, i)
	case int32:
		err = binary.Write(es, binary.LittleEndian, uint32(i))
	case uint32:
		err = binary.Write(es, binary.LittleEndian, i)
	case int64:
		err = binary.Write(es, binary.LittleEndian, uint64(i))
	case uint64:
		err = binary.Write(es, binary.LittleEndian, i)
	case int:
		err = binary.Write(es, binary.LittleEndian, int64(i))
	case uint:
		err = binary.Write(es, binary.LittleEndian, uint64(i))
	default:
		err = fmt.Errorf("could not encode fixed width integer, invalid type: %T", i)
	}
	return
}

// encodeStruct reads the number of fields in the struct and their types and writes to the buffer each of the struct fields
// encoded as their respective types
func (es *encodeState) encodeStruct(in interface{}) (err error) {
	v, indices, err := es.fieldScaleIndices(in)
	if err != nil {
		return
	}
	for _, i := range indices {
		field := v.Field(i.fieldIndex)
		if !field.CanInterface() {
			continue
		}
		err = es.marshal(field.Interface())
		if err != nil {
			return
		}
	}
	return
}

// encodeLength is a helper function that calls encodeUint, which is the scale length encoding
func (es *encodeState) encodeLength(l int) (err error) {
	return es.encodeUint(uint(l))
}

// encodeUint performs the following on integer i:
// i  -> i^0...i^n where n is the length in bits of i
// note that the bit representation of i is in little endian; ie i^0 is the least significant bit of i,
// and i^n is the most significant bit
// if n < 2^6 write [00 i^2...i^8 ] [ 8 bits = 1 byte encoded ]
// if 2^6 <= n < 2^14 write [01 i^2...i^16] [ 16 bits = 2 byte encoded ]
// if 2^14 <= n < 2^30 write [10 i^2...i^32] [ 32 bits = 4 byte encoded ]
// if n >= 2^30 write [lower 2 bits of first byte = 11] [upper 6 bits of first byte = # of bytes following less 4]
// [append i as a byte array to the first byte]
func (es *encodeState) encodeUint(i uint) (err error) {
	switch {
	case i < 1<<6:
		err = binary.Write(es, binary.LittleEndian, byte(i)<<2)
	case i < 1<<14:
		err = binary.Write(es, binary.LittleEndian, uint16(i<<2)+1)
	case i < 1<<30:
		err = binary.Write(es, binary.LittleEndian, uint32(i<<2)+2)
	default:
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

		err = binary.Write(es, binary.LittleEndian, lengthByte)
		if err == nil {
			binary.LittleEndian.PutUint64(o, uint64(i))
			err = binary.Write(es, binary.LittleEndian, o[0:numBytes])
		}
	}
	return
}

// encodeUint128 encodes a Uint128
func (es *encodeState) encodeUint128(i *Uint128) (err error) {
	err = binary.Write(es, binary.LittleEndian, i.Bytes())
	return
}
