// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package externalities

import "github.com/tidwall/btree"

type Extension interface {
	TypeId() string // reflect.TypeOf(v).String()
}

type Extensions struct {
	extensions btree.Map[string, Extension]
}

func NewExtensions() Extensions {
	return Extensions{
		extensions: btree.Map[string, Extension]{},
	}
}

type Externalities interface {
	// TODO: add methods, will be addressed in #4465
}
