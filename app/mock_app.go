package app

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/green-element-chain/gelchain/types"
	"reflect"
)

// MockIApp is a mock of IApp interface
type MockIApp struct {
	ctrl     *gomock.Controller
	recorder *MockIAppMockRecorder
}

// MockIAppMockRecorder is the mock recorder for MockIApp
type MockIAppMockRecorder struct {
	mock *MockIApp
}

// NewMockIApp creates a new mock instance
func NewMockIApp(ctrl *gomock.Controller) *MockIApp {
	mock := &MockIApp{ctrl: ctrl}
	mock.recorder = &MockIAppMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIApp) EXPECT() *MockIAppMockRecorder {
	return m.recorder
}

// GetAccountMap mocks base method
func (m *MockIApp) GetAccountMap(arg0 string) *types.AccountMap {
	ret := m.ctrl.Call(m, "GetAccountMap", arg0)
	ret0, _ := ret[0].(*types.AccountMap)
	return ret0
}

// GetAccountMap indicates an expected call of GetAccountMap
func (mr *MockIAppMockRecorder) GetAccountMap(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountMap", reflect.TypeOf((*MockIApp)(nil).GetAccountMap), arg0)
}

// RemoveValidatorTx mocks base method
func (m *MockIApp) RemoveValidatorTx(arg0 common.Address) (bool, error) {
	ret := m.ctrl.Call(m, "RemoveValidatorTx", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RemoveValidatorTx indicates an expected call of RemoveValidatorTx
func (mr *MockIAppMockRecorder) RemoveValidatorTx(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveValidatorTx", reflect.TypeOf((*MockIApp)(nil).RemoveValidatorTx), arg0)
}
