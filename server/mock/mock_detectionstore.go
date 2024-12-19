// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/security-onion-solutions/securityonion-soc/server (interfaces: Detectionstore)
//
// Generated by this command:
//
//	mockgen -destination mock/mock_detectionstore.go -package mock . Detectionstore
//
// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	model "github.com/security-onion-solutions/securityonion-soc/model"
	gomock "go.uber.org/mock/gomock"
)

// MockDetectionstore is a mock of Detectionstore interface.
type MockDetectionstore struct {
	ctrl     *gomock.Controller
	recorder *MockDetectionstoreMockRecorder
}

// MockDetectionstoreMockRecorder is the mock recorder for MockDetectionstore.
type MockDetectionstoreMockRecorder struct {
	mock *MockDetectionstore
}

// NewMockDetectionstore creates a new mock instance.
func NewMockDetectionstore(ctrl *gomock.Controller) *MockDetectionstore {
	mock := &MockDetectionstore{ctrl: ctrl}
	mock.recorder = &MockDetectionstoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDetectionstore) EXPECT() *MockDetectionstoreMockRecorder {
	return m.recorder
}

// CreateComment mocks base method.
func (m *MockDetectionstore) CreateComment(arg0 context.Context, arg1 *model.DetectionComment) (*model.DetectionComment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateComment", arg0, arg1)
	ret0, _ := ret[0].(*model.DetectionComment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateComment indicates an expected call of CreateComment.
func (mr *MockDetectionstoreMockRecorder) CreateComment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateComment", reflect.TypeOf((*MockDetectionstore)(nil).CreateComment), arg0, arg1)
}

// CreateDetection mocks base method.
func (m *MockDetectionstore) CreateDetection(arg0 context.Context, arg1 *model.Detection) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDetection", arg0, arg1)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDetection indicates an expected call of CreateDetection.
func (mr *MockDetectionstoreMockRecorder) CreateDetection(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDetection", reflect.TypeOf((*MockDetectionstore)(nil).CreateDetection), arg0, arg1)
}

// DeleteComment mocks base method.
func (m *MockDetectionstore) DeleteComment(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteComment", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteComment indicates an expected call of DeleteComment.
func (mr *MockDetectionstoreMockRecorder) DeleteComment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteComment", reflect.TypeOf((*MockDetectionstore)(nil).DeleteComment), arg0, arg1)
}

// DeleteDetection mocks base method.
func (m *MockDetectionstore) DeleteDetection(arg0 context.Context, arg1 string) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDetection", arg0, arg1)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteDetection indicates an expected call of DeleteDetection.
func (mr *MockDetectionstoreMockRecorder) DeleteDetection(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDetection", reflect.TypeOf((*MockDetectionstore)(nil).DeleteDetection), arg0, arg1)
}

// DoesTemplateExist mocks base method.
func (m *MockDetectionstore) DoesTemplateExist(arg0 context.Context, arg1 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DoesTemplateExist", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DoesTemplateExist indicates an expected call of DoesTemplateExist.
func (mr *MockDetectionstoreMockRecorder) DoesTemplateExist(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DoesTemplateExist", reflect.TypeOf((*MockDetectionstore)(nil).DoesTemplateExist), arg0, arg1)
}

// GetAllDetections mocks base method.
func (m *MockDetectionstore) GetAllDetections(arg0 context.Context, arg1 ...model.GetAllOption) (map[string]*model.Detection, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetAllDetections", varargs...)
	ret0, _ := ret[0].(map[string]*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllDetections indicates an expected call of GetAllDetections.
func (mr *MockDetectionstoreMockRecorder) GetAllDetections(arg0 any, arg1 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllDetections", reflect.TypeOf((*MockDetectionstore)(nil).GetAllDetections), varargs...)
}

// GetComment mocks base method.
func (m *MockDetectionstore) GetComment(arg0 context.Context, arg1 string) (*model.DetectionComment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetComment", arg0, arg1)
	ret0, _ := ret[0].(*model.DetectionComment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetComment indicates an expected call of GetComment.
func (mr *MockDetectionstoreMockRecorder) GetComment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetComment", reflect.TypeOf((*MockDetectionstore)(nil).GetComment), arg0, arg1)
}

// GetComments mocks base method.
func (m *MockDetectionstore) GetComments(arg0 context.Context, arg1 string) ([]*model.DetectionComment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetComments", arg0, arg1)
	ret0, _ := ret[0].([]*model.DetectionComment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetComments indicates an expected call of GetComments.
func (mr *MockDetectionstoreMockRecorder) GetComments(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetComments", reflect.TypeOf((*MockDetectionstore)(nil).GetComments), arg0, arg1)
}

// GetDetection mocks base method.
func (m *MockDetectionstore) GetDetection(arg0 context.Context, arg1 string) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDetection", arg0, arg1)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDetection indicates an expected call of GetDetection.
func (mr *MockDetectionstoreMockRecorder) GetDetection(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDetection", reflect.TypeOf((*MockDetectionstore)(nil).GetDetection), arg0, arg1)
}

// GetDetectionByPublicId mocks base method.
func (m *MockDetectionstore) GetDetectionByPublicId(arg0 context.Context, arg1 string) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDetectionByPublicId", arg0, arg1)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDetectionByPublicId indicates an expected call of GetDetectionByPublicId.
func (mr *MockDetectionstoreMockRecorder) GetDetectionByPublicId(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDetectionByPublicId", reflect.TypeOf((*MockDetectionstore)(nil).GetDetectionByPublicId), arg0, arg1)
}

// GetDetectionHistory mocks base method.
func (m *MockDetectionstore) GetDetectionHistory(arg0 context.Context, arg1 string) ([]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDetectionHistory", arg0, arg1)
	ret0, _ := ret[0].([]any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDetectionHistory indicates an expected call of GetDetectionHistory.
func (mr *MockDetectionstoreMockRecorder) GetDetectionHistory(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDetectionHistory", reflect.TypeOf((*MockDetectionstore)(nil).GetDetectionHistory), arg0, arg1)
}

// Query mocks base method.
func (m *MockDetectionstore) Query(arg0 context.Context, arg1 string, arg2 int) ([]any, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", arg0, arg1, arg2)
	ret0, _ := ret[0].([]any)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Query indicates an expected call of Query.
func (mr *MockDetectionstoreMockRecorder) Query(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockDetectionstore)(nil).Query), arg0, arg1, arg2)
}

// UpdateComment mocks base method.
func (m *MockDetectionstore) UpdateComment(arg0 context.Context, arg1 *model.DetectionComment) (*model.DetectionComment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateComment", arg0, arg1)
	ret0, _ := ret[0].(*model.DetectionComment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateComment indicates an expected call of UpdateComment.
func (mr *MockDetectionstoreMockRecorder) UpdateComment(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateComment", reflect.TypeOf((*MockDetectionstore)(nil).UpdateComment), arg0, arg1)
}

// UpdateDetection mocks base method.
func (m *MockDetectionstore) UpdateDetection(arg0 context.Context, arg1 *model.Detection) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDetection", arg0, arg1)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDetection indicates an expected call of UpdateDetection.
func (mr *MockDetectionstoreMockRecorder) UpdateDetection(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDetection", reflect.TypeOf((*MockDetectionstore)(nil).UpdateDetection), arg0, arg1)
}

// UpdateDetectionField mocks base method.
func (m *MockDetectionstore) UpdateDetectionField(arg0 context.Context, arg1 string, arg2 map[string]any) (*model.Detection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDetectionField", arg0, arg1, arg2)
	ret0, _ := ret[0].(*model.Detection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDetectionField indicates an expected call of UpdateDetectionField.
func (mr *MockDetectionstoreMockRecorder) UpdateDetectionField(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDetectionField", reflect.TypeOf((*MockDetectionstore)(nil).UpdateDetectionField), arg0, arg1, arg2)
}