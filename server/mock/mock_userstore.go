// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/security-onion-solutions/securityonion-soc/server (interfaces: Userstore)
//
// Generated by this command:
//
//	mockgen -destination mock/mock_userstore.go -package mock . Userstore
//
// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	model "github.com/security-onion-solutions/securityonion-soc/model"
	gomock "go.uber.org/mock/gomock"
)

// MockUserstore is a mock of Userstore interface.
type MockUserstore struct {
	ctrl     *gomock.Controller
	recorder *MockUserstoreMockRecorder
}

// MockUserstoreMockRecorder is the mock recorder for MockUserstore.
type MockUserstoreMockRecorder struct {
	mock *MockUserstore
}

// NewMockUserstore creates a new mock instance.
func NewMockUserstore(ctrl *gomock.Controller) *MockUserstore {
	mock := &MockUserstore{ctrl: ctrl}
	mock.recorder = &MockUserstoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserstore) EXPECT() *MockUserstoreMockRecorder {
	return m.recorder
}

// GetUserById mocks base method.
func (m *MockUserstore) GetUserById(arg0 context.Context, arg1 string) (*model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserById", arg0, arg1)
	ret0, _ := ret[0].(*model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserById indicates an expected call of GetUserById.
func (mr *MockUserstoreMockRecorder) GetUserById(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserById", reflect.TypeOf((*MockUserstore)(nil).GetUserById), arg0, arg1)
}

// GetUsers mocks base method.
func (m *MockUserstore) GetUsers(arg0 context.Context) ([]*model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsers", arg0)
	ret0, _ := ret[0].([]*model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUsers indicates an expected call of GetUsers.
func (mr *MockUserstoreMockRecorder) GetUsers(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsers", reflect.TypeOf((*MockUserstore)(nil).GetUsers), arg0)
}