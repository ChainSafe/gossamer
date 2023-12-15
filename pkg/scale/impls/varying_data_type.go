package impls

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupportedVaryingDataTypeValue = errors.New("unsupported VaryingDataTypeValue")
	ErrVaryingDataTypeNotSet           = errors.New("varying data type not set")
	ErrMustProvideVaryingDataTypeValue = errors.New("must provide at least one VaryingDataTypeValue")
)

type VaryingDataTypeValue interface {
	Index() uint
}

// VaryingDataType is analogous to a rust enum.  Name is taken from polkadot spec.
type VaryingDataType struct {
	value VaryingDataTypeValue
	cache map[uint]VaryingDataTypeValue
}

// var _ = scale.VaryingDataType(&VaryingDataType{})

// Set will set the VaryingDataType value
func (vdt *VaryingDataType) SetValue(value any) (err error) {
	vdtv, ok := value.(VaryingDataTypeValue)
	if !ok {
		err = fmt.Errorf("%w: %v (%T)", ErrUnsupportedVaryingDataTypeValue, value, value)
		return
	}
	_, ok = vdt.cache[vdtv.Index()]
	if !ok {
		err = fmt.Errorf("%w: %v (%T)", ErrUnsupportedVaryingDataTypeValue, value, value)
		return
	}
	vdt.value = vdtv
	return
}

// ValueAt returns VaryingDataTypeValue with the matching index
func (vdt *VaryingDataType) ValueAt(index uint) (value any, err error) {
	val, ok := vdt.cache[index]
	if !ok {
		return nil, ErrUnsupportedVaryingDataTypeValue
	}
	return any(val), nil
}

// Value returns value stored in vdt
func (vdt *VaryingDataType) IndexValue() (uint, any, error) {
	if vdt.value == nil {
		return 0, nil, ErrVaryingDataTypeNotSet
	}
	return vdt.value.Index(), vdt.value, nil
}

// Value returns value stored in vdt
func (vdt *VaryingDataType) Value() (any, error) {
	_, value, err := vdt.IndexValue()
	return value, err
}

func (vdt *VaryingDataType) MustValue() any {
	value, err := vdt.Value()
	if err != nil {
		panic(err)
	}
	return value
}

func (vdt *VaryingDataType) String() string {
	if vdt.value == nil {
		return "VaryingDataType(nil)"
	}
	stringer, ok := vdt.value.(fmt.Stringer)
	if !ok {
		return fmt.Sprintf("VaryingDataType(%v)", vdt.value)
	}
	return stringer.String()
}

// NewVaryingDataType is constructor for VaryingDataType
func NewVaryingDataType(values ...VaryingDataTypeValue) (vdt VaryingDataType, err error) {
	if len(values) == 0 {
		err = fmt.Errorf("%w", ErrMustProvideVaryingDataTypeValue)
		return
	}
	vdt.cache = make(map[uint]VaryingDataTypeValue)
	for _, value := range values {
		_, ok := vdt.cache[value.Index()]
		if ok {
			err = fmt.Errorf("duplicate index with VaryingDataType: %T with index: %d", value, value.Index())
			return
		}
		vdt.cache[value.Index()] = value
	}
	return
}

// MustNewVaryingDataType is constructor for VaryingDataType
func MustNewVaryingDataType(values ...VaryingDataTypeValue) (vdt VaryingDataType) {
	vdt, err := NewVaryingDataType(values...)
	if err != nil {
		panic(err)
	}
	return
}
