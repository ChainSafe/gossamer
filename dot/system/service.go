// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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

// Pause Not needed for system service but required for interface
func (s *Service) Pause() error {
	panic("only here for to adhere to interface")
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
func (*Service) Start() error {
	return nil
}

// Stop implements Service interface
func (*Service) Stop() error {
	return nil
}
