package types

import (
	"reflect"
	"testing"
)

func TestBodyToExtrinsics(t *testing.T) {
	exts := []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	body, err := NewBodyFromExtrinsics(exts)
	if err != nil {
		t.Fatal(err)
	}

	res, err := body.AsExtrinsics()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, exts) {
		t.Fatalf("Fail: got %x expected %x", res, exts)
	}
}