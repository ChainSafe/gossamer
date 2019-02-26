package codec

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"
)

type reverseByteTest struct {
	val    []byte
	output []byte
}

type decodeIntTest struct {
	val          []byte
	output       int64
	bytesDecoded int
}

type decodeBigIntTest struct {
	val          []byte
	output       *big.Int
	bytesDecoded int
}

type decodeByteArrayTest struct {
	val          []byte
	output       []byte
	bytesDecoded int64
}

type decodeBoolTest struct {
	val    byte
	output bool
}

type decodeTupleTest struct {
	val    []byte
	t      interface{}
	output interface{}
}

var decodeIntTests = []decodeIntTest{
	// compact integers
	{val: []byte{0x00}, output: int64(0), bytesDecoded: 1},
	{val: []byte{0x04}, output: int64(1), bytesDecoded: 1},
	{val: []byte{0xa8}, output: int64(42), bytesDecoded: 1},
	{val: []byte{0x01, 0x01}, output: int64(64), bytesDecoded: 2},
	{val: []byte{0x15, 0x01}, output: int64(69), bytesDecoded: 2},
	{val: []byte{0xfd, 0xff}, output: int64(16383), bytesDecoded: 2},
	{val: []byte{0x02, 0x00, 0x01, 0x00}, output: int64(16384), bytesDecoded: 4},
	{val: []byte{0xfe, 0xff, 0xff, 0xff}, output: int64(1073741823), bytesDecoded: 4},
	{val: []byte{0x03, 0x00, 0x00, 0x00, 0x40}, output: int64(1073741824), bytesDecoded: 5},
	{val: []byte{0x03, 0xff, 0xff, 0xff, 0xff}, output: int64(1<<32 - 1), bytesDecoded: 5},
	{val: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01}, output: int64(1 << 32), bytesDecoded: 6},
}

var decodeBigIntTests = []decodeBigIntTest{
	// compact integers
	{val: []byte{0x00}, output: big.NewInt(0), bytesDecoded: 1},
	{val: []byte{0x04}, output: big.NewInt(1), bytesDecoded: 1},
	{val: []byte{0xa8}, output: big.NewInt(42), bytesDecoded: 1},
	{val: []byte{0x01, 0x01}, output: big.NewInt(64), bytesDecoded: 2},
	{val: []byte{0x15, 0x01}, output: big.NewInt(69), bytesDecoded: 2},
	{val: []byte{0xfd, 0xff}, output: big.NewInt(16383), bytesDecoded: 2},
	{val: []byte{0x02, 0x00, 0x01, 0x00}, output: big.NewInt(16384), bytesDecoded: 4},
	{val: []byte{0xfe, 0xff, 0xff, 0xff}, output: big.NewInt(1073741823), bytesDecoded: 4},
	{val: []byte{0x03, 0x00, 0x00, 0x00, 0x40}, output: big.NewInt(1073741824), bytesDecoded: 5},
	{val: []byte{0x03, 0xff, 0xff, 0xff, 0xff}, output: big.NewInt(1<<32 - 1), bytesDecoded: 5},
	{val: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01}, output: big.NewInt(1 << 32), bytesDecoded: 6},
}

var decodeByteArrayTests = []decodeByteArrayTest{
	// byte arrays
	{val: []byte{0x04, 0x01}, output: []byte{0x01}, bytesDecoded: 2},
	{val: []byte{0x04, 0xff}, output: []byte{0xff}, bytesDecoded: 2},
	{val: []byte{0x08, 0x01, 0x01}, output: []byte{0x01, 0x01}, bytesDecoded: 3},
	{val: append([]byte{0x01, 0x01}, byteArray(64)...), output: byteArray(64), bytesDecoded: 66},
	{val: append([]byte{0xfd, 0xff}, byteArray(16383)...), output: byteArray(16383), bytesDecoded: 16385},
	{val: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...), output: byteArray(16384), bytesDecoded: 16388},
	{val: append([]byte{0xfe, 0xff, 0xff, 0xff}, byteArray(1073741823)...), output: byteArray(1073741823), bytesDecoded: 1073741827},
	// Causes CI to crash
	//{val: append([]byte{0x03, 0x00, 0x00, 0x00, 0x40}, byteArray(1073741824)...), output: byteArray(1073741824), bytesDecoded: 1073741829},
}

var decodeBoolTests = []decodeBoolTest{
	{val: 0x01, output: true},
	{val: 0x00, output: false},
}

var decodeTupleTests = []decodeTupleTest{
	{val: []byte{0x04, 0x01, 0x08}, t: &struct {
		Foo []byte
		Bar int64
	}{}, output: &struct {
		Foo []byte
		Bar int64
	}{[]byte{0x01}, 2}},

	{val: []byte{0x04, 0x01, 0x03, 0xff, 0xff, 0xff, 0xff}, t: &struct {
		Foo []byte
		Bar int64
	}{}, output: &struct {
		Foo []byte
		Bar int64
	}{[]byte{0x01}, int64(1<<32 - 1)}},

	{val: append([]byte{0x04, 0x01, 0x02, 0x00, 0x01, 0x00}, byteArray(16384)...), t: &struct {
		Foo []byte
		Bar []byte
	}{}, output: &struct {
		Foo []byte
		Bar []byte
	}{[]byte{0x01}, byteArray(16384)}},

	{val: []byte{0x04, 0x01, 0xfd, 0xff, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01}, t: &struct {
		Foo  []byte
		Bar  int64
		Noot int64
	}{}, output: &struct {
		Foo  []byte
		Bar  int64
		Noot int64
	}{[]byte{0x01}, 16383, int64(1 << 32)}},

	{val: []byte{0x04, 0x01, 0xfd, 0xff, 0x01, 0x07, 0x00, 0x00, 0x00, 0x00, 0x01}, t: &struct {
		Foo  []byte
		Bar  int64
		Bo   bool
		Noot int64
	}{}, output: &struct {
		Foo  []byte
		Bar  int64
		Bo   bool
		Noot int64
	}{[]byte{0x01}, 16383, true, int64(1 << 32)}},
}

var reverseByteTests = []reverseByteTest{
	{val: []byte{0x00, 0x01, 0x02}, output: []byte{0x02, 0x01, 0x00}},
	{val: []byte{0x04, 0x05, 0x06, 0x07}, output: []byte{0x07, 0x06, 0x05, 0x04}},
	{val: []byte{0xff}, output: []byte{0xff}},
}

func TestReverseBytes(t *testing.T) {
	for _, test := range reverseByteTests {
		output := reverseBytes(test.val)
		if !bytes.Equal(output, test.output) {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		}
	}
}

func TestReadByte(t *testing.T) {
	buf := bytes.Buffer{}
	sd := Decoder{&buf}
	buf.Write([]byte{0xff})
	output, err := sd.ReadByte()
	if err != nil {
		t.Error(err)
	} else if output != 0xff {
		t.Errorf("Fail: got %x expected %x", output, 0xff)
	}
}

func TestDecodeInts(t *testing.T) {
	for _, test := range decodeIntTests {
		buf := bytes.Buffer{}
		sd := Decoder{&buf}
		buf.Write(test.val)
		var i int64
		output, err := sd.Decode(i)
		if err != nil {
			t.Error(err)
		} else if output != test.output {
			t.Errorf("Fail: input %d got %d expected %d", test.val, output, test.output)
		}
	}
}

func TestDecodeBigInts(t *testing.T) {
	for _, test := range decodeBigIntTests {
		buf := bytes.Buffer{}
		sd := Decoder{&buf}
		buf.Write(test.val)
		var tmp *big.Int
		output, err := sd.Decode(tmp)
		if err != nil {
			t.Error(err)
		} else if output.(*big.Int).Cmp(test.output) != 0 {
			t.Errorf("Fail: got %s expected %s", output.(*big.Int).String(), test.output.String())
		}
	}
}

func TestDecodeByteArrays(t *testing.T) {
	for _, test := range decodeByteArrayTests {
		buf := bytes.Buffer{}
		sd := Decoder{&buf}
		buf.Write(test.val)
		var tmp []byte
		output, err := sd.Decode(tmp)
		if err != nil {
			t.Error(err)
		} else if !bytes.Equal(output.([]byte), test.output) {
			t.Errorf("Fail: got %d expected %d", len(output.([]byte)), len(test.output))
		}
	}
}

func TestDecodeBool(t *testing.T) {
	for _, test := range decodeBoolTests {
		buf := bytes.Buffer{}
		sd := Decoder{&buf}
		buf.Write([]byte{test.val})
		var tmp bool
		output, err := sd.Decode(tmp)
		if err != nil {
			t.Error(err)
		} else if output != test.output {
			t.Errorf("Fail: got %t expected %t", output, test.output)
		}
	}

	buf := bytes.Buffer{}
	sd := Decoder{&buf}
	buf.Write([]byte{0xff})
	output, err := sd.DecodeBool()
	if err == nil {
		t.Error("did not error for invalid bool")
	} else if output {
		t.Errorf("Fail: got %t expected false", output)
	}
}

func TestDecodeTuples(t *testing.T) {
	for _, test := range decodeTupleTests {
		buf := bytes.Buffer{}
		buf.Write(test.val)
		sd := Decoder{&buf}
		output, err := sd.Decode(test.t)
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(output,test.output) {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		}
	}
}
