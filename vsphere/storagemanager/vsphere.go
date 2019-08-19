package storagemanager

import (
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/storagedistribution"
	"github.com/libopenstorage/cloudops/unsupported"
)

type vsphereStorageManager struct {
	cloudops.StorageManager
	decisionMatrix *cloudops.StorageDecisionMatrix
}

// newVsphereStorageManager returns an vsphere implementation for Storage Management
func newVsphereStorageManager(
	decisionMatrix cloudops.StorageDecisionMatrix,
) (cloudops.StorageManager, error) {
	return &vsphereStorageManager{
		StorageManager: unsupported.NewUnsupportedStorageManager(),
		decisionMatrix: &decisionMatrix}, nil
}

func (a *vsphereStorageManager) GetStorageDistribution(
	request *cloudops.StorageDistributionRequest,
) (*cloudops.StorageDistributionResponse, error) {
	return storagedistribution.GetStorageDistribution(request, a.decisionMatrix)
}

func (a *vsphereStorageManager) RecommendInstanceStorageUpdate(
	request *cloudops.StorageUpdateRequest) (*cloudops.StorageUpdateResponse, error) {
	return storagedistribution.GetStorageUpdateConfig(request, a.decisionMatrix)
}
func init() {
	cloudops.RegisterStorageManager(cloudops.Vsphere, newVsphereStorageManager)
}
