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

package services

import (
	"reflect"

	logc "github.com/ChainSafe/gossamer/internal/log/common"
)

//go:generate mockery --name Service --structname Service --case underscore --keeptree

// Service must be implemented by all services
type Service interface {
	Start() error
	Stop() error
}

// ServiceRegistry is a structure to manage core system services
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to service instances
	serviceTypes []reflect.Type           // all known service types, used to iterate through services
	logger       logc.Logger
}

// NewServiceRegistry creates an empty registry
func NewServiceRegistry(logger logc.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
		logger:   logger,
	}
}

// RegisterService stores a new service in the map. If a service of that type has been seen
func (s *ServiceRegistry) RegisterService(service Service) {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		s.logger.Warnf("Tried to add service type %s that has already been seen", kind)
		return
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
}

// StartAll calls `Service.Start()` for all registered services
func (s *ServiceRegistry) StartAll() {
	s.logger.Infof("Starting services: %v", s.serviceTypes)
	for _, typ := range s.serviceTypes {
		s.logger.Debugf("Starting service %v", typ)
		err := s.services[typ].Start()
		if err != nil {
			s.logger.Errorf("Cannot start service %s: %s", typ, err)
		}
	}
	s.logger.Debug("All services started.")
}

// StopAll calls `Service.Stop()` for all registered services
func (s *ServiceRegistry) StopAll() {
	s.logger.Infof("Stopping services: %v", s.serviceTypes)
	for _, typ := range s.serviceTypes {
		s.logger.Debugf("Stopping service %v", typ)
		err := s.services[typ].Stop()
		if err != nil {
			s.logger.Errorf("Error stopping service %s: %s", typ, err)
		}
	}
	s.logger.Debug("All services stopped.")
}

// Get retrieves a service and stores a reference to it in the passed in `srvc`
func (s *ServiceRegistry) Get(srvc interface{}) Service {
	if reflect.TypeOf(srvc).Kind() != reflect.Ptr {
		s.logger.Warnf("expected a pointer but got %T", srvc)
		return nil
	}
	e := reflect.ValueOf(srvc)

	if s, ok := s.services[e.Type()]; ok {
		return s
	}
	s.logger.Warnf("unknown service type %T", srvc)
	return nil
}
