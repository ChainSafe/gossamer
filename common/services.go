// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package common

import (
	"fmt"
	log "github.com/ChainSafe/log15"
	"reflect"
)

type Service interface {
	Start() <-chan error
	Stop()
}

type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to service instances
	serviceTypes []reflect.Type           // all known service types, used to iterate through `ServiceRegistry.services`
}

// NewServiceRegistry creates an empty registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type] Service),
	}
}

// RegisterService stores a new service in the map. If a service of that type has been seen
func (s *ServiceRegistry) RegisterService(service Service) {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		log.Warn("Tried to add service type that has already been seen", "type", kind)
		return
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
}

// StartAll calls `Service.Start()` for all registered services
func (s *ServiceRegistry) StartAll() {
	log.Info("Starting services: %v", s.serviceTypes)
	for _, kind := range s.serviceTypes {
		log.Debug(fmt.Sprintf("Starting service %v", kind))
		// TODO: Handle channel that is returned
		s.services[kind].Start()
	}
}

// StopAll calls `Service.Stop()` for all registered services
func (s *ServiceRegistry) StopAll() {
	// TODO: Implement
}