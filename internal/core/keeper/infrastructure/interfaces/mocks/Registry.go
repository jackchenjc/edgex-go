// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	models "github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	mock "github.com/stretchr/testify/mock"
)

// Registry is an autogenerated mock type for the Registry type
type Registry struct {
	mock.Mock
}

// DeregisterByServiceId provides a mock function with given fields: id
func (_m *Registry) DeregisterByServiceId(id string) {
	_m.Called(id)
}

// Register provides a mock function with given fields: r
func (_m *Registry) Register(r models.Registration) {
	_m.Called(r)
}

// NewRegistry creates a new instance of Registry. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRegistry(t interface {
	mock.TestingT
	Cleanup(func())
}) *Registry {
	mock := &Registry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
