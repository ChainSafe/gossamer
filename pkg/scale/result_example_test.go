package scale_test

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

func ExampleResult() {
	// pass in zero or non-zero values of the types for Ok and Err cases
	res := scale.NewResult(bool(false), string(""))

	// set the OK case with a value of true, any values for OK that are not bool will return an error
	err := res.Set(scale.OK, true)
	if err != nil {
		panic(err)
	}

	bytes, err := scale.Marshal(res)
	if err != nil {
		panic(err)
	}

	// [0x00, 0x01]
	fmt.Printf("%v\n", bytes)

	res1 := scale.NewResult(bool(false), string(""))

	err = scale.Unmarshal(bytes, &res1)
	if err != nil {
		panic(err)
	}

	// res1 should be Set with OK mode and value of true
	ok, err := res1.Unwrap()
	if err != nil {
		panic(err)
	}

	switch ok := ok.(type) {
	case bool:
		if !ok {
			panic(fmt.Errorf("unexpected ok value: %v", ok))
		}
	default:
		panic(fmt.Errorf("unexpected type: %T", ok))
	}
}

func TestExampleResult(t *testing.T) {
	ExampleResult()
}
