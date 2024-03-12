// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package services

import (
	"reflect"
)

// Service must be implemented by all services
type Service interface {
	Start() error
	Stop() error
	Pause() error
}

// ServiceRegistry is a structure to manage core system services
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to service instances
	serviceTypes []reflect.Type           // all known service types, used to iterate through services
	logger       Logger
}

// NewServiceRegistry creates an empty registry
func NewServiceRegistry(logger Logger) *ServiceRegistry {
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
		s.logger.Debugf("Starting service %s", typ)
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

	// Pause the sync and state service to allow for graceful shutdown
	syncService := s.serviceTypes[5]
	err := s.services[syncService].Pause()
	if err != nil {
		s.logger.Errorf("Error pausing service %s: %s", syncService, err)
	}

	stateService := s.serviceTypes[len(s.serviceTypes)-1]
	err = s.services[stateService].Pause()
	if err != nil {
		s.logger.Errorf("Error pausing service %s: %s", stateService, err)
	}

	for _, typ := range s.serviceTypes {
		s.logger.Debugf("Stopping service %s", typ)
		err := s.services[typ].Stop()
		if err != nil {
			s.logger.Errorf("Error stopping service %s: %s", typ, err)
		}
	}
	s.logger.Debugf("All services stopped.")
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
