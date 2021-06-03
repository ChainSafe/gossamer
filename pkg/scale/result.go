package scale

import (
	"fmt"
	"reflect"
)

type resultCache map[string]map[bool]interface{}
type Result struct {
	Ok  interface{}
	Err interface{}
}

// Validate ensures Result is valid.  Only one of the Ok and Err attributes should be nil
func (r Result) Validate() (err error) {
	switch {
	case r.Ok == nil && r.Err != nil, r.Ok != nil && r.Err == nil:
	default:
		err = fmt.Errorf("Result is invalid: %+v", r)
	}
	return
}

var resCache resultCache = make(resultCache)

func RegisterResult(in interface{}, inOK interface{}, inErr interface{}) (err error) {
	t := reflect.TypeOf(in)
	if !t.ConvertibleTo(reflect.TypeOf(Result{})) {
		err = fmt.Errorf("%T is not a Result", in)
		return
	}

	key := fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	_, ok := resCache[key]
	if !ok {
		resCache[key] = make(map[bool]interface{})
	}
	resCache[key][true] = inOK
	resCache[key][false] = inErr
	return
}
