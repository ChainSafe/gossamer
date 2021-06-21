# go-scale Codec

Go implementation of the SCALE (Simple Concatenated Aggregate Little-Endian) data format for types used in the Parity Substrate framework.

SCALE is a light-weight format which allows encoding (and decoding) which makes it highly suitable for resource-constrained execution environments like blockchain runtimes and low-power, low-memory devices.

It is important to note that the encoding context (knowledge of how the types and data structures look) needs to be known separately at both encoding and decoding ends. The encoded data does not include this contextual information.

This codec attempts to translate the primitive Go types to the associated types in SCALE.  It also introduces a few custom types to implement Rust primitives that have no direct translation to a Go primitive type.

## Translating From SCALE to Go

When translating from SCALE to native Go data,
go-scale returns primitive Go data values for corresponding SCALE data
values. The table below shows how go-scale translates SCALE types to Go.

### Primitives

| SCALE/Rust         | Go                       |
| ------------------ | ------------------------ |
| `i8`               | `int8`                   |
| `u8`               | `uint8`                  |
| `i16`              | `int16`                  |
| `u16`              | `uint16`                 |
| `i32`              | `int32`                  |
| `u32`              | `uint32`                 |
| `i64`              | `int64`                  |
| `u64`              | `uint64`                 |
| `i128`             | `*big.Int`               |
| `u128`             | `*scale.Uint128`         |
| `bytes`            | `[]byte`                 |
| `string`           | `string`                 |
| `enum`             | `scale.VaryingDataType`  |
| `struct`           | `struct`                 |

### Structs

When decoding SCALE data, knowledge of the structure of the destination data type is required to decode.  Structs are encoded as a SCALE Tuple, where each struct field is encoded in the sequence of the fields.  

#### Struct Tags

go-scale uses a `scale` struct tag to modify the order of the field values during encoding.  This is also used when decoding attributes back to the original type.  This essentially allows you to modify struct field ordering but preserve the encoding/decoding ordering.

See the [usage example](#Struct-Tag-Example).

### Option

For all `Option<T>` a pointer to the underlying type is used in go-scale. In the `None` case the pointer value is `nil`.

| SCALE/Rust         | Go                       |
| ------------------ | ------------------------ |
| `Option<i8>`       | `*int8`                  |
| `Option<u8>`       | `*uint8`                 |
| `Option<i16>`      | `*int16`                 |
| `Option<u16>`      | `*uint16`                |
| `Option<i32>`      | `*int32`                 |
| `Option<u32>`      | `*uint32`                |
| `Option<i64>`      | `*int64`                 |
| `Option<u64>`      | `*uint64`                |
| `Option<i128>`     | `**big.Int`              |
| `Option<u128>`     | `**scale.Uint128`        |
| `Option<bytes>`    | `*[]byte`                |
| `Option<string>`   | `*string`                |
| `Option<enum>`     | `*scale.VaryingDataType` |
| `Option<struct>`   | `*struct`                |
| `None`             | `nil`                    |

### Compact Encoding

SCALE uses a compact encoding for variable width unsigned integers.

| SCALE/Rust         | Go                       |
| ------------------ | ------------------------ |
| `Compact<u8>`       | `uint`                  |
| `Compact<u16>`      | `uint`                  |
| `Compact<u32>`      | `uint`                  |
| `Compact<u64>`      | `uint`                  |
| `Compact<u128>`     | `*big.Int`              |

## Usage

### Basic Example

Basic example which encodes and decodes a `uint`.
```
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func ExampleBasic() {
	// compact length encoded uint
	var ui uint = 999
	bytes, err := scale.Marshal(ui)
	if err != nil {
		panic(err)
	}

	var unmarshaled uint
	err = scale.Unmarshal(bytes, &unmarshaled)
	if err != nil {
		panic(err)
	}

	// 999
	fmt.Printf("%d", unmarshaled)
}
```

### Struct Tag Example

Use the `scale` struct tag for struct fields to conform to specific encoding sequence of struct field values.  A struct tag of `"-"` will be omitted from encoding and decoding.

```
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func ExampleStruct() {
	type MyStruct struct {
		Baz bool      `scale:"3"`
		Bar int32     `scale:"2"`
		Foo []byte    `scale:"1"`
		Ignored int64 `scale:"-"`
	}
	var ms = MyStruct{
		Baz: true,
		Bar: 999,
		Foo: []byte{1, 2},
		Ignored: 999
	}
	bytes, err := scale.Marshal(ms)
	if err != nil {
		panic(err)
	}

	var unmarshaled MyStruct
	err = scale.Unmarshal(bytes, &unmarshaled)
	if err != nil {
		panic(err)
	}

	// {Baz:true Bar:999 Foo:[1 2] Ignored:0}
	fmt.Printf("%+v", unmarshaled)
}
```

### Result

A `Result` is custom type analogous to a rust result.  A `Result` needs to be constructed using the `NewResult` constructor.  The two parameters accepted are the expected types that are associated to the `Ok`, and `Err` cases.  

```
// Rust
Result<i32, i32> = Ok(10)

// go-scale
result := scale.NewResult(int32(0), int32(0)
result.Set(scale.Ok, 10)
```

```
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func ExampleResult() {
	// pass in zero or non-zero values of the types for Ok and Err cases
	res := scale.NewResult(bool(false), string(""))

	// set the OK case with a value of true, any values for OK that are not bool will return an error
	err := res.Set(scale.OK, true)
	if err != nil {
		panic(err)
	}

	bytes, err := scale.Marshal(res)
	if err != nil {
		panic(err)
	}

	// [0x00, 0x01]
	fmt.Printf("%v\n", bytes)

	res1 := scale.NewResult(bool(false), string(""))

	err = scale.Unmarshal(bytes, &res1)
	if err != nil {
		panic(err)
	}

	// res1 should be Set with OK mode and value of true
	ok, err := res1.Unwrap()
	if err != nil {
		panic(err)
	}

	switch ok := ok.(type) {
	case bool:
		if !ok {
			panic(fmt.Errorf("unexpected ok value: %v", ok))
		}
	default:
		panic(fmt.Errorf("unexpected type: %T", ok))
	}
}

```

### Varying Data Type

A `VaryingDataType` is analogous to a Rust enum.  A `VaryingDataType` needs to be constructed using the  `NewVaryingDataType` constructor.  `VaryingDataTypeValue` is an
interface with one `Index() uint` method that needs to be implemented.  The returned `uint` index should be unique per type and needs to be the same index as defined in the Rust enum to ensure interopability.  To set the value of the `VaryingDataType`, the `VaryingDataType.Set()` function should be called with an associated `VaryingDataTypeValue`.

```
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type MyStruct struct {
	Baz bool
	Bar uint32
	Foo []byte
}

func (ms MyStruct) Index() uint {
	return 1
}

type MyOtherStruct struct {
	Foo string
	Bar uint64
	Baz uint
}

func (mos MyOtherStruct) Index() uint {
	return 2
}

type MyInt16 int16

func (mi16 MyInt16) Index() uint {
	return 3
}

func ExampleVaryingDataType() {
	vdt, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
	if err != nil {
		panic(err)
	}

	err = vdt.Set(MyStruct{
		Baz: true,
		Bar: 999,
		Foo: []byte{1, 2},
	})
	if err != nil {
		panic(err)
	}

	bytes, err := scale.Marshal(vdt)
	if err != nil {
		panic(err)
	}

	vdt1, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
	if err != nil {
		panic(err)
	}

	err = scale.Unmarshal(bytes, &vdt1)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(vdt, vdt1) {
		panic(fmt.Errorf("uh oh: %+v %+v", vdt, vdt1))
	}
}
```

A `VaryingDataTypeSlice` is a slice containing multiple `VaryingDataType` elements.  Each `VaryingDataTypeValue` must be of a supported type of the `VaryingDataType` passed into the `NewVaryingDataTypeSlice` constructor.  The method to call to add `VaryingDataTypeValue` instances is `VaryingDataTypeSlice.Add()`.

```
func ExampleVaryingDataTypeSlice() {
	vdt, err := scale.NewVaryingDataType(MyStruct{}, MyOtherStruct{}, MyInt16(0))
	if err != nil {
		panic(err)
	}

	vdts := scale.NewVaryingDataTypeSlice(vdt)

	err = vdts.Add(
		MyStruct{
			Baz: true,
			Bar: 999,
			Foo: []byte{1, 2},
		},
		MyInt16(1),
	)
	if err != nil {
		panic(err)
	}

	bytes, err := scale.Marshal(vdts)
	if err != nil {
		panic(err)
	}

	vdts1 := scale.NewVaryingDataTypeSlice(vdt)
	if err != nil {
		panic(err)
	}

	err = scale.Unmarshal(bytes, &vdts1)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(vdts, vdts1) {
		panic(fmt.Errorf("uh oh: %+v %+v", vdts, vdts1))
	}
}
```