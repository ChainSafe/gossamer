# go-scale Codec

Go implementation of the SCALE (Simple Concatenated Aggregate Little-Endian) data format for types used in the Parity Substrate framework.

SCALE is a light-weight format which allows encoding (and decoding) which makes it highly suitable for resource-constrained execution environments like blockchain runtimes and low-power, low-memory devices.

It is important to note that the encoding context (knowledge of how the types and data structures look) needs to be known separately at both encoding and decoding ends. The encoded data does not include this contextual information.

This codec attempts to translate the primitive Go types to the associated types in SCALE.  It also introduces a few custom types to implement SCALE primitives that have no direct translation to a Go primmitive type.

## Translating From SCALE to Go

Goavro does not use Go's structure tags to translate data between
native Go types and Avro encoded data.

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

When decoding SCALE data knowledge of the structure of the destination data type is required to decode.  Structs are encoded as a SCALE Tuple, where each struct field is encoded in the sequence of the fields.  

#### Struct Tags

go-scale uses `scale` struct tag do modify the order of the field values during encoding.  This is also used when decoding attributes back to the original type.  This essentially allows you to modify struct field ordering but preserve the encoding/decoding functionality.

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
| `Compact<u16>`      | `*uint16`               |
| `Compact<u32>`      | `*uint32`               |
| `Compact<u64>`      | `*uint64`               |
| `Compact<u128>`     | `*big.Int`              |

### Result

To be implemented and documented.

## Usage

### Basic Example

Basic example which encodes and decodes a `uint`.
```
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func basicExample() {
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

func structExample() {
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

### Varying Data Type

A VaryingDataType is analogous to a Rust enum.  A VaryingDataType needs to be registered using the  `RegisterVaryingDataType` function with it's associated `VaryingDataTypeValue` types.  `VaryingDataTypeValue` is an
interface with one `Index() uint` method that needs to be implemented.  The returned `uint` index should be unique per type and needs to be the same index as defined in the Rust enum.

```
import (
	"fmt"
	"testing"

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

type MyVaryingDataType scale.VaryingDataType

func varyingDataTypeExample() {
	err := scale.RegisterVaryingDataType(MyVaryingDataType{}, MyStruct{}, MyOtherStruct{}, MyInt16(0))
	if err != nil {
		panic(err)
	}

	mvdt := MyVaryingDataType{
		MyStruct{
			Baz: true,
			Bar: 999,
			Foo: []byte{1, 2},
		},
		MyOtherStruct{
			Foo: "hello",
			Bar: 999,
			Baz: 888,
		},
		MyInt16(111),
	}
	bytes, err := scale.Marshal(mvdt)
	if err != nil {
		panic(err)
	}

	var unmarshaled MyVaryingDataType
	err = scale.Unmarshal(bytes, &unmarshaled)
	if err != nil {
		panic(err)
	}

	// [{Baz:true Bar:999 Foo:[1 2]} {Foo:hello Bar:999 Baz:888} 111]
	fmt.Printf("%+v", unmarshaled)
}
```