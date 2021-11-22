// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package httpserver

// GetAddress obtains the address the HTTP server is listening on.
func (s *Server) GetAddress() (address string) {
	<-s.addressSet
	return s.address
}
