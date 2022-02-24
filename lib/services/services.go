// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package services

import (
	"reflect"

	"github.com/ChainSafe/gossamer/internal/log"
)

//go:generate mockery --name Service --structname Service --case underscore --keeptree

// Service must be implemented by all Services
type Service interface {
	Start() error
	Stop() error
}

// ServiceRegistry is a structure to manage core system Services
type ServiceRegistry struct {
	Services     map[reflect.Type]Service // map of types to service instances
	ServiceTypes []reflect.Type           // all known service types, used to iterate through Services
	logger       log.LeveledLogger
}

// NewServiceRegistry creates an empty registry
func NewServiceRegistry(logger log.LeveledLogger) *ServiceRegistry {
	return &ServiceRegistry{
		Services: make(map[reflect.Type]Service),
		logger:   logger,
	}
}

// RegisterService stores a new service in the map. If a service of that type has been seen
func (s *ServiceRegistry) RegisterService(service Service) {
	kind := reflect.TypeOf(service)
	if _, exists := s.Services[kind]; exists {
		s.logger.Warnf("Tried to add service type %s that has already been seen", kind)
		return
	}
	s.Services[kind] = service
	s.ServiceTypes = append(s.ServiceTypes, kind)
}

// StartAll calls `Service.Start()` for all registered Services
func (s *ServiceRegistry) StartAll() {
	s.logger.Infof("Starting Services: %v", s.ServiceTypes)
	for _, typ := range s.ServiceTypes {
		s.logger.Debugf("Starting service %s", typ)
		err := s.Services[typ].Start()
		if err != nil {
			s.logger.Errorf("Cannot start service %s: %s", typ, err)
		}
	}
	s.logger.Debug("All Services started.")
}

// StopAll calls `Service.Stop()` for all registered Services
func (s *ServiceRegistry) StopAll() {
	s.logger.Infof("Stopping Services: %v", s.ServiceTypes)
	for _, typ := range s.ServiceTypes {
		s.logger.Debugf("Stopping service %s", typ)
		err := s.Services[typ].Stop()
		if err != nil {
			s.logger.Errorf("Error stopping service %s: %s", typ, err)
		}
	}
	s.logger.Debug("All Services stopped.")
}

// Get retrieves a service and stores a reference to it in the passed in `srvc`
func (s *ServiceRegistry) Get(srvc interface{}) Service {
	if reflect.TypeOf(srvc).Kind() != reflect.Ptr {
		s.logger.Warnf("expected a pointer but got %T", srvc)
		return nil
	}
	e := reflect.ValueOf(srvc)

	if s, ok := s.Services[e.Type()]; ok {
		return s
	}
	s.logger.Warnf("unknown service type %T", srvc)
	return nil
}
