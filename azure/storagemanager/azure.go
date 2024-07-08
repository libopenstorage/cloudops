package storagemanager

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-08-01/compute"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/storagedistribution"
	"github.com/libopenstorage/cloudops/unsupported"
)

type azureStorageManager struct {
	cloudops.StorageManager
	decisionMatrix *cloudops.StorageDecisionMatrix
}

// NewAzureStorageManager returns an azure implementation for Storage Management
func NewAzureStorageManager(
	decisionMatrix cloudops.StorageDecisionMatrix,
) (cloudops.StorageManager, error) {
	return &azureStorageManager{
		StorageManager: unsupported.NewUnsupportedStorageManager(),
		decisionMatrix: &decisionMatrix}, nil
}

func (a *azureStorageManager) GetStorageDistribution(
	request *cloudops.StorageDistributionRequest,
) (*cloudops.StorageDistributionResponse, error) {
	response := &cloudops.StorageDistributionResponse{}
	for _, userRequest := range request.UserStorageSpec {
		// for request, find how many instances per zone needs to have storage
		// and the storage spec for each of them
		instStorage, instancePerZone, row, err :=
			storagedistribution.GetStorageDistributionForPool(
				a.decisionMatrix,
				userRequest,
				request.InstancesPerZone,
				request.ZoneCount,
			)
		if err != nil {
			return nil, err
		}
		response.InstanceStorage = append(
			response.InstanceStorage,
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: instStorage.DriveCapacityGiB,
				DriveType:        instStorage.DriveType,
				InstancesPerZone: instancePerZone,
				DriveCount:       instStorage.DriveCount,
				IOPS:             determineIOPSForPool(instStorage, row, userRequest.IOPS),
			},
		)

	}
	return response, nil
}

func (a *azureStorageManager) RecommendStoragePoolUpdate(
	request *cloudops.StoragePoolUpdateRequest) (*cloudops.StoragePoolUpdateResponse, error) {
	resp, row, err := storagedistribution.GetStorageUpdateConfig(request, a.decisionMatrix)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.InstanceStorage) != 1 {
		return nil, fmt.Errorf("could not find a valid instance storage object")
	}
	resp.InstanceStorage[0].IOPS = determineIOPSForPool(resp.InstanceStorage[0], row, request.CurrentIOPS)
	return resp, nil
}

func (a *azureStorageManager) GetMaxDriveSize(
	request *cloudops.MaxDriveSizeRequest) (*cloudops.MaxDriveSizeResponse, error) {
	resp, err := storagedistribution.GetMaxDriveSize(request, a.decisionMatrix)
	return resp, err
}

func determineIOPSForPool(instStorage *cloudops.StoragePoolSpec, row *cloudops.StorageDecisionMatrixRow, currentIOPS uint64) uint64 {
	if instStorage.DriveType == string(compute.UltraSSDLRS) || instStorage.DriveType == string(compute.PremiumV2LRS) {
		// ultra SSD LRS and Premium v2 LRS IOPS are independent of the drive size and is a configurable parameter.
		return currentIOPS
	}
	return row.MinIOPS
}

func init() {
	cloudops.RegisterStorageManager(cloudops.Azure, NewAzureStorageManager)
}
