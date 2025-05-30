// Code generated by mockery v2.50.0. DO NOT EDIT.

package service

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	model "github.com/webitel/webitel-wfm/internal/model"

	options "github.com/webitel/webitel-wfm/internal/model/options"
)

// MockAgentAbsenceManager is an autogenerated mock type for the AgentAbsenceManager type
type MockAgentAbsenceManager struct {
	mock.Mock
}

type MockAgentAbsenceManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAgentAbsenceManager) EXPECT() *MockAgentAbsenceManager_Expecter {
	return &MockAgentAbsenceManager_Expecter{mock: &_m.Mock}
}

// CreateAgentAbsence provides a mock function with given fields: ctx, read, in
func (_m *MockAgentAbsenceManager) CreateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error) {
	ret := _m.Called(ctx, read, in)

	if len(ret) == 0 {
		panic("no return value specified for CreateAgentAbsence")
	}

	var r0 *model.Absence
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read, *model.Absence) (*model.Absence, error)); ok {
		return rf(ctx, read, in)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read, *model.Absence) *model.Absence); ok {
		r0 = rf(ctx, read, in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Absence)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Read, *model.Absence) error); ok {
		r1 = rf(ctx, read, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAgentAbsenceManager_CreateAgentAbsence_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateAgentAbsence'
type MockAgentAbsenceManager_CreateAgentAbsence_Call struct {
	*mock.Call
}

// CreateAgentAbsence is a helper method to define mock.On call
//   - ctx context.Context
//   - read *options.Read
//   - in *model.Absence
func (_e *MockAgentAbsenceManager_Expecter) CreateAgentAbsence(ctx interface{}, read interface{}, in interface{}) *MockAgentAbsenceManager_CreateAgentAbsence_Call {
	return &MockAgentAbsenceManager_CreateAgentAbsence_Call{Call: _e.mock.On("CreateAgentAbsence", ctx, read, in)}
}

func (_c *MockAgentAbsenceManager_CreateAgentAbsence_Call) Run(run func(ctx context.Context, read *options.Read, in *model.Absence)) *MockAgentAbsenceManager_CreateAgentAbsence_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Read), args[2].(*model.Absence))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_CreateAgentAbsence_Call) Return(_a0 *model.Absence, _a1 error) *MockAgentAbsenceManager_CreateAgentAbsence_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAgentAbsenceManager_CreateAgentAbsence_Call) RunAndReturn(run func(context.Context, *options.Read, *model.Absence) (*model.Absence, error)) *MockAgentAbsenceManager_CreateAgentAbsence_Call {
	_c.Call.Return(run)
	return _c
}

// CreateAgentsAbsences provides a mock function with given fields: ctx, search, in
func (_m *MockAgentAbsenceManager) CreateAgentsAbsences(ctx context.Context, search *options.Search, in []*model.AgentAbsences) ([]*model.AgentAbsences, error) {
	ret := _m.Called(ctx, search, in)

	if len(ret) == 0 {
		panic("no return value specified for CreateAgentsAbsences")
	}

	var r0 []*model.AgentAbsences
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search, []*model.AgentAbsences) ([]*model.AgentAbsences, error)); ok {
		return rf(ctx, search, in)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search, []*model.AgentAbsences) []*model.AgentAbsences); ok {
		r0 = rf(ctx, search, in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.AgentAbsences)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Search, []*model.AgentAbsences) error); ok {
		r1 = rf(ctx, search, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAgentAbsenceManager_CreateAgentsAbsences_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateAgentsAbsences'
type MockAgentAbsenceManager_CreateAgentsAbsences_Call struct {
	*mock.Call
}

// CreateAgentsAbsences is a helper method to define mock.On call
//   - ctx context.Context
//   - search *options.Search
//   - in []*model.AgentAbsences
func (_e *MockAgentAbsenceManager_Expecter) CreateAgentsAbsences(ctx interface{}, search interface{}, in interface{}) *MockAgentAbsenceManager_CreateAgentsAbsences_Call {
	return &MockAgentAbsenceManager_CreateAgentsAbsences_Call{Call: _e.mock.On("CreateAgentsAbsences", ctx, search, in)}
}

func (_c *MockAgentAbsenceManager_CreateAgentsAbsences_Call) Run(run func(ctx context.Context, search *options.Search, in []*model.AgentAbsences)) *MockAgentAbsenceManager_CreateAgentsAbsences_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Search), args[2].([]*model.AgentAbsences))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_CreateAgentsAbsences_Call) Return(_a0 []*model.AgentAbsences, _a1 error) *MockAgentAbsenceManager_CreateAgentsAbsences_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAgentAbsenceManager_CreateAgentsAbsences_Call) RunAndReturn(run func(context.Context, *options.Search, []*model.AgentAbsences) ([]*model.AgentAbsences, error)) *MockAgentAbsenceManager_CreateAgentsAbsences_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteAgentAbsence provides a mock function with given fields: ctx, read
func (_m *MockAgentAbsenceManager) DeleteAgentAbsence(ctx context.Context, read *options.Read) error {
	ret := _m.Called(ctx, read)

	if len(ret) == 0 {
		panic("no return value specified for DeleteAgentAbsence")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read) error); ok {
		r0 = rf(ctx, read)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAgentAbsenceManager_DeleteAgentAbsence_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteAgentAbsence'
type MockAgentAbsenceManager_DeleteAgentAbsence_Call struct {
	*mock.Call
}

// DeleteAgentAbsence is a helper method to define mock.On call
//   - ctx context.Context
//   - read *options.Read
func (_e *MockAgentAbsenceManager_Expecter) DeleteAgentAbsence(ctx interface{}, read interface{}) *MockAgentAbsenceManager_DeleteAgentAbsence_Call {
	return &MockAgentAbsenceManager_DeleteAgentAbsence_Call{Call: _e.mock.On("DeleteAgentAbsence", ctx, read)}
}

func (_c *MockAgentAbsenceManager_DeleteAgentAbsence_Call) Run(run func(ctx context.Context, read *options.Read)) *MockAgentAbsenceManager_DeleteAgentAbsence_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Read))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_DeleteAgentAbsence_Call) Return(_a0 error) *MockAgentAbsenceManager_DeleteAgentAbsence_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAgentAbsenceManager_DeleteAgentAbsence_Call) RunAndReturn(run func(context.Context, *options.Read) error) *MockAgentAbsenceManager_DeleteAgentAbsence_Call {
	_c.Call.Return(run)
	return _c
}

// ReadAgentAbsence provides a mock function with given fields: ctx, read
func (_m *MockAgentAbsenceManager) ReadAgentAbsence(ctx context.Context, read *options.Read) (*model.Absence, error) {
	ret := _m.Called(ctx, read)

	if len(ret) == 0 {
		panic("no return value specified for ReadAgentAbsence")
	}

	var r0 *model.Absence
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read) (*model.Absence, error)); ok {
		return rf(ctx, read)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read) *model.Absence); ok {
		r0 = rf(ctx, read)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Absence)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Read) error); ok {
		r1 = rf(ctx, read)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAgentAbsenceManager_ReadAgentAbsence_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadAgentAbsence'
type MockAgentAbsenceManager_ReadAgentAbsence_Call struct {
	*mock.Call
}

// ReadAgentAbsence is a helper method to define mock.On call
//   - ctx context.Context
//   - read *options.Read
func (_e *MockAgentAbsenceManager_Expecter) ReadAgentAbsence(ctx interface{}, read interface{}) *MockAgentAbsenceManager_ReadAgentAbsence_Call {
	return &MockAgentAbsenceManager_ReadAgentAbsence_Call{Call: _e.mock.On("ReadAgentAbsence", ctx, read)}
}

func (_c *MockAgentAbsenceManager_ReadAgentAbsence_Call) Run(run func(ctx context.Context, read *options.Read)) *MockAgentAbsenceManager_ReadAgentAbsence_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Read))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_ReadAgentAbsence_Call) Return(_a0 *model.Absence, _a1 error) *MockAgentAbsenceManager_ReadAgentAbsence_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAgentAbsenceManager_ReadAgentAbsence_Call) RunAndReturn(run func(context.Context, *options.Read) (*model.Absence, error)) *MockAgentAbsenceManager_ReadAgentAbsence_Call {
	_c.Call.Return(run)
	return _c
}

// SearchAgentAbsence provides a mock function with given fields: ctx, search
func (_m *MockAgentAbsenceManager) SearchAgentAbsence(ctx context.Context, search *options.Search) ([]*model.Absence, error) {
	ret := _m.Called(ctx, search)

	if len(ret) == 0 {
		panic("no return value specified for SearchAgentAbsence")
	}

	var r0 []*model.Absence
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search) ([]*model.Absence, error)); ok {
		return rf(ctx, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search) []*model.Absence); ok {
		r0 = rf(ctx, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Absence)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Search) error); ok {
		r1 = rf(ctx, search)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAgentAbsenceManager_SearchAgentAbsence_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SearchAgentAbsence'
type MockAgentAbsenceManager_SearchAgentAbsence_Call struct {
	*mock.Call
}

// SearchAgentAbsence is a helper method to define mock.On call
//   - ctx context.Context
//   - search *options.Search
func (_e *MockAgentAbsenceManager_Expecter) SearchAgentAbsence(ctx interface{}, search interface{}) *MockAgentAbsenceManager_SearchAgentAbsence_Call {
	return &MockAgentAbsenceManager_SearchAgentAbsence_Call{Call: _e.mock.On("SearchAgentAbsence", ctx, search)}
}

func (_c *MockAgentAbsenceManager_SearchAgentAbsence_Call) Run(run func(ctx context.Context, search *options.Search)) *MockAgentAbsenceManager_SearchAgentAbsence_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Search))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_SearchAgentAbsence_Call) Return(_a0 []*model.Absence, _a1 error) *MockAgentAbsenceManager_SearchAgentAbsence_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAgentAbsenceManager_SearchAgentAbsence_Call) RunAndReturn(run func(context.Context, *options.Search) ([]*model.Absence, error)) *MockAgentAbsenceManager_SearchAgentAbsence_Call {
	_c.Call.Return(run)
	return _c
}

// SearchAgentsAbsences provides a mock function with given fields: ctx, search
func (_m *MockAgentAbsenceManager) SearchAgentsAbsences(ctx context.Context, search *options.Search) ([]*model.AgentAbsences, bool, error) {
	ret := _m.Called(ctx, search)

	if len(ret) == 0 {
		panic("no return value specified for SearchAgentsAbsences")
	}

	var r0 []*model.AgentAbsences
	var r1 bool
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search) ([]*model.AgentAbsences, bool, error)); ok {
		return rf(ctx, search)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Search) []*model.AgentAbsences); ok {
		r0 = rf(ctx, search)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.AgentAbsences)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Search) bool); ok {
		r1 = rf(ctx, search)
	} else {
		r1 = ret.Get(1).(bool)
	}

	if rf, ok := ret.Get(2).(func(context.Context, *options.Search) error); ok {
		r2 = rf(ctx, search)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// MockAgentAbsenceManager_SearchAgentsAbsences_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SearchAgentsAbsences'
type MockAgentAbsenceManager_SearchAgentsAbsences_Call struct {
	*mock.Call
}

// SearchAgentsAbsences is a helper method to define mock.On call
//   - ctx context.Context
//   - search *options.Search
func (_e *MockAgentAbsenceManager_Expecter) SearchAgentsAbsences(ctx interface{}, search interface{}) *MockAgentAbsenceManager_SearchAgentsAbsences_Call {
	return &MockAgentAbsenceManager_SearchAgentsAbsences_Call{Call: _e.mock.On("SearchAgentsAbsences", ctx, search)}
}

func (_c *MockAgentAbsenceManager_SearchAgentsAbsences_Call) Run(run func(ctx context.Context, search *options.Search)) *MockAgentAbsenceManager_SearchAgentsAbsences_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Search))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_SearchAgentsAbsences_Call) Return(_a0 []*model.AgentAbsences, _a1 bool, _a2 error) *MockAgentAbsenceManager_SearchAgentsAbsences_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *MockAgentAbsenceManager_SearchAgentsAbsences_Call) RunAndReturn(run func(context.Context, *options.Search) ([]*model.AgentAbsences, bool, error)) *MockAgentAbsenceManager_SearchAgentsAbsences_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateAgentAbsence provides a mock function with given fields: ctx, read, in
func (_m *MockAgentAbsenceManager) UpdateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error) {
	ret := _m.Called(ctx, read, in)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAgentAbsence")
	}

	var r0 *model.Absence
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read, *model.Absence) (*model.Absence, error)); ok {
		return rf(ctx, read, in)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *options.Read, *model.Absence) *model.Absence); ok {
		r0 = rf(ctx, read, in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Absence)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *options.Read, *model.Absence) error); ok {
		r1 = rf(ctx, read, in)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAgentAbsenceManager_UpdateAgentAbsence_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateAgentAbsence'
type MockAgentAbsenceManager_UpdateAgentAbsence_Call struct {
	*mock.Call
}

// UpdateAgentAbsence is a helper method to define mock.On call
//   - ctx context.Context
//   - read *options.Read
//   - in *model.Absence
func (_e *MockAgentAbsenceManager_Expecter) UpdateAgentAbsence(ctx interface{}, read interface{}, in interface{}) *MockAgentAbsenceManager_UpdateAgentAbsence_Call {
	return &MockAgentAbsenceManager_UpdateAgentAbsence_Call{Call: _e.mock.On("UpdateAgentAbsence", ctx, read, in)}
}

func (_c *MockAgentAbsenceManager_UpdateAgentAbsence_Call) Run(run func(ctx context.Context, read *options.Read, in *model.Absence)) *MockAgentAbsenceManager_UpdateAgentAbsence_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*options.Read), args[2].(*model.Absence))
	})
	return _c
}

func (_c *MockAgentAbsenceManager_UpdateAgentAbsence_Call) Return(_a0 *model.Absence, _a1 error) *MockAgentAbsenceManager_UpdateAgentAbsence_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAgentAbsenceManager_UpdateAgentAbsence_Call) RunAndReturn(run func(context.Context, *options.Read, *model.Absence) (*model.Absence, error)) *MockAgentAbsenceManager_UpdateAgentAbsence_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockAgentAbsenceManager creates a new instance of MockAgentAbsenceManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAgentAbsenceManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAgentAbsenceManager {
	mock := &MockAgentAbsenceManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
