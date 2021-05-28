package scale

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

func Unmarshal(data []byte, dst interface{}) (err error) {
	dstv := reflect.ValueOf(dst)
	if dstv.Kind() != reflect.Ptr || dstv.IsNil() {
		err = fmt.Errorf("unsupported dst: %T", dst)
		return
	}

	buf := &bytes.Buffer{}
	ds := decodeState{}
	_, err = buf.Write(data)
	if err != nil {
		return
	}
	ds.Buffer = *buf
	out, err := ds.unmarshal(dstv.Elem())
	if err != nil {
		return
	}

	elem := dstv.Elem()
	if out != nil {
		elem.Set(reflect.ValueOf(out))
	} else {
		elem.Set(reflect.Zero(elem.Type()))
	}
	return
}

type decodeState struct {
	bytes.Buffer
}

func (ds *decodeState) unmarshal(dstv reflect.Value) (out interface{}, err error) {
	in := dstv.Interface()
	switch in.(type) {
	case *big.Int:
		in, err = ds.decodeBigInt()
	case *Uint128:
		in, err = ds.decodeUint128()
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		in, err = ds.decodeFixedWidthInt(in)
	case []byte:
		in, err = ds.decodeBytes()
	case string:
		var b []byte
		b, err = ds.decodeBytes()
		in = string(b)
	case bool:
		in, err = ds.decodeBool()
	// case VaryingDataType:
	// 	in, err = ds.decodeVaryingDataType()
	default:
		switch reflect.TypeOf(in).Kind() {
		case reflect.Ptr:
			var rb byte
			rb, err = ds.ReadByte()
			if err != nil {
				break
			}
			switch rb {
			case 0x00:
				in = nil
			case 0x01:
				elem := reflect.ValueOf(in).Elem()
				out := elem.Interface()
				out, err = ds.unmarshal(elem)
				if err != nil {
					break
				}
				in = &out
			default:
				err = fmt.Errorf("unsupported Option value: %v, bytes: %v", rb, ds.Bytes())
			}
		case reflect.Struct:
			in, err = ds.decodeStruct(in)
		case reflect.Array:
			in, err = ds.decodeArray(in)
		case reflect.Slice:
			t := reflect.TypeOf(in)
			// check if this is a convertible to VaryingDataType, if so encode using encodeVaryingDataType
			switch t.ConvertibleTo(reflect.TypeOf(VaryingDataType{})) {
			case true:
				in, err = ds.decodeVaryingDataType(in)
			case false:
				in, err = ds.decodeSlice(in)
			}

		default:
			err = fmt.Errorf("unsupported type: %T", in)
		}
	}

	if err != nil {
		return
	}
	out = in
	return
}

func (ds *decodeState) decodeVaryingDataType(dst interface{}) (vdt interface{}, err error) {
	l, err := ds.decodeLength()
	if err != nil {
		return nil, err
	}

	dstt := reflect.TypeOf(dst)
	key := fmt.Sprintf("%s.%s", dstt.PkgPath(), dstt.Name())
	mappedValues, ok := vdtCache[key]
	if !ok {
		err = fmt.Errorf("unable to find registered custom VaryingDataType: %T", dst)
		return
	}

	temp := reflect.New(dstt)
	for i := 0; i < l; i++ {
		var b byte
		b, err = ds.ReadByte()
		if err != nil {
			return
		}

		val, ok := mappedValues[uint(b)]
		if !ok {
			err = fmt.Errorf("unable to find registered VaryingDataTypeValue for type: %T", dst)
			return
		}

		tempVal := reflect.New(reflect.TypeOf(val)).Elem()
		var out interface{}
		out, err = ds.unmarshal(tempVal)
		if err != nil {
			return
		}

		if reflect.ValueOf(out).IsValid() {
			tempVal.Set(reflect.ValueOf(out))
		} else {
			tempVal.Set(reflect.Zero(tempVal.Type()))
		}
		temp.Elem().Set(reflect.Append(temp.Elem(), tempVal))
	}
	vdt = temp.Elem().Interface()
	return
}

// decodeArray comment about this
func (ds *decodeState) decodeSlice(dst interface{}) (arr interface{}, err error) {
	dstv := reflect.ValueOf(dst)
	// seems redundant since this shouldn't be called unless it is a slice,
	// but I'll leave this error check in for completeness
	if dstv.Kind() != reflect.Slice {
		err = fmt.Errorf("can not decodeSlice dst type: %T", dstv)
		return
	}

	l, err := ds.decodeLength()
	if err != nil {
		return nil, err
	}

	switch dst.(type) {
	case []int:
		temp := make([]int, l)
		for i := 0; i < l; i++ {
			var ui uint64
			ui, err = ds.decodeUint()
			if err != nil {
				return
			}
			temp[i] = int(ui)
		}
		arr = temp
	default:
		temp := reflect.New(reflect.TypeOf(dst))
		for i := 0; i < l; i++ {
			tempElemType := reflect.TypeOf(dst).Elem()
			tempElem := reflect.New(tempElemType).Elem()

			var out interface{}
			out, err = ds.unmarshal(tempElem)
			if err != nil {
				return
			}

			if reflect.ValueOf(out).IsValid() {
				tempElem.Set(reflect.ValueOf(out))
			} else {
				tempElem.Set(reflect.Zero(tempElem.Type()))
			}
			temp.Elem().Set(reflect.Append(temp.Elem(), tempElem))
		}
		arr = temp.Elem().Interface()
	}

	return
}

// decodeArray comment about this
func (ds *decodeState) decodeArray(dst interface{}) (arr interface{}, err error) {
	dstv := reflect.ValueOf(dst)
	// seems redundant since this shouldn't be called unless it is an array,
	// but I'll leave this error check in for completeness
	if dstv.Kind() != reflect.Array {
		err = fmt.Errorf("can not decodeArray dst type: %T", dstv)
		return
	}

	// temp := reflect.New(reflect.Indirect(reflect.ValueOf(dst)).Type())
	temp := reflect.New(reflect.TypeOf(dst))
	for i := 0; i < dstv.Len(); i++ {
		elem := dstv.Index(i)
		elemIn := elem.Interface()

		var out interface{}
		switch elemIn.(type) {
		case int:
			var ui uint64
			ui, err = ds.decodeUint()
			if err != nil {
				return
			}
			out = int(ui)
		default:
			out, err = ds.unmarshal(elem)
			if err != nil {
				return
			}
		}

		tempElem := temp.Elem().Index(i)
		if reflect.ValueOf(out).IsValid() {
			tempElem.Set(reflect.ValueOf(out))
		} else {
			tempElem.Set(reflect.Zero(tempElem.Type()))
		}
	}

	arr = temp.Elem().Interface()
	return
}

// decodeStruct comment about this
func (ds *decodeState) decodeStruct(dst interface{}) (s interface{}, err error) {
	dstv := reflect.ValueOf(dst)
	// seems redundant since this shouldn't be called unless it is a struct,
	// but I'll leave this error check in for completeness
	if dstv.Kind() != reflect.Struct {
		err = fmt.Errorf("can not decodeStruct dst type: %T", dstv)
		return
	}

	v, indices, err := cache.fieldScaleIndices(dst)
	if err != nil {
		return
	}

	temp := reflect.New(reflect.TypeOf(dst))
	// temp := reflect.New(reflect.Indirect(reflect.ValueOf(dst)).Type())
	for _, i := range indices {
		field := v.Field(i.fieldIndex)
		if !field.CanInterface() {
			continue
		}

		var out interface{}
		out, err = ds.unmarshal(field)
		if err != nil {
			err = fmt.Errorf("%s, field: %+v", err, field)
			return
		}
		if out == nil {
			continue
		}

		tempElem := temp.Elem().Field(i.fieldIndex)
		if reflect.ValueOf(out).IsValid() {
			tempElem.Set(reflect.ValueOf(out))
		} else {
			tempElem.Set(reflect.Zero(tempElem.Type()))
		}
	}
	s = temp.Elem().Interface()
	return
}

// decodeBool accepts a byte array representing a SCALE encoded bool and performs SCALE decoding
// of the bool then returns it. if invalid, return false and an error
func (ds *decodeState) decodeBool() (b bool, err error) {
	rb, err := ds.ReadByte()
	if err != nil {
		return
	}

	switch rb {
	case 0x00:
	case 0x01:
		b = true
	default:
		err = fmt.Errorf("could not decode invalid bool")
	}
	return
}

// DecodeUnsignedInteger will decode unsigned integer
func (ds *decodeState) decodeUint() (o uint64, err error) {
	b, err := ds.ReadByte()
	if err != nil {
		return 0, err
	}

	// check mode of encoding, stored at 2 least significant bits
	mode := b & 3
	if mode <= 2 {
		val, e := ds.decodeSmallInt(b, mode)
		return uint64(val), e
	}

	// >4 byte mode
	topSixBits := b >> 2
	byteLen := uint(topSixBits) + 4

	buf := make([]byte, byteLen)
	_, err = ds.Read(buf)
	if err != nil {
		return 0, err
	}

	if byteLen == 4 {
		o = uint64(binary.LittleEndian.Uint32(buf))
	} else if byteLen > 4 && byteLen < 8 {
		tmp := make([]byte, 8)
		copy(tmp, buf)
		o = binary.LittleEndian.Uint64(tmp)
	} else {
		err = errors.New("could not decode invalid integer")
	}

	return o, err
}

// decodeLength accepts a byte array representing a SCALE encoded integer and performs SCALE decoding of the int
// if the encoding is valid, it then returns (o, bytesDecoded, err) where o is the decoded integer, bytesDecoded is the
// number of input bytes decoded, and err is nil
// otherwise, it returns 0, 0, and error
func (ds *decodeState) decodeLength() (l int, err error) {
	ui, err := ds.decodeUint()
	l = int(ui)
	return
}

// DecodeByteArray accepts a byte array representing a SCALE encoded byte array and performs SCALE decoding
// of the byte array
// if the encoding is valid, it then returns the decoded byte array, the total number of input bytes decoded, and nil
// otherwise, it returns nil, 0, and error
func (ds *decodeState) decodeBytes() (b []byte, err error) {
	length, err := ds.decodeLength()
	if err != nil {
		return nil, err
	}

	b = make([]byte, length)
	_, err = ds.Read(b)
	if err != nil {
		return nil, errors.New("could not decode invalid byte array: reached early EOF")
	}
	return
}

// decodeSmallInt is used in the DecodeInteger and decodeBigInt functions when the mode is <= 2
// need to pass in the first byte, since we assume it's already been read
func (ds *decodeState) decodeSmallInt(firstByte, mode byte) (out int64, err error) {
	switch mode {
	case 0:
		out = int64(firstByte >> 2)
	case 1:
		var buf byte
		buf, err = ds.ReadByte()
		if err != nil {
			break
		}
		out = int64(binary.LittleEndian.Uint16([]byte{firstByte, buf}) >> 2)
	case 2:
		buf := make([]byte, 3)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = int64(binary.LittleEndian.Uint32(append([]byte{firstByte}, buf...)) >> 2)
	}
	return
}

// decodeBigInt decodes a SCALE encoded byte array into a *big.Int
// Works for all integers, including ints > 2**64
func (ds *decodeState) decodeBigInt() (output *big.Int, err error) {
	b, err := ds.ReadByte()
	if err != nil {
		return
	}

	// check mode of encoding, stored at 2 least significant bits
	mode := b & 0x03
	switch {
	case mode <= 2:
		var tmp int64
		tmp, err = ds.decodeSmallInt(b, mode)
		if err != nil {
			break
		}
		output = big.NewInt(tmp)

	default:
		// >4 byte mode
		topSixBits := b >> 2
		byteLen := uint(topSixBits) + 4

		buf := make([]byte, byteLen)
		_, err = ds.Read(buf)
		if err == nil {
			o := reverseBytes(buf)
			output = big.NewInt(0).SetBytes(o)
		} else {
			err = errors.New("could not decode invalid big.Int: reached early EOF")
		}
	}
	return
}

// decodeFixedWidthInt decodes integers < 2**32 by reading the bytes in little endian
func (ds *decodeState) decodeFixedWidthInt(in interface{}) (out interface{}, err error) {
	switch in.(type) {
	case int8:
		var b byte
		b, err = ds.ReadByte()
		if err != nil {
			break
		}
		out = int8(b)
	case uint8:
		var b byte
		b, err = ds.ReadByte()
		if err != nil {
			break
		}
		out = uint8(b)
	case int16:
		buf := make([]byte, 2)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = int16(binary.LittleEndian.Uint16(buf))
	case uint16:
		buf := make([]byte, 2)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = binary.LittleEndian.Uint16(buf)
	case int32:
		buf := make([]byte, 4)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = int32(binary.LittleEndian.Uint32(buf))
	case uint32:
		buf := make([]byte, 4)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = binary.LittleEndian.Uint32(buf)
	case int64:
		buf := make([]byte, 8)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = int64(binary.LittleEndian.Uint64(buf))
	case uint64:
		buf := make([]byte, 8)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = binary.LittleEndian.Uint64(buf)
	case int:
		buf := make([]byte, 8)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = int(binary.LittleEndian.Uint64(buf))
	case uint:
		buf := make([]byte, 8)
		_, err = ds.Read(buf)
		if err != nil {
			break
		}
		out = uint(binary.LittleEndian.Uint64(buf))
	}
	return
}

// decodeUint128 accepts a byte array representing Scale encoded common.Uint128 and performs SCALE decoding of the Uint128
// if the encoding is valid, it then returns (i interface{}, nil) where i is the decoded common.Uint128 , otherwise
// it returns nil and error
func (ds *decodeState) decodeUint128() (ui *Uint128, err error) {
	buf := make([]byte, 16)
	err = binary.Read(ds, binary.LittleEndian, buf)
	if err != nil {
		return nil, err
	}
	return NewUint128(buf)
}
