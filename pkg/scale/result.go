// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
	"reflect"
)

// ResultMode is the mode the Result is set to
type ResultMode int

const (
	// Unset ResultMode is zero value mode
	Unset ResultMode = iota
	// OK case
	OK
	// Err case
	Err
)

// Result encapsulates an Ok or an Err case
type Result struct {
	ok   interface{}
	err  interface{}
	mode ResultMode
}

// NewResult is constructor for Result. Use nil to represent empty tuple () in Rust.
func NewResult(okIn, errIn interface{}) (res Result) {
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

// Set takes in a mode (OK/Err) and the associated interface and sets the Result value
func (r *Result) Set(mode ResultMode, in interface{}) (err error) {
	switch mode {
	case OK:
		if reflect.TypeOf(r.ok) == reflect.TypeOf(empty{}) && in == nil {
			r.mode = mode
			return
		} else if reflect.TypeOf(r.ok) != reflect.TypeOf(in) {
			err = fmt.Errorf("type mistmatch for result.ok: %T, and inputted: %T", r.ok, in)
			return
		}
		r.ok = in
		r.mode = mode
	case Err:
		if reflect.TypeOf(r.err) == reflect.TypeOf(empty{}) && in == nil {
			r.mode = mode
			return
		} else if reflect.TypeOf(r.err) != reflect.TypeOf(in) {
			err = fmt.Errorf("type mistmatch for result.err: %T, and inputted: %T", r.ok, in)
			return
		}
		r.err = in
		r.mode = mode
	default:
		err = fmt.Errorf("invalid ResultMode %v", mode)
	}
	return
}

// UnsetResult is error when Result is unset with a value.
type UnsetResult error

// Unwrap returns the result in go standard wrapping the Err case in a ResultErr
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

// WrappedErr is returned by Result.Unwrap().  The underlying Err value is wrapped and stored in Err attribute
type WrappedErr struct {
	Err interface{}
}

// Error fulfils the error interface
func (r WrappedErr) Error() string {
	return fmt.Sprintf("ResultErr %+v", r.Err)
}
