package scale

import (
	"fmt"
	"reflect"
)

type varyingDataTypeCache map[string]map[uint]VaryingDataTypeValue

var vdtCache varyingDataTypeCache = make(varyingDataTypeCache)

type VaryingDataType []VaryingDataTypeValue

func RegisterVaryingDataType(in interface{}, values ...VaryingDataTypeValue) (err error) {
	t := reflect.TypeOf(in)
	if !t.ConvertibleTo(reflect.TypeOf(VaryingDataType{})) {
		err = fmt.Errorf("%T is not a VaryingDataType", in)
		return
	}

	key := fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	_, ok := vdtCache[key]
	if !ok {
		vdtCache[key] = make(map[uint]VaryingDataTypeValue)
	}
	for _, val := range values {
		vdtCache[key][val.Index()] = val
	}
	return
}

// VaryingDataType is used to represent scale encodable types
type VaryingDataTypeValue interface {
	Index() uint
}
