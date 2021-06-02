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
		err = fmt.Errorf("unsupported dst: %T, must be a pointer to a destination", dst)
		return
	}

	buf := &bytes.Buffer{}
	ds := decodeState{}
	_, err = buf.Write(data)
	if err != nil {
		return
	}
	ds.Buffer = *buf
	err = ds.unmarshal(dstv.Elem())
	if err != nil {
		return
	}
	return
}

type decodeState struct {
	bytes.Buffer
}

func (ds *decodeState) unmarshal(dstv reflect.Value) (err error) {
	in := dstv.Interface()
	switch in.(type) {
	case *big.Int:
		err = ds.decodeBigInt(dstv)
	case *Uint128:
		err = ds.decodeUint128(dstv)
	case int, uint:
		err = ds.decodeUint(dstv)
	case int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		err = ds.decodeFixedWidthInt(dstv)
	case []byte:
		err = ds.decodeBytes(dstv)
	case string:
		err = ds.decodeBytes(dstv)
	case bool:
		err = ds.decodeBool(dstv)
	default:
		switch reflect.TypeOf(in).Kind() {
		case reflect.Ptr:
			err = ds.decodePointer(dstv)
		case reflect.Struct:
			err = ds.decodeStruct(dstv)
		case reflect.Array:
			err = ds.decodeArray(dstv)
		case reflect.Slice:
			t := reflect.TypeOf(in)
			// check if this is a convertible to VaryingDataType, if so encode using encodeVaryingDataType
			switch t.ConvertibleTo(reflect.TypeOf(VaryingDataType{})) {
			case true:
				err = ds.decodeVaryingDataType(dstv)
			case false:
				err = ds.decodeSlice(dstv)
			}
		default:
			_, ok := in.(VaryingDataTypeValue)
			switch ok {
			case true:
				var temp reflect.Value
				t := reflect.TypeOf(in)
				switch t.Kind() {
				// TODO: support more primitive types.  Do we need to support arrays and slices as well?
				case reflect.Int:
					temp = reflect.New(reflect.TypeOf(int(1)))
					err = ds.unmarshal(temp.Elem())
					if err != nil {
						break
					}
				case reflect.Int16:
					temp = reflect.New(reflect.TypeOf(int16(1)))
					err = ds.unmarshal(temp.Elem())
					if err != nil {
						break
					}
				default:
					err = fmt.Errorf("unsupported kind for VaryingDataTypeValue: %s", t.Kind())
					return
				}
				dstv.Set(temp.Elem().Convert(t))
			default:
				err = fmt.Errorf("unsupported type: %T", in)
			}
		}
	}
	return
}

func (ds *decodeState) decodePointer(dstv reflect.Value) (err error) {
	var rb byte
	rb, err = ds.ReadByte()
	if err != nil {
		return
	}
	switch rb {
	case 0x00:
		// nil case
	case 0x01:
		elemType := reflect.TypeOf(dstv.Interface()).Elem()
		tempElem := reflect.New(elemType)
		err = ds.unmarshal(tempElem.Elem())
		if err != nil {
			break
		}
		dstv.Set(tempElem)
	default:
		err = fmt.Errorf("unsupported Option value: %v, bytes: %v", rb, ds.Bytes())
	}
	return
}

func (ds *decodeState) decodeVaryingDataType(dstv reflect.Value) (err error) {
	l, err := ds.decodeLength()
	if err != nil {
		return
	}

	dstt := reflect.TypeOf(dstv.Interface())
	key := fmt.Sprintf("%s.%s", dstt.PkgPath(), dstt.Name())
	mappedValues, ok := vdtCache[key]
	if !ok {
		err = fmt.Errorf("unable to find registered custom VaryingDataType: %T", dstv.Interface())
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
			err = fmt.Errorf("unable to find registered VaryingDataTypeValue for type: %T", dstv.Interface())
			return
		}

		tempVal := reflect.New(reflect.TypeOf(val)).Elem()
		err = ds.unmarshal(tempVal)
		if err != nil {
			return
		}

		temp.Elem().Set(reflect.Append(temp.Elem(), tempVal))
	}
	dstv.Set(temp.Elem())
	return
}

func (ds *decodeState) decodeSlice(dstv reflect.Value) (err error) {
	l, err := ds.decodeLength()
	if err != nil {
		return
	}
	in := dstv.Interface()
	temp := reflect.New(reflect.ValueOf(in).Type())
	for i := 0; i < l; i++ {
		tempElemType := reflect.TypeOf(in).Elem()
		tempElem := reflect.New(tempElemType).Elem()

		err = ds.unmarshal(tempElem)
		if err != nil {
			return
		}
		temp.Elem().Set(reflect.Append(temp.Elem(), tempElem))
	}
	dstv.Set(temp.Elem())

	return
}

func (ds *decodeState) decodeArray(dstv reflect.Value) (err error) {
	in := dstv.Interface()
	temp := reflect.New(reflect.ValueOf(in).Type())
	for i := 0; i < temp.Elem().Len(); i++ {
		elem := temp.Elem().Index(i)
		err = ds.unmarshal(elem)
		if err != nil {
			return
		}
	}
	dstv.Set(temp.Elem())
	return
}

// decodeStruct comment about this
func (ds *decodeState) decodeStruct(dstv reflect.Value) (err error) {
	in := dstv.Interface()
	_, indices, err := cache.fieldScaleIndices(in)
	if err != nil {
		return
	}
	temp := reflect.New(reflect.ValueOf(in).Type())
	for _, i := range indices {
		field := temp.Elem().Field(i.fieldIndex)
		if !field.CanInterface() {
			continue
		}
		err = ds.unmarshal(field)
		if err != nil {
			err = fmt.Errorf("%s, field: %+v", err, field)
			return
		}
	}
	dstv.Set(temp.Elem())
	return
}

// decodeBool accepts a byte array representing a SCALE encoded bool and performs SCALE decoding
// of the bool then returns it. if invalid returns an error
func (ds *decodeState) decodeBool(dstv reflect.Value) (err error) {
	rb, err := ds.ReadByte()
	if err != nil {
		return
	}

	var b bool
	switch rb {
	case 0x00:
	case 0x01:
		b = true
	default:
		err = fmt.Errorf("could not decode invalid bool")
	}
	dstv.Set(reflect.ValueOf(b))
	return
}

// decodeUint will decode unsigned integer
func (ds *decodeState) decodeUint(dstv reflect.Value) (err error) {
	b, err := ds.ReadByte()
	if err != nil {
		return
	}

	in := dstv.Interface()
	temp := reflect.New(reflect.TypeOf(in))
	// check mode of encoding, stored at 2 least significant bits
	mode := b & 3
	switch {
	case mode <= 2:
		var val int64
		val, err = ds.decodeSmallInt(b, mode)
		if err != nil {
			return
		}
		temp.Elem().Set(reflect.ValueOf(val).Convert(reflect.TypeOf(in)))
		dstv.Set(temp.Elem())
	default:
		// >4 byte mode
		topSixBits := b >> 2
		byteLen := uint(topSixBits) + 4

		buf := make([]byte, byteLen)
		_, err = ds.Read(buf)
		if err != nil {
			return
		}

		var o uint64
		if byteLen == 4 {
			o = uint64(binary.LittleEndian.Uint32(buf))
		} else if byteLen > 4 && byteLen <= 8 {
			tmp := make([]byte, 8)
			copy(tmp, buf)
			o = binary.LittleEndian.Uint64(tmp)
		} else {
			err = errors.New("could not decode invalid integer")
			return
		}
		dstv.Set(reflect.ValueOf(o).Convert(reflect.TypeOf(in)))
	}
	return
}

// decodeLength is helper method which calls decodeUint and casts to int
func (ds *decodeState) decodeLength() (l int, err error) {
	dstv := reflect.New(reflect.TypeOf(l))
	err = ds.decodeUint(dstv.Elem())
	if err != nil {
		return
	}
	l = dstv.Elem().Interface().(int)
	return
}

// decodeBytes is used to decode with a destination of []byte or string type
func (ds *decodeState) decodeBytes(dstv reflect.Value) (err error) {
	length, err := ds.decodeLength()
	if err != nil {
		return
	}

	b := make([]byte, length)
	_, err = ds.Read(b)
	if err != nil {
		return
	}

	in := dstv.Interface()
	inType := reflect.TypeOf(in)
	dstv.Set(reflect.ValueOf(b).Convert(inType))
	return
}

// decodeSmallInt is used in the decodeUint and decodeBigInt functions when the mode is <= 2
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
func (ds *decodeState) decodeBigInt(dstv reflect.Value) (err error) {
	b, err := ds.ReadByte()
	if err != nil {
		return
	}

	var output *big.Int
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
		if err != nil {
			err = fmt.Errorf("could not decode invalid big.Int: %v", err)
			break
		}
		o := reverseBytes(buf)
		output = big.NewInt(0).SetBytes(o)
	}
	dstv.Set(reflect.ValueOf(output))
	return
}

// decodeFixedWidthInt decodes integers < 2**32 by reading the bytes in little endian
func (ds *decodeState) decodeFixedWidthInt(dstv reflect.Value) (err error) {
	in := dstv.Interface()
	var out interface{}
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
	default:
		err = fmt.Errorf("invalid type: %T", in)
		return
	}
	dstv.Set(reflect.ValueOf(out))
	return
}

// decodeUint128 accepts a byte array representing Scale encoded common.Uint128 and performs SCALE decoding of the Uint128
func (ds *decodeState) decodeUint128(dstv reflect.Value) (err error) {
	buf := make([]byte, 16)
	err = binary.Read(ds, binary.LittleEndian, buf)
	if err != nil {
		return
	}
	ui128, err := NewUint128(buf)
	if err != nil {
		return
	}
	dstv.Set(reflect.ValueOf(ui128))
	return
}
