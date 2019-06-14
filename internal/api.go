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

import (
	"github.com/ChainSafe/gossamer/p2p"
)

// Service couples all components required for the API.
type Service struct {
	Core *coreModule
}

// Module represents a collection of API endpoints.
type Module string


// NewApiService creates a new API instance.
func NewApiService(p2p *p2p.Service) *Service {
	return &Service{
		Core: &coreModule{
			p2p,
		},
	}
}
