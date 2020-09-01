// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package gcpclient is a generated GoMock package.
package gcpclient

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	iam "google.golang.org/api/iam/v1"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// GetServiceAccount mocks base method
func (m *MockClient) GetServiceAccount(accountName string) (*iam.ServiceAccount, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceAccount", accountName)
	ret0, _ := ret[0].(*iam.ServiceAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServiceAccount indicates an expected call of GetServiceAccount
func (mr *MockClientMockRecorder) GetServiceAccount(accountName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceAccount", reflect.TypeOf((*MockClient)(nil).GetServiceAccount), accountName)
}

// CreateServiceAccount mocks base method
func (m *MockClient) CreateServiceAccount(name, displayName string) (*iam.ServiceAccount, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServiceAccount", name, displayName)
	ret0, _ := ret[0].(*iam.ServiceAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateServiceAccount indicates an expected call of CreateServiceAccount
func (mr *MockClientMockRecorder) CreateServiceAccount(name, displayName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServiceAccount", reflect.TypeOf((*MockClient)(nil).CreateServiceAccount), name, displayName)
}

// DeleteServiceAccount mocks base method
func (m *MockClient) DeleteServiceAccount(accountEmail string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServiceAccount", accountEmail)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServiceAccount indicates an expected call of DeleteServiceAccount
func (mr *MockClientMockRecorder) DeleteServiceAccount(accountEmail interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServiceAccount", reflect.TypeOf((*MockClient)(nil).DeleteServiceAccount), accountEmail)
}

// CreateServiceAccountKey mocks base method
func (m *MockClient) CreateServiceAccountKey(serviceAccountEmail string) (*iam.ServiceAccountKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServiceAccountKey", serviceAccountEmail)
	ret0, _ := ret[0].(*iam.ServiceAccountKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateServiceAccountKey indicates an expected call of CreateServiceAccountKey
func (mr *MockClientMockRecorder) CreateServiceAccountKey(serviceAccountEmail interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServiceAccountKey", reflect.TypeOf((*MockClient)(nil).CreateServiceAccountKey), serviceAccountEmail)
}

// DeleteServiceAccountKeys mocks base method
func (m *MockClient) DeleteServiceAccountKeys(serviceAccountEmail string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServiceAccountKeys", serviceAccountEmail)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServiceAccountKeys indicates an expected call of DeleteServiceAccountKeys
func (mr *MockClientMockRecorder) DeleteServiceAccountKeys(serviceAccountEmail interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServiceAccountKeys", reflect.TypeOf((*MockClient)(nil).DeleteServiceAccountKeys), serviceAccountEmail)
}

// GetIamPolicy mocks base method
func (m *MockClient) GetIamPolicy(projectName string) (*cloudresourcemanager.Policy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIamPolicy", projectName)
	ret0, _ := ret[0].(*cloudresourcemanager.Policy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIamPolicy indicates an expected call of GetIamPolicy
func (mr *MockClientMockRecorder) GetIamPolicy(projectName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIamPolicy", reflect.TypeOf((*MockClient)(nil).GetIamPolicy), projectName)
}

// SetIamPolicy mocks base method
func (m *MockClient) SetIamPolicy(setIamPolicyRequest *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetIamPolicy", setIamPolicyRequest)
	ret0, _ := ret[0].(*cloudresourcemanager.Policy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetIamPolicy indicates an expected call of SetIamPolicy
func (mr *MockClientMockRecorder) SetIamPolicy(setIamPolicyRequest interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetIamPolicy", reflect.TypeOf((*MockClient)(nil).SetIamPolicy), setIamPolicyRequest)
}

// ListProjects mocks base method
func (m *MockClient) ListProjects() ([]*cloudresourcemanager.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProjects")
	ret0, _ := ret[0].([]*cloudresourcemanager.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListProjects indicates an expected call of ListProjects
func (mr *MockClientMockRecorder) ListProjects() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProjects", reflect.TypeOf((*MockClient)(nil).ListProjects))
}

// CreateProject mocks base method
func (m *MockClient) CreateProject(parentFolder string) (*cloudresourcemanager.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateProject", parentFolder)
	ret0, _ := ret[0].(*cloudresourcemanager.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateProject indicates an expected call of CreateProject
func (mr *MockClientMockRecorder) CreateProject(parentFolder interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateProject", reflect.TypeOf((*MockClient)(nil).CreateProject), parentFolder)
}

// DeleteProject mocks base method
func (m *MockClient) DeleteProject(parentFolder string) (*cloudresourcemanager.Empty, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteProject", parentFolder)
	ret0, _ := ret[0].(*cloudresourcemanager.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteProject indicates an expected call of DeleteProject
func (mr *MockClientMockRecorder) DeleteProject(parentFolder interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteProject", reflect.TypeOf((*MockClient)(nil).DeleteProject), parentFolder)
}

// GetProject mocks base method
func (m *MockClient) GetProject(projectID string) (*cloudresourcemanager.Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProject", projectID)
	ret0, _ := ret[0].(*cloudresourcemanager.Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProject indicates an expected call of GetProject
func (mr *MockClientMockRecorder) GetProject(projectID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProject", reflect.TypeOf((*MockClient)(nil).GetProject), projectID)
}

// EnableAPI mocks base method
func (m *MockClient) EnableAPI(projectID, api string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnableAPI", projectID, api)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnableAPI indicates an expected call of EnableAPI
func (mr *MockClientMockRecorder) EnableAPI(projectID, api interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnableAPI", reflect.TypeOf((*MockClient)(nil).EnableAPI), projectID, api)
}

// ListAPIs mocks base method
func (m *MockClient) ListAPIs(projectID string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAPIs", projectID)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAPIs indicates an expected call of ListAPIs
func (mr *MockClientMockRecorder) ListAPIs(projectID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAPIs", reflect.TypeOf((*MockClient)(nil).ListAPIs), projectID)
}

// CreateCloudBillingAccount mocks base method
func (m *MockClient) CreateCloudBillingAccount(projectID, billingAccount string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCloudBillingAccount", projectID, billingAccount)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateCloudBillingAccount indicates an expected call of CreateCloudBillingAccount
func (mr *MockClientMockRecorder) CreateCloudBillingAccount(projectID, billingAccount interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCloudBillingAccount", reflect.TypeOf((*MockClient)(nil).CreateCloudBillingAccount), projectID, billingAccount)
}

// ListAvailabilityZones mocks base method
func (m *MockClient) ListAvailabilityZones(projectID, region string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAvailabilityZones", projectID, region)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAvailabilityZones indicates an expected call of ListAvailabilityZones
func (mr *MockClientMockRecorder) ListAvailabilityZones(projectID, region interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAvailabilityZones", reflect.TypeOf((*MockClient)(nil).ListAvailabilityZones), projectID, region)
}