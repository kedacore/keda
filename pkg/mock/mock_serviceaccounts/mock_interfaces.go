// Generated from these commands and then edited:
//
//	mockgen -source=k8s.io/client-go/kubernetes/typed/core/v1/serviceaccount.go -imports=k8s.io/client-go/kubernetes/typed/core/v1/core_client.go
//	mockgen k8s.io/client-go/kubernetes/typed/core/v1 CoreV1Interface
//
// Package mock_v1 is a generated GoMock package from various generated sources and edited to remove unnecessary code.
//

package mock_v1 //nolint:revive

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	v10 "k8s.io/api/authentication/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// MockCoreV1Interface is a mock of CoreV1Interface interface.
type MockCoreV1Interface struct {
	v1.CoreV1Interface
	mockServiceAccount *MockServiceAccountInterface
	ctrl               *gomock.Controller
	recorder           *MockCoreV1InterfaceMockRecorder
}

// MockCoreV1InterfaceMockRecorder is the mock recorder for MockCoreV1Interface.
type MockCoreV1InterfaceMockRecorder struct {
	mock *MockCoreV1Interface
}

// NewMockCoreV1Interface creates a new mock instance.
func NewMockCoreV1Interface(ctrl *gomock.Controller) *MockCoreV1Interface {
	mock := &MockCoreV1Interface{ctrl: ctrl}
	mock.mockServiceAccount = NewMockServiceAccountInterface(ctrl)
	mock.recorder = &MockCoreV1InterfaceMockRecorder{mock}
	return mock
}

// GetServiceAccountInterface returns the mock for ServiceAccountInterface.
func (m *MockCoreV1Interface) GetServiceAccountInterface() *MockServiceAccountInterface {
	return m.mockServiceAccount
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCoreV1Interface) EXPECT() *MockCoreV1InterfaceMockRecorder {
	return m.recorder
}

// ServiceAccounts mocks base method.
func (m *MockCoreV1Interface) ServiceAccounts(_ string) v1.ServiceAccountInterface {
	return m.mockServiceAccount
}

// MockServiceAccountInterface is a mock of ServiceAccountInterface interface.
type MockServiceAccountInterface struct {
	v1.ServiceAccountInterface
	ctrl     *gomock.Controller
	recorder *MockServiceAccountInterfaceMockRecorder
}

// MockServiceAccountInterfaceMockRecorder is the mock recorder for MockServiceAccountInterface.
type MockServiceAccountInterfaceMockRecorder struct {
	mock *MockServiceAccountInterface
}

// NewMockServiceAccountInterface creates a new mock instance.
func NewMockServiceAccountInterface(ctrl *gomock.Controller) *MockServiceAccountInterface {
	mock := &MockServiceAccountInterface{ctrl: ctrl}
	mock.recorder = &MockServiceAccountInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceAccountInterface) EXPECT() *MockServiceAccountInterfaceMockRecorder {
	return m.recorder
}

// CreateToken mocks base method.
func (m *MockServiceAccountInterface) CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *v10.TokenRequest, opts v12.CreateOptions) (*v10.TokenRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateToken", ctx, serviceAccountName, tokenRequest, opts)
	ret0, _ := ret[0].(*v10.TokenRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateToken indicates an expected call of CreateToken.
func (mr *MockServiceAccountInterfaceMockRecorder) CreateToken(ctx, serviceAccountName, tokenRequest, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateToken", reflect.TypeOf((*MockServiceAccountInterface)(nil).CreateToken), ctx, serviceAccountName, tokenRequest, opts)
}
