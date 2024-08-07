// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/libopenstorage/cloudops (interfaces: Ops)

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	cloudops "github.com/libopenstorage/cloudops"
	reflect "reflect"
	time "time"
)

// MockOps is a mock of Ops interface
type MockOps struct {
	ctrl     *gomock.Controller
	recorder *MockOpsMockRecorder
}

// MockOpsMockRecorder is the mock recorder for MockOps
type MockOpsMockRecorder struct {
	mock *MockOps
}

// NewMockOps creates a new mock instance
func NewMockOps(ctrl *gomock.Controller) *MockOps {
	mock := &MockOps{ctrl: ctrl}
	mock.recorder = &MockOpsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockOps) EXPECT() *MockOpsMockRecorder {
	return m.recorder
}

// ApplyTags mocks base method
func (m *MockOps) ApplyTags(arg0 string, arg1, arg2 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyTags", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyTags indicates an expected call of ApplyTags
func (mr *MockOpsMockRecorder) ApplyTags(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyTags", reflect.TypeOf((*MockOps)(nil).ApplyTags), arg0, arg1, arg2)
}

// AreVolumesReadyToExpand mocks base method
func (m *MockOps) AreVolumesReadyToExpand(arg0 []*string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AreVolumesReadyToExpand", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AreVolumesReadyToExpand indicates an expected call of AreVolumesReadyToExpand
func (mr *MockOpsMockRecorder) AreVolumesReadyToExpand(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AreVolumesReadyToExpand", reflect.TypeOf((*MockOps)(nil).AreVolumesReadyToExpand), arg0)
}

// Attach mocks base method
func (m *MockOps) Attach(arg0 string, arg1 map[string]string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Attach", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Attach indicates an expected call of Attach
func (mr *MockOpsMockRecorder) Attach(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Attach", reflect.TypeOf((*MockOps)(nil).Attach), arg0, arg1)
}

// Create mocks base method
func (m *MockOps) Create(arg0 interface{}, arg1, arg2 map[string]string) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1, arg2)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockOpsMockRecorder) Create(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockOps)(nil).Create), arg0, arg1, arg2)
}

// Delete mocks base method
func (m *MockOps) Delete(arg0 string, arg1 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockOpsMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockOps)(nil).Delete), arg0, arg1)
}

// DeleteFrom mocks base method
func (m *MockOps) DeleteFrom(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFrom", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFrom indicates an expected call of DeleteFrom
func (mr *MockOpsMockRecorder) DeleteFrom(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFrom", reflect.TypeOf((*MockOps)(nil).DeleteFrom), arg0, arg1)
}

// DeleteInstance mocks base method
func (m *MockOps) DeleteInstance(arg0, arg1 string, arg2 time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteInstance indicates an expected call of DeleteInstance
func (mr *MockOpsMockRecorder) DeleteInstance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteInstance", reflect.TypeOf((*MockOps)(nil).DeleteInstance), arg0, arg1, arg2)
}

// Describe mocks base method
func (m *MockOps) Describe() (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Describe")
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Describe indicates an expected call of Describe
func (mr *MockOpsMockRecorder) Describe() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Describe", reflect.TypeOf((*MockOps)(nil).Describe))
}

// Detach mocks base method
func (m *MockOps) Detach(arg0 string, arg1 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Detach", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Detach indicates an expected call of Detach
func (mr *MockOpsMockRecorder) Detach(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Detach", reflect.TypeOf((*MockOps)(nil).Detach), arg0, arg1)
}

// DetachFrom mocks base method
func (m *MockOps) DetachFrom(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DetachFrom", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DetachFrom indicates an expected call of DetachFrom
func (mr *MockOpsMockRecorder) DetachFrom(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DetachFrom", reflect.TypeOf((*MockOps)(nil).DetachFrom), arg0, arg1)
}

// DeviceMappings mocks base method
func (m *MockOps) DeviceMappings() (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeviceMappings")
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeviceMappings indicates an expected call of DeviceMappings
func (mr *MockOpsMockRecorder) DeviceMappings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeviceMappings", reflect.TypeOf((*MockOps)(nil).DeviceMappings))
}

// DevicePath mocks base method
func (m *MockOps) DevicePath(arg0 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DevicePath", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DevicePath indicates an expected call of DevicePath
func (mr *MockOpsMockRecorder) DevicePath(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DevicePath", reflect.TypeOf((*MockOps)(nil).DevicePath), arg0)
}

// Enumerate mocks base method
func (m *MockOps) Enumerate(arg0 []*string, arg1 map[string]string, arg2 string) (map[string][]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enumerate", arg0, arg1, arg2)
	ret0, _ := ret[0].(map[string][]interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Enumerate indicates an expected call of Enumerate
func (mr *MockOpsMockRecorder) Enumerate(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enumerate", reflect.TypeOf((*MockOps)(nil).Enumerate), arg0, arg1, arg2)
}

// Expand mocks base method
func (m *MockOps) Expand(arg0 string, arg1 uint64, arg2 map[string]string) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Expand", arg0, arg1, arg2)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Expand indicates an expected call of Expand
func (mr *MockOpsMockRecorder) Expand(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Expand", reflect.TypeOf((*MockOps)(nil).Expand), arg0, arg1, arg2)
}

// FreeDevices mocks base method
func (m *MockOps) FreeDevices() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FreeDevices")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FreeDevices indicates an expected call of FreeDevices
func (mr *MockOpsMockRecorder) FreeDevices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeDevices", reflect.TypeOf((*MockOps)(nil).FreeDevices))
}

// GetClusterSizeForInstance mocks base method
func (m *MockOps) GetClusterSizeForInstance(arg0 string) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterSizeForInstance", arg0)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterSizeForInstance indicates an expected call of GetClusterSizeForInstance
func (mr *MockOpsMockRecorder) GetClusterSizeForInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterSizeForInstance", reflect.TypeOf((*MockOps)(nil).GetClusterSizeForInstance), arg0)
}

// GetDeviceID mocks base method
func (m *MockOps) GetDeviceID(arg0 interface{}) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeviceID", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeviceID indicates an expected call of GetDeviceID
func (mr *MockOpsMockRecorder) GetDeviceID(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeviceID", reflect.TypeOf((*MockOps)(nil).GetDeviceID), arg0)
}

// GetInstance mocks base method
func (m *MockOps) GetInstance(arg0 string) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstance", arg0)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstance indicates an expected call of GetInstance
func (mr *MockOpsMockRecorder) GetInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstance", reflect.TypeOf((*MockOps)(nil).GetInstance), arg0)
}

// GetInstanceGroupSize mocks base method
func (m *MockOps) GetInstanceGroupSize(arg0 string) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstanceGroupSize", arg0)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstanceGroupSize indicates an expected call of GetInstanceGroupSize
func (mr *MockOpsMockRecorder) GetInstanceGroupSize(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstanceGroupSize", reflect.TypeOf((*MockOps)(nil).GetInstanceGroupSize), arg0)
}

// Inspect mocks base method
func (m *MockOps) Inspect(arg0 []*string, arg1 map[string]string) ([]interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Inspect", arg0, arg1)
	ret0, _ := ret[0].([]interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Inspect indicates an expected call of Inspect
func (mr *MockOpsMockRecorder) Inspect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Inspect", reflect.TypeOf((*MockOps)(nil).Inspect), arg0, arg1)
}

// InspectInstance mocks base method
func (m *MockOps) InspectInstance(arg0 string) (*cloudops.InstanceInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InspectInstance", arg0)
	ret0, _ := ret[0].(*cloudops.InstanceInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InspectInstance indicates an expected call of InspectInstance
func (mr *MockOpsMockRecorder) InspectInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InspectInstance", reflect.TypeOf((*MockOps)(nil).InspectInstance), arg0)
}

// InspectInstanceGroupForInstance mocks base method
func (m *MockOps) InspectInstanceGroupForInstance(arg0 string) (*cloudops.InstanceGroupInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InspectInstanceGroupForInstance", arg0)
	ret0, _ := ret[0].(*cloudops.InstanceGroupInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InspectInstanceGroupForInstance indicates an expected call of InspectInstanceGroupForInstance
func (mr *MockOpsMockRecorder) InspectInstanceGroupForInstance(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InspectInstanceGroupForInstance", reflect.TypeOf((*MockOps)(nil).InspectInstanceGroupForInstance), arg0)
}

// InstanceID mocks base method
func (m *MockOps) InstanceID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstanceID")
	ret0, _ := ret[0].(string)
	return ret0
}

// InstanceID indicates an expected call of InstanceID
func (mr *MockOpsMockRecorder) InstanceID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstanceID", reflect.TypeOf((*MockOps)(nil).InstanceID))
}

// Name mocks base method
func (m *MockOps) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockOpsMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockOps)(nil).Name))
}

// RemoveTags mocks base method
func (m *MockOps) RemoveTags(arg0 string, arg1, arg2 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveTags", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveTags indicates an expected call of RemoveTags
func (mr *MockOpsMockRecorder) RemoveTags(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveTags", reflect.TypeOf((*MockOps)(nil).RemoveTags), arg0, arg1, arg2)
}

// SetClusterVersion mocks base method
func (m *MockOps) SetClusterVersion(arg0 string, arg1 time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetClusterVersion", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetClusterVersion indicates an expected call of SetClusterVersion
func (mr *MockOpsMockRecorder) SetClusterVersion(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetClusterVersion", reflect.TypeOf((*MockOps)(nil).SetClusterVersion), arg0, arg1)
}

// SetInstanceGroupSize mocks base method
func (m *MockOps) SetInstanceGroupSize(arg0 string, arg1 int64, arg2 time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetInstanceGroupSize", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetInstanceGroupSize indicates an expected call of SetInstanceGroupSize
func (mr *MockOpsMockRecorder) SetInstanceGroupSize(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetInstanceGroupSize", reflect.TypeOf((*MockOps)(nil).SetInstanceGroupSize), arg0, arg1, arg2)
}

// SetInstanceGroupVersion mocks base method
func (m *MockOps) SetInstanceGroupVersion(arg0, arg1 string, arg2 time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetInstanceGroupVersion", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetInstanceGroupVersion indicates an expected call of SetInstanceGroupVersion
func (mr *MockOpsMockRecorder) SetInstanceGroupVersion(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetInstanceGroupVersion", reflect.TypeOf((*MockOps)(nil).SetInstanceGroupVersion), arg0, arg1, arg2)
}

// Snapshot mocks base method
func (m *MockOps) Snapshot(arg0 string, arg1 bool, arg2 map[string]string) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Snapshot", arg0, arg1, arg2)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Snapshot indicates an expected call of Snapshot
func (mr *MockOpsMockRecorder) Snapshot(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Snapshot", reflect.TypeOf((*MockOps)(nil).Snapshot), arg0, arg1, arg2)
}

// SnapshotDelete mocks base method
func (m *MockOps) SnapshotDelete(arg0 string, arg1 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SnapshotDelete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SnapshotDelete indicates an expected call of SnapshotDelete
func (mr *MockOpsMockRecorder) SnapshotDelete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SnapshotDelete", reflect.TypeOf((*MockOps)(nil).SnapshotDelete), arg0, arg1)
}

// Tags mocks base method
func (m *MockOps) Tags(arg0 string) (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tags", arg0)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Tags indicates an expected call of Tags
func (mr *MockOpsMockRecorder) Tags(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tags", reflect.TypeOf((*MockOps)(nil).Tags), arg0)
}

// SetInstanceUpgradeStrategy mocks base method
func (m *MockOps) SetInstanceUpgradeStrategy(arg0 string, arg1 string, arg2 time.Duration, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetInstanceUpgradeStrategy", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetInstanceUpgradeStrategy indicates an expected call of SetInstanceUpgradeStrategy
func (mr *MockOpsMockRecorder) SetInstanceUpgradeStrategy(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetInstanceUpgradeStrategy", reflect.TypeOf((*MockOps)(nil).SetInstanceUpgradeStrategy), arg0, arg1, arg2, arg3)
}