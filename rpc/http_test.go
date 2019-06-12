package rpc

import "testing"

func TestServerSetup(t *testing.T) {
	s := NewHttpServer(3000)
	s.Start()
}