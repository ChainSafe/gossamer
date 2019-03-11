package p2p

import (
	"testing"
)

var testServiceConfig = &ServiceConfig{
	//bootstrapNode: ""
}

func TestStart(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	err = s.Start()
	if err != nil {
		t.Errorf("Start error :%s", err)
	}
}

func TestStartDHT(t *testing.T) {
	s, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	err = s.Start()
	if err != nil {
		t.Errorf("TestStart:%s", err)
	}

	_, err = startDHT(s.host)
	if err != nil {
		t.Errorf("TestStartDHT:%s", err)
	}
}
