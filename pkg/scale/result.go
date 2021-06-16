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

type ResultMode int

const (
	Unset ResultMode = iota
	OK
	Err
)

// Result encapsulates an Ok or an Err case
type Result struct {
	ok   interface{}
	err  interface{}
	mode ResultMode
}

// NewResult is constructor for Result. Use nil to represent empty tuple () in Rust.
func NewResult(okIn interface{}, errIn interface{}) (res Result) {
	switch okIn {
	case nil:
		res.ok = empty{}
	default:
		res.ok = okIn
	}
	switch errIn {
	case nil:
		res.err = empty{}
	default:
		res.err = errIn
	}
	return
}

func (r *Result) Set(mode ResultMode, in interface{}) (err error) {
	switch mode {
	case OK:
		if reflect.TypeOf(r.ok) != reflect.TypeOf(in) {
			err = fmt.Errorf("type mistmatch for result.ok: %T, and inputted: %T", r.ok, in)
			return
		}
		r.ok = in
		r.mode = mode
	case Err:
		if reflect.TypeOf(r.err) != reflect.TypeOf(in) {
			err = fmt.Errorf("type mistmatch for result.ok: %T, and inputted: %T", r.ok, in)
			return
		}
		r.err = in
		r.mode = mode
	default:
		err = fmt.Errorf("invalid ResultMode %v", mode)
	}
	return
}

type UnsetResult error

// Result returns the result in go standard wrapping the Err case in a ResultErr
func (r *Result) Unwrap() (ok interface{}, err error) {
	if !r.IsSet() {
		err = UnsetResult(fmt.Errorf("result is not set"))
		return
	}
	switch r.mode {
	case OK:
		switch r.ok.(type) {
		case empty:
			ok = nil
		default:
			ok = r.ok
		}
	case Err:
		switch r.err.(type) {
		case empty:
			err = WrappedErr{nil}
		default:
			err = WrappedErr{r.err}
		}
	}
	return
}

// IsSet returns whether the Result is set with an Ok or Err value.
func (r *Result) IsSet() bool {
	if r.ok == nil || r.err == nil {
		return false
	}
	switch r.mode {
	case OK, Err:
	default:
		return false
	}
	return true
}

type empty struct{}

type WrappedErr struct {
	Err interface{}
}

func (r WrappedErr) Error() string {
	return fmt.Sprintf("ResultErr %+v", r.Err)
}
