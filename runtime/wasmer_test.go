package runtime

import (
	"testing"
)

func TestExecWasmer(t *testing.T) {
	ret, err := Exec()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ret)
}