// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package system

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

// Service struct to hold rpc service data
type Service struct {
	systemInfo  *types.SystemInfo
	genesisData *genesis.Data
}

// NewService create a new instance of Service
func NewService(si *types.SystemInfo, gd *genesis.Data) *Service {
	return &Service{
		systemInfo:  si,
		genesisData: gd,
	}
}

// SystemName returns the app name
func (s *Service) SystemName() string {
	return s.systemInfo.SystemName
}

// ChainType returns the system's chain type
func (s *Service) ChainType() string {
	return s.genesisData.ChainType
}

// SystemVersion returns the app version
func (s *Service) SystemVersion() string {
	return s.systemInfo.SystemVersion
}

// ChainName returns the chain name defined in genesis.json
func (s *Service) ChainName() string {
	return s.genesisData.Name
}

// Properties Get a custom set of properties as a JSON object, defined in the chain spec.
func (s *Service) Properties() map[string]interface{} {
	return s.genesisData.Properties
}

// Start implements Service interface
func (s *Service) Start() error {
	return nil
}

// Stop implements Service interface
func (s *Service) Stop() error {
	return nil
}
