package p2p

import (
	"testing"
)

func TestStart(t *testing.T) {
	_, err := Start()
	if err != nil {
		t.Errorf("TestStart:%s", err)
	}
}