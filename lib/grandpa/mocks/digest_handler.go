// Code generated by mockery v2.8.0. DO NOT EDIT.

package grandpa

import mock "github.com/stretchr/testify/mock"

// MockDigestHandler is an autogenerated mock type for the DigestHandler type
type MockDigestHandler struct {
	mock.Mock
}

// NextGrandpaAuthorityChange provides a mock function with given fields:
func (_m *MockDigestHandler) NextGrandpaAuthorityChange() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}
