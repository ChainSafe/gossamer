// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package services

import (
	"reflect"
)

// Service must be implemented by all services.
// Defines the lifecycle methods Start and Stop for services.
type Service interface {
	Start() error
	Stop() error
}

// Pausable must be implemented by services that must be paused before shutdown.
// Defines a Pause method for services that require a graceful shutdown process.
type Pausable interface {
	Pause() error
}

// ServiceRegistry is a structure to manage core system services.
// It maintains a registry of services, allowing for controlled startup, pausing, and shutdown.
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to service instances
	serviceTypes []reflect.Type           // all known service types, used to iterate through services
	logger       Logger                   // Logger for logging service operations.
}

// NewServiceRegistry creates an empty registry and return it as a pointer.
func NewServiceRegistry(logger Logger) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
		logger:   logger,
	}
}

// RegisterService stores a new service in the registry.
// The method accept as a parameter an instance of the Service interface to be registered.
// If a service of that type has already been registered, it logs a warning and does not register the service again.
// The method guarantee that only one instance of a service can be added in the registry.
//
// The order in which services are added to the registry is important because later they will be started in the same order
func (s *ServiceRegistry) RegisterService(service Service) {
	// by using type of the service as a key in the map, we guarantee that only one instance of the service can be registered.
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		s.logger.Warnf("Tried to add service type %s that has already been seen", kind)
		return
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
}

// StartAll calls Service.Start() for all registered services.
// The services are started in the order they were registered.
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

// PauseServices pauses key services before shutdown to allow a graceful shutdown.
// Only services that implement the Pausable interface will be paused.
func (s *ServiceRegistry) PauseServices() {
	s.logger.Infof("Pausing key services")
	for _, typ := range s.serviceTypes {
		pausable, ok := s.services[typ].(Pausable)
		if ok && pausable != nil {
			err := pausable.Pause()
			if err != nil {
				s.logger.Errorf("Error pausing %s service: %s", typ, err)
			}
		} else if ok {
			s.logger.Errorf("Error pausing required services")
		}
	}

	s.logger.Infof("Paused key services")
}

// StopAll calls Service.Stop() for all registered services.
// Before stopping, it pauses the services if they implement the Pausable interface.
func (s *ServiceRegistry) StopAll() {
	s.logger.Infof("Stopping services: %v", s.serviceTypes)

	s.PauseServices()

	for _, typ := range s.serviceTypes {
		s.logger.Debugf("Stopping service %s", typ)
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
