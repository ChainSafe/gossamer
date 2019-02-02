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

func TestStartDHT(t *testing.T) {
	host, err := Start()
	if err != nil {
		t.Errorf("TestStart:%s", err)
	}

	_, err = startDHT(host)
	if err != nil {
		t.Errorf("TestStartDHT:%s", err)
	}
}