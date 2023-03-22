package store

import (
	"testing"

	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockK8sStoreConfigMap struct {
	mock.Mock
}

func (m *MockK8sStoreConfigMap) Lock(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) LockWithKey(owner, key string) error {
	args := m.Called(owner, key)
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) Unlock() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) UnlockWithKey(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) IsKeyLocked(key string) (bool, string, error) {
	args := m.Called(key)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockK8sStoreConfigMap) Patch(data map[string]string) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) Update(data map[string]string) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockK8sStoreConfigMap) Get() (map[string]string, error) {
	args := m.Called()
	return args[0].(map[string]string), args.Error(1)
}

func (m *MockK8sStoreConfigMap) Delete() error {
	args := m.Called()
	return args.Error(0)
}

func TestPutRetrySucceced(t *testing.T) {
	configMapMock := MockK8sStoreConfigMap{}
	store := k8sStore{cm: &configMapMock}

	dataDriver := make(map[string]*DriveSet)

	configMapMock.On("Patch", mock.Anything).Return(nil)

	err := store.Put(dataDriver)
	assert.Empty(t, err)
	configMapMock.AssertNumberOfCalls(t, "Patch", 1)
}

func TestPutRetryFailed(t *testing.T) {
	configMapMock := MockK8sStoreConfigMap{}
	store := k8sStore{cm: &configMapMock}

	dataDriver := make(map[string]*DriveSet)

	// random etcdserver Error that doesn't need retry
	configMapMock.On("Patch", mock.Anything).Return(rpctypes.ErrNoSpace)

	err := store.Put(dataDriver)
	assert.NotEmpty(t, err)
	// should only call it once
	configMapMock.AssertNumberOfCalls(t, "Patch", 1)
}

func TestPutRetryLeaderChangedFailed(t *testing.T) {
	configMapMock := MockK8sStoreConfigMap{}
	store := k8sStore{cm: &configMapMock}

	dataDriver := make(map[string]*DriveSet)

	configMapMock.On("Patch", mock.Anything).Return(etcdErrorsToRetry[0])

	err := store.Put(dataDriver)
	assert.NotEmpty(t, err)
	configMapMock.AssertNumberOfCalls(t, "Patch", waitSteps)
}

func TestPutRetryLeaderChangedSucceced(t *testing.T) {
	configMapMock := MockK8sStoreConfigMap{}
	store := k8sStore{cm: &configMapMock}

	dataDriver := make(map[string]*DriveSet)

	configMapMock.On("Patch", mock.Anything).Return(etcdErrorsToRetry[0]).Once()
	configMapMock.On("Patch", mock.Anything).Return(nil)

	err := store.Put(dataDriver)
	assert.Empty(t, err)
	configMapMock.AssertNumberOfCalls(t, "Patch", 2)
}
