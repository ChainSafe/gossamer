// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.

// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.

// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package api

// Service couples all components required for the API.
type Service struct {
	Api *Api
	err <-chan error
}

// Api contains all the available modules
type Api struct {
	P2pSystem *p2pModule
	RtSystem  *rtModule
}

type p2p struct {
	p2p P2pApi
}

type runtime struct {
	runtime RuntimeApi
}

// P2pApi is the interface expected to implemented by `p2p` package
type P2pApi interface {
	PeerCount() int
	Peers() []string
	ShouldHavePeers() bool
	ID() string
}

// RuntimeApi is the interface expected to implemented by `runtime` package
type RuntimeApi interface {
	// Chain() string  //Cannot implement yet
	Name() string //Replace with dynamic name later
	// properties() string //Cannot implement yet
	Version() string
}

// Module represents a collection of API endpoints.
type Module string

// NewApiService creates a new API instance.
func NewApiService(p2p P2pApi, rt RuntimeApi) *Service {
	return &Service{
		&Api{
			P2pSystem: &p2pModule{
				p2p,
			},
			RtSystem: &rtModule{
				rt,
			},
		}, nil,
	}
}

// Start creates, stores and returns an error channel
func (s *Service) Start() <-chan error {
	s.err = make(chan error)
	return s.err
}

func (s *Service) Stop() <-chan error {
	return nil
}
