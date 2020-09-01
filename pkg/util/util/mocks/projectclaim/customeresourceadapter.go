// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openshift/gcp-project-operator/pkg/controller/projectclaim (interfaces: CustomResourceAdapter)

// Package projectclaim is a generated GoMock package.
package projectclaim

import (
	gomock "github.com/golang/mock/gomock"
	projectclaim "github.com/openshift/gcp-project-operator/controllers"
	util "github.com/openshift/gcp-project-operator/pkg/util"
	reflect "reflect"
)

// MockCustomResourceAdapter is a mock of CustomResourceAdapter interface
type MockCustomResourceAdapter struct {
	ctrl     *gomock.Controller
	recorder *MockCustomResourceAdapterMockRecorder
}

// MockCustomResourceAdapterMockRecorder is the mock recorder for MockCustomResourceAdapter
type MockCustomResourceAdapterMockRecorder struct {
	mock *MockCustomResourceAdapter
}

// NewMockCustomResourceAdapter creates a new mock instance
func NewMockCustomResourceAdapter(ctrl *gomock.Controller) *MockCustomResourceAdapter {
	mock := &MockCustomResourceAdapter{ctrl: ctrl}
	mock.recorder = &MockCustomResourceAdapterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCustomResourceAdapter) EXPECT() *MockCustomResourceAdapterMockRecorder {
	return m.recorder
}

// EnsureFinalizer mocks base method
func (m *MockCustomResourceAdapter) EnsureFinalizer() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureFinalizer")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureFinalizer indicates an expected call of EnsureFinalizer
func (mr *MockCustomResourceAdapterMockRecorder) EnsureFinalizer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureFinalizer", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureFinalizer))
}

// EnsureProjectClaimDeletionProcessed mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectClaimDeletionProcessed() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectClaimDeletionProcessed")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectClaimDeletionProcessed indicates an expected call of EnsureProjectClaimDeletionProcessed
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectClaimDeletionProcessed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectClaimDeletionProcessed", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectClaimDeletionProcessed))
}

// EnsureProjectClaimInitialized mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectClaimInitialized() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectClaimInitialized")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectClaimInitialized indicates an expected call of EnsureProjectClaimInitialized
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectClaimInitialized() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectClaimInitialized", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectClaimInitialized))
}

// EnsureProjectClaimStatePending mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectClaimStatePending() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectClaimStatePending")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectClaimStatePending indicates an expected call of EnsureProjectClaimStatePending
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectClaimStatePending() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectClaimStatePending", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectClaimStatePending))
}

// EnsureProjectClaimStatePendingProject mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectClaimStatePendingProject() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectClaimStatePendingProject")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectClaimStatePendingProject indicates an expected call of EnsureProjectClaimStatePendingProject
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectClaimStatePendingProject() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectClaimStatePendingProject", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectClaimStatePendingProject))
}

// EnsureProjectReferenceExists mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectReferenceExists() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectReferenceExists")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectReferenceExists indicates an expected call of EnsureProjectReferenceExists
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectReferenceExists() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectReferenceExists", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectReferenceExists))
}

// EnsureProjectReferenceLink mocks base method
func (m *MockCustomResourceAdapter) EnsureProjectReferenceLink() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureProjectReferenceLink")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureProjectReferenceLink indicates an expected call of EnsureProjectReferenceLink
func (mr *MockCustomResourceAdapterMockRecorder) EnsureProjectReferenceLink() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureProjectReferenceLink", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureProjectReferenceLink))
}

// EnsureRegionSupported mocks base method
func (m *MockCustomResourceAdapter) EnsureRegionSupported() (util.OperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureRegionSupported")
	ret0, _ := ret[0].(util.OperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnsureRegionSupported indicates an expected call of EnsureRegionSupported
func (mr *MockCustomResourceAdapterMockRecorder) EnsureRegionSupported() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureRegionSupported", reflect.TypeOf((*MockCustomResourceAdapter)(nil).EnsureRegionSupported))
}

// FinalizeProjectClaim mocks base method
func (m *MockCustomResourceAdapter) FinalizeProjectClaim() (projectclaim.ObjectState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FinalizeProjectClaim")
	ret0, _ := ret[0].(projectclaim.ObjectState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FinalizeProjectClaim indicates an expected call of FinalizeProjectClaim
func (mr *MockCustomResourceAdapterMockRecorder) FinalizeProjectClaim() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FinalizeProjectClaim", reflect.TypeOf((*MockCustomResourceAdapter)(nil).FinalizeProjectClaim))
}

// ProjectReferenceExists mocks base method
func (m *MockCustomResourceAdapter) ProjectReferenceExists() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProjectReferenceExists")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ProjectReferenceExists indicates an expected call of ProjectReferenceExists
func (mr *MockCustomResourceAdapterMockRecorder) ProjectReferenceExists() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProjectReferenceExists", reflect.TypeOf((*MockCustomResourceAdapter)(nil).ProjectReferenceExists))
}

// SetProjectClaimCondition mocks base method
func (m *MockCustomResourceAdapter) SetProjectClaimCondition(arg0 string, arg1 error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetProjectClaimCondition", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetProjectClaimCondition indicates an expected call of SetProjectClaimCondition
func (mr *MockCustomResourceAdapterMockRecorder) SetProjectClaimCondition(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetProjectClaimCondition", reflect.TypeOf((*MockCustomResourceAdapter)(nil).SetProjectClaimCondition), arg0, arg1)
}
