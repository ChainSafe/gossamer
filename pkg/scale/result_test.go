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
	"reflect"
	"testing"
)

func TestEncodeDecodeResult(t *testing.T) {
	ms := MyStruct{
		Foo: []byte{0x01},
		Bar: 2,
		Baz: true,
	}
	res := NewResult(ms, nil)
	res.Set(OK, ms)

	bytes, err := Marshal(res)
	if err != nil {
		t.Errorf("%v", err)
	}

	if !reflect.DeepEqual([]byte{0x00, 0x04, 0x01, 0x02, 0, 0, 0, 0x01}, bytes) {
		t.Errorf("unexpected bytes: %v", bytes)
	}

	res = NewResult(nil, true)
	res.Set(Err, true)
	bytes, err = Marshal(res)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual([]byte{0x01, 0x01}, bytes) {
		t.Errorf("unexpected bytes: %v", bytes)
	}

	res = NewResult(nil, true)
	res.Set(Err, false)
	bytes, err = Marshal(res)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !reflect.DeepEqual([]byte{0x01, 0x00}, bytes) {
		t.Errorf("unexpected bytes: %v", bytes)
	}

	mr2 := NewResult(ms, nil)
	err = Unmarshal([]byte{0x00, 0x04, 0x01, 0x02, 0, 0, 0, 0x01}, &mr2)
	if err != nil {
		t.Errorf("%v", err)
	}
	expected := NewResult(ms, nil)
	expected.Set(OK, ms)
	if !reflect.DeepEqual(expected, mr2) {
		t.Errorf("unexpected MyResult %+v %+v", expected, mr2)
	}

	mr3 := NewResult(nil, true)
	err = Unmarshal([]byte{0x01, 0x01}, &mr3)
	if err != nil {
		t.Errorf("%v", err)
	}
	expected = NewResult(nil, true)
	expected.Set(Err, true)
	if !reflect.DeepEqual(expected, mr3) {
		t.Errorf("unexpected MyResult %+v %+v", expected, mr3)
	}
}

func TestResult_IsSet(t *testing.T) {
	type fields struct {
		ok   interface{}
		err  interface{}
		mode ResultMode
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			want: false,
		},
		{
			fields: fields{
				ok: empty{},
			},
			want: false,
		},
		{
			fields: fields{
				ok:  empty{},
				err: empty{},
			},
			want: false,
		},
		{
			fields: fields{
				ok:   empty{},
				err:  empty{},
				mode: OK,
			},
			want: true,
		},
		{
			fields: fields{
				ok:   empty{},
				err:  empty{},
				mode: Err,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				ok:   tt.fields.ok,
				err:  tt.fields.err,
				mode: tt.fields.mode,
			}
			if got := r.IsSet(); got != tt.want {
				t.Errorf("Result.IsSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_Unwrap(t *testing.T) {
	type fields struct {
		ok   interface{}
		err  interface{}
		mode ResultMode
	}
	tests := []struct {
		name    string
		fields  fields
		wantOk  interface{}
		wantErr bool
	}{
		{
			fields: fields{
				ok:  empty{},
				err: empty{},
			},
			wantErr: true,
		},
		{
			fields: fields{
				ok:   empty{},
				err:  empty{},
				mode: OK,
			},
		},
		{
			fields: fields{
				ok:   empty{},
				err:  empty{},
				mode: Err,
			},
			wantErr: true,
		},
		{
			fields: fields{
				ok:   true,
				err:  empty{},
				mode: OK,
			},
			wantOk: true,
		},
		{
			fields: fields{
				ok:   empty{},
				err:  true,
				mode: Err,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				ok:   tt.fields.ok,
				err:  tt.fields.err,
				mode: tt.fields.mode,
			}
			gotOk, err := r.Unwrap()
			if (err != nil) != tt.wantErr {
				t.Errorf("Result.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotOk, tt.wantOk) {
				t.Errorf("Result.Unwrap() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestResult_Set(t *testing.T) {
	type args struct {
		mode ResultMode
		in   interface{}
	}
	tests := []struct {
		name       string
		res        Result
		args       args
		wantErr    bool
		wantResult Result
	}{
		{
			args: args{
				mode: Unset,
			},
			res:     NewResult(nil, nil),
			wantErr: true,
			wantResult: Result{
				ok: empty{}, err: empty{},
			},
		},
		{
			args: args{
				mode: OK,
				in:   nil,
			},
			res: NewResult(nil, nil),
			wantResult: Result{
				ok:   empty{},
				err:  empty{},
				mode: OK,
			},
		},
		{
			args: args{
				mode: Err,
				in:   nil,
			},
			res: NewResult(nil, nil),
			wantResult: Result{
				ok:   empty{},
				err:  empty{},
				mode: Err,
			},
		},
		{
			args: args{
				mode: OK,
				in:   true,
			},
			res: NewResult(true, nil),
			wantResult: Result{
				ok:   true,
				err:  empty{},
				mode: OK,
			},
		},
		{
			args: args{
				mode: Err,
				in:   true,
			},
			res: NewResult(nil, true),
			wantResult: Result{
				ok:   empty{},
				err:  true,
				mode: Err,
			},
		},
		{
			args: args{
				mode: OK,
				in:   true,
			},
			res:     NewResult("ok", "err"),
			wantErr: true,
			wantResult: Result{
				ok:  "ok",
				err: "err",
			},
		},
		{
			args: args{
				mode: Err,
				in:   nil,
			},
			res:     NewResult(nil, true),
			wantErr: true,
			wantResult: Result{
				ok:  empty{},
				err: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.res
			if err := r.Set(tt.args.mode, tt.args.in); (err != nil) != tt.wantErr {
				t.Errorf("Result.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.wantResult, r) {
				t.Errorf("Result.Unwrap() = %v, want %v", tt.wantResult, r)
			}
		})
	}
}
