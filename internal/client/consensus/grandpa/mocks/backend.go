// Code generated by mockery v2.39.1. DO NOT EDIT.

package mocks

import (
	api "github.com/ChainSafe/gossamer/internal/client/api"
	blockchain "github.com/ChainSafe/gossamer/internal/primitives/blockchain"

	mock "github.com/stretchr/testify/mock"

	runtime "github.com/ChainSafe/gossamer/internal/primitives/runtime"

	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"

	sync "sync"
)

// Backend is an autogenerated mock type for the Backend type
type Backend[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	mock.Mock
}

type Backend_Expecter[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	mock *mock.Mock
}

func (_m *Backend[H, N, Hasher]) EXPECT() *Backend_Expecter[H, N, Hasher] {
	return &Backend_Expecter[H, N, Hasher]{mock: &_m.Mock}
}

// BeginOperation provides a mock function with given fields:
func (_m *Backend[H, N, Hasher]) BeginOperation() (api.BlockImportOperation[N, H, Hasher], error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for BeginOperation")
	}

	var r0 api.BlockImportOperation[N, H, Hasher]
	var r1 error
	if rf, ok := ret.Get(0).(func() (api.BlockImportOperation[N, H, Hasher], error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() api.BlockImportOperation[N, H, Hasher]); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.BlockImportOperation[N, H, Hasher])
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Backend_BeginOperation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BeginOperation'
type Backend_BeginOperation_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// BeginOperation is a helper method to define mock.On call
func (_e *Backend_Expecter[H, N, Hasher]) BeginOperation() *Backend_BeginOperation_Call[H, N, Hasher] {
	return &Backend_BeginOperation_Call[H, N, Hasher]{Call: _e.mock.On("BeginOperation")}
}

func (_c *Backend_BeginOperation_Call[H, N, Hasher]) Run(run func()) *Backend_BeginOperation_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Backend_BeginOperation_Call[H, N, Hasher]) Return(_a0 api.BlockImportOperation[N, H, Hasher], _a1 error) *Backend_BeginOperation_Call[H, N, Hasher] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Backend_BeginOperation_Call[H, N, Hasher]) RunAndReturn(run func() (api.BlockImportOperation[N, H, Hasher], error)) *Backend_BeginOperation_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// BeginStateOperation provides a mock function with given fields: operation, block
func (_m *Backend[H, N, Hasher]) BeginStateOperation(operation *api.BlockImportOperation[N, H, Hasher], block H) error {
	ret := _m.Called(operation, block)

	if len(ret) == 0 {
		panic("no return value specified for BeginStateOperation")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*api.BlockImportOperation[N, H, Hasher], H) error); ok {
		r0 = rf(operation, block)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Backend_BeginStateOperation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BeginStateOperation'
type Backend_BeginStateOperation_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// BeginStateOperation is a helper method to define mock.On call
//   - operation *api.BlockImportOperation[N,H,Hasher]
//   - block H
func (_e *Backend_Expecter[H, N, Hasher]) BeginStateOperation(operation interface{}, block interface{}) *Backend_BeginStateOperation_Call[H, N, Hasher] {
	return &Backend_BeginStateOperation_Call[H, N, Hasher]{Call: _e.mock.On("BeginStateOperation", operation, block)}
}

func (_c *Backend_BeginStateOperation_Call[H, N, Hasher]) Run(run func(operation *api.BlockImportOperation[N, H, Hasher], block H)) *Backend_BeginStateOperation_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*api.BlockImportOperation[N, H, Hasher]), args[1].(H))
	})
	return _c
}

func (_c *Backend_BeginStateOperation_Call[H, N, Hasher]) Return(_a0 error) *Backend_BeginStateOperation_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_BeginStateOperation_Call[H, N, Hasher]) RunAndReturn(run func(*api.BlockImportOperation[N, H, Hasher], H) error) *Backend_BeginStateOperation_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// Blockchain provides a mock function with given fields:
func (_m *Backend[H, N, Hasher]) Blockchain() blockchain.Backend[H, N] {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Blockchain")
	}

	var r0 blockchain.Backend[H, N]
	if rf, ok := ret.Get(0).(func() blockchain.Backend[H, N]); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(blockchain.Backend[H, N])
		}
	}

	return r0
}

// Backend_Blockchain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Blockchain'
type Backend_Blockchain_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// Blockchain is a helper method to define mock.On call
func (_e *Backend_Expecter[H, N, Hasher]) Blockchain() *Backend_Blockchain_Call[H, N, Hasher] {
	return &Backend_Blockchain_Call[H, N, Hasher]{Call: _e.mock.On("Blockchain")}
}

func (_c *Backend_Blockchain_Call[H, N, Hasher]) Run(run func()) *Backend_Blockchain_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Backend_Blockchain_Call[H, N, Hasher]) Return(_a0 blockchain.Backend[H, N]) *Backend_Blockchain_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_Blockchain_Call[H, N, Hasher]) RunAndReturn(run func() blockchain.Backend[H, N]) *Backend_Blockchain_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// CommitOperation provides a mock function with given fields: transaction
func (_m *Backend[H, N, Hasher]) CommitOperation(transaction api.BlockImportOperation[N, H, Hasher]) error {
	ret := _m.Called(transaction)

	if len(ret) == 0 {
		panic("no return value specified for CommitOperation")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(api.BlockImportOperation[N, H, Hasher]) error); ok {
		r0 = rf(transaction)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Backend_CommitOperation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CommitOperation'
type Backend_CommitOperation_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// CommitOperation is a helper method to define mock.On call
//   - transaction api.BlockImportOperation[N,H,Hasher]
func (_e *Backend_Expecter[H, N, Hasher]) CommitOperation(transaction interface{}) *Backend_CommitOperation_Call[H, N, Hasher] {
	return &Backend_CommitOperation_Call[H, N, Hasher]{Call: _e.mock.On("CommitOperation", transaction)}
}

func (_c *Backend_CommitOperation_Call[H, N, Hasher]) Run(run func(transaction api.BlockImportOperation[N, H, Hasher])) *Backend_CommitOperation_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(api.BlockImportOperation[N, H, Hasher]))
	})
	return _c
}

func (_c *Backend_CommitOperation_Call[H, N, Hasher]) Return(_a0 error) *Backend_CommitOperation_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_CommitOperation_Call[H, N, Hasher]) RunAndReturn(run func(api.BlockImportOperation[N, H, Hasher]) error) *Backend_CommitOperation_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// GetAux provides a mock function with given fields: key
func (_m *Backend[H, N, Hasher]) GetAux(key []byte) (*[]byte, error) {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for GetAux")
	}

	var r0 *[]byte
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) (*[]byte, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func([]byte) *[]byte); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*[]byte)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Backend_GetAux_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAux'
type Backend_GetAux_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// GetAux is a helper method to define mock.On call
//   - key []byte
func (_e *Backend_Expecter[H, N, Hasher]) GetAux(key interface{}) *Backend_GetAux_Call[H, N, Hasher] {
	return &Backend_GetAux_Call[H, N, Hasher]{Call: _e.mock.On("GetAux", key)}
}

func (_c *Backend_GetAux_Call[H, N, Hasher]) Run(run func(key []byte)) *Backend_GetAux_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]byte))
	})
	return _c
}

func (_c *Backend_GetAux_Call[H, N, Hasher]) Return(_a0 *[]byte, _a1 error) *Backend_GetAux_Call[H, N, Hasher] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Backend_GetAux_Call[H, N, Hasher]) RunAndReturn(run func([]byte) (*[]byte, error)) *Backend_GetAux_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// GetImportLock provides a mock function with given fields:
func (_m *Backend[H, N, Hasher]) GetImportLock() *sync.RWMutex {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetImportLock")
	}

	var r0 *sync.RWMutex
	if rf, ok := ret.Get(0).(func() *sync.RWMutex); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sync.RWMutex)
		}
	}

	return r0
}

// Backend_GetImportLock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetImportLock'
type Backend_GetImportLock_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// GetImportLock is a helper method to define mock.On call
func (_e *Backend_Expecter[H, N, Hasher]) GetImportLock() *Backend_GetImportLock_Call[H, N, Hasher] {
	return &Backend_GetImportLock_Call[H, N, Hasher]{Call: _e.mock.On("GetImportLock")}
}

func (_c *Backend_GetImportLock_Call[H, N, Hasher]) Run(run func()) *Backend_GetImportLock_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Backend_GetImportLock_Call[H, N, Hasher]) Return(_a0 *sync.RWMutex) *Backend_GetImportLock_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_GetImportLock_Call[H, N, Hasher]) RunAndReturn(run func() *sync.RWMutex) *Backend_GetImportLock_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// HaveStateAt provides a mock function with given fields: hash, number
func (_m *Backend[H, N, Hasher]) HaveStateAt(hash H, number N) bool {
	ret := _m.Called(hash, number)

	if len(ret) == 0 {
		panic("no return value specified for HaveStateAt")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(H, N) bool); ok {
		r0 = rf(hash, number)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Backend_HaveStateAt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HaveStateAt'
type Backend_HaveStateAt_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// HaveStateAt is a helper method to define mock.On call
//   - hash H
//   - number N
func (_e *Backend_Expecter[H, N, Hasher]) HaveStateAt(hash interface{}, number interface{}) *Backend_HaveStateAt_Call[H, N, Hasher] {
	return &Backend_HaveStateAt_Call[H, N, Hasher]{Call: _e.mock.On("HaveStateAt", hash, number)}
}

func (_c *Backend_HaveStateAt_Call[H, N, Hasher]) Run(run func(hash H, number N)) *Backend_HaveStateAt_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(H), args[1].(N))
	})
	return _c
}

func (_c *Backend_HaveStateAt_Call[H, N, Hasher]) Return(_a0 bool) *Backend_HaveStateAt_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_HaveStateAt_Call[H, N, Hasher]) RunAndReturn(run func(H, N) bool) *Backend_HaveStateAt_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// InsertAux provides a mock function with given fields: insert, delete
func (_m *Backend[H, N, Hasher]) InsertAux(insert []api.KeyValue, delete [][]byte) error {
	ret := _m.Called(insert, delete)

	if len(ret) == 0 {
		panic("no return value specified for InsertAux")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]api.KeyValue, [][]byte) error); ok {
		r0 = rf(insert, delete)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Backend_InsertAux_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'InsertAux'
type Backend_InsertAux_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// InsertAux is a helper method to define mock.On call
//   - insert []api.KeyValue
//   - delete [][]byte
func (_e *Backend_Expecter[H, N, Hasher]) InsertAux(insert interface{}, delete interface{}) *Backend_InsertAux_Call[H, N, Hasher] {
	return &Backend_InsertAux_Call[H, N, Hasher]{Call: _e.mock.On("InsertAux", insert, delete)}
}

func (_c *Backend_InsertAux_Call[H, N, Hasher]) Run(run func(insert []api.KeyValue, delete [][]byte)) *Backend_InsertAux_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]api.KeyValue), args[1].([][]byte))
	})
	return _c
}

func (_c *Backend_InsertAux_Call[H, N, Hasher]) Return(_a0 error) *Backend_InsertAux_Call[H, N, Hasher] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Backend_InsertAux_Call[H, N, Hasher]) RunAndReturn(run func([]api.KeyValue, [][]byte) error) *Backend_InsertAux_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// StateAt provides a mock function with given fields: hash
func (_m *Backend[H, N, Hasher]) StateAt(hash H) (statemachine.Backend[H, Hasher], error) {
	ret := _m.Called(hash)

	if len(ret) == 0 {
		panic("no return value specified for StateAt")
	}

	var r0 statemachine.Backend[H, Hasher]
	var r1 error
	if rf, ok := ret.Get(0).(func(H) (statemachine.Backend[H, Hasher], error)); ok {
		return rf(hash)
	}
	if rf, ok := ret.Get(0).(func(H) statemachine.Backend[H, Hasher]); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(statemachine.Backend[H, Hasher])
		}
	}

	if rf, ok := ret.Get(1).(func(H) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Backend_StateAt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StateAt'
type Backend_StateAt_Call[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] struct {
	*mock.Call
}

// StateAt is a helper method to define mock.On call
//   - hash H
func (_e *Backend_Expecter[H, N, Hasher]) StateAt(hash interface{}) *Backend_StateAt_Call[H, N, Hasher] {
	return &Backend_StateAt_Call[H, N, Hasher]{Call: _e.mock.On("StateAt", hash)}
}

func (_c *Backend_StateAt_Call[H, N, Hasher]) Run(run func(hash H)) *Backend_StateAt_Call[H, N, Hasher] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(H))
	})
	return _c
}

func (_c *Backend_StateAt_Call[H, N, Hasher]) Return(_a0 statemachine.Backend[H, Hasher], _a1 error) *Backend_StateAt_Call[H, N, Hasher] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Backend_StateAt_Call[H, N, Hasher]) RunAndReturn(run func(H) (statemachine.Backend[H, Hasher], error)) *Backend_StateAt_Call[H, N, Hasher] {
	_c.Call.Return(run)
	return _c
}

// NewBackend creates a new instance of Backend. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBackend[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]](t interface {
	mock.TestingT
	Cleanup(func())
}) *Backend[H, N, Hasher] {
	mock := &Backend[H, N, Hasher]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
