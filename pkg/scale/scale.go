// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
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
	scaleIndex *int
}
type fieldScaleIndices []fieldScaleIndex

// fieldScaleIndicesCache stores the order of the fields per struct
type fieldScaleIndicesCache struct {
	cache map[string]fieldScaleIndices
	sync.RWMutex
}

func (fsic *fieldScaleIndicesCache) fieldScaleIndices(in interface{}) (
	v reflect.Value, indices fieldScaleIndices, err error) {
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
			scaleIndex, indexErr := strconv.Atoi(tag)
			if indexErr != nil {
				err = fmt.Errorf("%w: %v", ErrInvalidScaleIndex, indexErr)
				return
			}
			indices = append(indices, fieldScaleIndex{
				fieldIndex: i,
				scaleIndex: &scaleIndex,
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
