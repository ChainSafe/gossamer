// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
