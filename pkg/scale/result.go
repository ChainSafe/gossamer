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

type resultCache map[string]map[bool]interface{}

// Result encapsulates an Ok or an Err case. It's not a valid result unless one of the
// attributes != nil
type Result struct {
	ok  *interface{}
	err *interface{}
}

func (r *Result) SetOk(in interface{}) (err error) {
	r.ok = &in
	return
}

func (r *Result) SetErr(in interface{}) (err error) {
	r.err = &in
	return
}

// func (r *Result) Ok() (okIn interface{}, err error) {
// 	return
// }

// func (r *Result) Err() (errIn interface{}, err error) {
// 	return
// }

type ResultErr struct {
	Err interface{}
}

func (r ResultErr) Error() string {
	return fmt.Sprintf("ResultErr %+v", r.Err)
}

// Result returns the result in go standard wrapping the Err case in a ResultErr
func (r *Result) Result() (in interface{}, err error) {
	if !r.IsValid() {
		err = fmt.Errorf("result is not valid")
		return
	}
	if r.ok != nil {
		in = *r.ok
	} else {
		in = *r.err
		err = ResultErr{*r.err}
	}
	return
}

// Valid returns whether the Result is valid.  Only one of the Ok and Err attributes should be nil
func (r *Result) IsValid() bool {
	return (r.ok == nil && r.err != nil) || (r.ok != nil && r.err == nil)
}

var resCache resultCache = make(resultCache)

func RegisterResult(in interface{}, inOK interface{}, inErr interface{}) (err error) {
	t := reflect.TypeOf(in)
	field, ok := t.FieldByName("Result")
	if !ok {
		err = fmt.Errorf("yao")
		return
	}

	if !field.Type.ConvertibleTo(reflect.TypeOf(Result{})) {
		err = fmt.Errorf("%T is not a Result", in)
		return
	}

	key := fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	_, ok = resCache[key]
	if !ok {
		resCache[key] = make(map[bool]interface{})
	}
	resCache[key][true] = inOK
	resCache[key][false] = inErr
	return
}
