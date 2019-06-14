package common

import "testing"

// ------------------- Mock Services --------------------
type MockSrvcA struct {
	running bool
}

func(s *MockSrvcA) Start() <-chan error{
	s.running = true
	return make(chan error)
}
func(s *MockSrvcA) Stop() {
	s.running = false
}

type MockSrvcB struct {
	running bool
}

func(s *MockSrvcB) Start() <-chan error{
	s.running = true
	return make(chan error)
}
func(s *MockSrvcB) Stop() {
	s.running = false
}

// --------------------------------------------------------

func TestServiceRegistry_RegisterService(t *testing.T) {
	r := NewServiceRegistry()

	a1 := &MockSrvcA{}
	a2 := &MockSrvcA{}

	r.RegisterService(a1)
	r.RegisterService(a2)

	if len(r.serviceTypes) > 1 {
		t.Fatalf("should not allow services of the same type to be registered")
	}
}

func TestServiceRegistry_StartAll(t *testing.T) {
	r := NewServiceRegistry()

	a := &MockSrvcA{}
	b := &MockSrvcB{}

	r.RegisterService(a)
	r.RegisterService(b)

	r.StartAll()

	if a.running != true || b.running != true {
		t.Fatal("failed to start service")
	}
}