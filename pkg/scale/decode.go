package scale

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

func Unmarshal(data []byte, dst interface{}) (err error) {
	buf := &bytes.Buffer{}
	ds := decodeState{}
	_, err = buf.Write(data)
	if err != nil {
		return
	}
	ds.Buffer = *buf
	err = ds.unmarshal(dst)
	return
}

type decodeState struct {
	bytes.Buffer
}

func (ds *decodeState) unmarshal(dst interface{}) (err error) {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		err = fmt.Errorf("unsupported dst: %T", dst)
	}

	in := rv.Elem().Interface()
	switch in.(type) {
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
		in, err = ds.decodeFixedWidthInt(in)
	}
	if err != nil {
		return
	}
	rv.Elem().Set(reflect.ValueOf(in))
	return
}

// DdcodeFixedWidthInt decodes integers < 2**32 by reading the bytes in little endian
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
