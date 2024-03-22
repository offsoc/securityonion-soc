// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/security-onion-solutions/securityonion-soc/server/modules/elastalert (interfaces: IOManager)
//
// Generated by this command:
//
//	mockgen -destination mock/mock_iomanager.go -package mock . IOManager
//
// Package mock is a generated GoMock package.
package mock

import (
	fs "io/fs"
	http "net/http"
	exec "os/exec"
	reflect "reflect"
	time "time"

	gomock "go.uber.org/mock/gomock"
)

// MockIOManager is a mock of IOManager interface.
type MockIOManager struct {
	ctrl     *gomock.Controller
	recorder *MockIOManagerMockRecorder
}

// MockIOManagerMockRecorder is the mock recorder for MockIOManager.
type MockIOManagerMockRecorder struct {
	mock *MockIOManager
}

// NewMockIOManager creates a new mock instance.
func NewMockIOManager(ctrl *gomock.Controller) *MockIOManager {
	mock := &MockIOManager{ctrl: ctrl}
	mock.recorder = &MockIOManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIOManager) EXPECT() *MockIOManagerMockRecorder {
	return m.recorder
}

// DeleteFile mocks base method.
func (m *MockIOManager) DeleteFile(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *MockIOManagerMockRecorder) DeleteFile(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*MockIOManager)(nil).DeleteFile), arg0)
}

// ExecCommand mocks base method.
func (m *MockIOManager) ExecCommand(arg0 *exec.Cmd) ([]byte, int, time.Duration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecCommand", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(time.Duration)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ExecCommand indicates an expected call of ExecCommand.
func (mr *MockIOManagerMockRecorder) ExecCommand(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecCommand", reflect.TypeOf((*MockIOManager)(nil).ExecCommand), arg0)
}

// MakeRequest mocks base method.
func (m *MockIOManager) MakeRequest(arg0 *http.Request) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeRequest", arg0)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MakeRequest indicates an expected call of MakeRequest.
func (mr *MockIOManagerMockRecorder) MakeRequest(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeRequest", reflect.TypeOf((*MockIOManager)(nil).MakeRequest), arg0)
}

// ReadDir mocks base method.
func (m *MockIOManager) ReadDir(arg0 string) ([]fs.DirEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadDir", arg0)
	ret0, _ := ret[0].([]fs.DirEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadDir indicates an expected call of ReadDir.
func (mr *MockIOManagerMockRecorder) ReadDir(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadDir", reflect.TypeOf((*MockIOManager)(nil).ReadDir), arg0)
}

// ReadFile mocks base method.
func (m *MockIOManager) ReadFile(arg0 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFile", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFile indicates an expected call of ReadFile.
func (mr *MockIOManagerMockRecorder) ReadFile(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFile", reflect.TypeOf((*MockIOManager)(nil).ReadFile), arg0)
}

// WalkDir mocks base method.
func (m *MockIOManager) WalkDir(arg0 string, arg1 fs.WalkDirFunc) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WalkDir", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// WalkDir indicates an expected call of WalkDir.
func (mr *MockIOManagerMockRecorder) WalkDir(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WalkDir", reflect.TypeOf((*MockIOManager)(nil).WalkDir), arg0, arg1)
}

// WriteFile mocks base method.
func (m *MockIOManager) WriteFile(arg0 string, arg1 []byte, arg2 fs.FileMode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteFile indicates an expected call of WriteFile.
func (mr *MockIOManagerMockRecorder) WriteFile(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteFile", reflect.TypeOf((*MockIOManager)(nil).WriteFile), arg0, arg1, arg2)
}
