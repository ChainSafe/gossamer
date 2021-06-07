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
	"sort"
	"strings"
	"sync"
)

// package level cache for fieldScaleIndicies
var cache = &fieldScaleIndicesCache{
	cache: make(map[string]fieldScaleIndices),
}

// fieldScaleIndex is used to map field index to scale index
type fieldScaleIndex struct {
	fieldIndex int
	scaleIndex *string
}
type fieldScaleIndices []fieldScaleIndex

// fieldScaleIndicesCache stores the order of the fields per struct
type fieldScaleIndicesCache struct {
	cache map[string]fieldScaleIndices
	sync.RWMutex
}

func (fsic *fieldScaleIndicesCache) fieldScaleIndices(in interface{}) (v reflect.Value, indices fieldScaleIndices, err error) {
	t := reflect.TypeOf(in)
	v = reflect.ValueOf(in)
	key := fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	if key != "." {
		var ok bool
		fsic.RLock()
		indices, ok = fsic.cache[key]
		fsic.RUnlock()
		if ok {
			return
		}
	}

	if !v.IsValid() {
		err = fmt.Errorf("inputted value is not valid: %v", v)
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("scale")
		switch strings.TrimSpace(tag) {
		case "":
			indices = append(indices, fieldScaleIndex{
				fieldIndex: i,
			})
		case "-":
			// ignore this field
			continue
		default:
			indices = append(indices, fieldScaleIndex{
				fieldIndex: i,
				scaleIndex: &tag,
			})
		}
	}

	sort.Slice(indices[:], func(i, j int) bool {
		switch {
		case indices[i].scaleIndex == nil && indices[j].scaleIndex != nil:
			return false
		case indices[i].scaleIndex != nil && indices[j].scaleIndex == nil:
			return true
		case indices[i].scaleIndex == nil && indices[j].scaleIndex == nil:
			return indices[i].fieldIndex < indices[j].fieldIndex
		case indices[i].scaleIndex != nil && indices[j].scaleIndex != nil:
			return *indices[i].scaleIndex < *indices[j].scaleIndex
		}
		return false
	})

	if key != "." {
		fsic.Lock()
		fsic.cache[key] = indices
		fsic.Unlock()
	}
	return
}

func reverseBytes(a []byte) []byte {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
	return a
}
