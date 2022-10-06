package storagemanager

import (
	"fmt"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/storagedistribution"
	"github.com/libopenstorage/cloudops/unsupported"
)

type oracleStorageManager struct {
	cloudops.StorageManager
	decisionMatrix *cloudops.StorageDecisionMatrix
}

// NewStorageManager returns a Oracle specific implementation of StorageManager interface.
func NewStorageManager(decisionMatrix cloudops.StorageDecisionMatrix) (cloudops.StorageManager, error) {
	return &oracleStorageManager{
		StorageManager: unsupported.NewUnsupportedStorageManager(),
		decisionMatrix: &decisionMatrix}, nil
}

func (o *oracleStorageManager) GetStorageDistribution(
	request *cloudops.StorageDistributionRequest,
) (*cloudops.StorageDistributionResponse, error) {
	response := &cloudops.StorageDistributionResponse{}
	var currentDriveType string
	for _, userRequest := range request.UserStorageSpec {
		currentDriveType = userRequest.DriveType
		// for request, find how many instances per zone needs to have storage
		// and the storage spec for each of them
		instStorage, instancePerZone, row, err :=
			storagedistribution.GetStorageDistributionForPool(
				o.decisionMatrix,
				userRequest,
				request.InstancesPerZone,
				request.ZoneCount,
			)
		if err != nil {
			return nil, err
		}
		if currentDriveType == "" {
			currentDriveType = instStorage.DriveType
		}
		response.InstanceStorage = append(
			response.InstanceStorage,
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: instStorage.DriveCapacityGiB,
				DriveType:        currentDriveType,
				InstancesPerZone: instancePerZone,
				DriveCount:       instStorage.DriveCount,
				IOPS:             determineIOPSForPool(instStorage, row),
			},
		)
	}
	return response, nil
}
func (o *oracleStorageManager) RecommendStoragePoolUpdate(request *cloudops.StoragePoolUpdateRequest) (*cloudops.StoragePoolUpdateResponse, error) {
	resp, row, err := storagedistribution.GetStorageUpdateConfig(request, o.decisionMatrix)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.InstanceStorage) != 1 {
		return nil, fmt.Errorf("could not find a valid instance storage object")
	}
	resp.InstanceStorage[0].IOPS = determineIOPSForPool(resp.InstanceStorage[0], row)
	if request.CurrentDriveType != "" {
		resp.InstanceStorage[0].DriveType = request.CurrentDriveType
	}
	return resp, nil
}

func determineIOPSForPool(instStorage *cloudops.StoragePoolSpec, row *cloudops.StorageDecisionMatrixRow) uint64 {
	var iopsPerGB, maxIopsPerVol int64
	switch row.DriveType {
	case "0_vpus":
		iopsPerGB = 2
		maxIopsPerVol = 3000
	case "10_vpus":
		iopsPerGB = 60
		maxIopsPerVol = 25000
	case "20_vpus":
		iopsPerGB = 75
		maxIopsPerVol = 50000
	case "30_vpus":
		iopsPerGB = 90
		maxIopsPerVol = 75000
	case "40_vpus":
		iopsPerGB = 105
		maxIopsPerVol = 100000
	case "50_vpus":
		iopsPerGB = 120
		maxIopsPerVol = 125000
	case "60_vpus":
		iopsPerGB = 135
		maxIopsPerVol = 150000
	case "70_vpus":
		iopsPerGB = 150
		maxIopsPerVol = 175000
	case "80_vpus":
		iopsPerGB = 165
		maxIopsPerVol = 200000
	case "90_vpus":
		iopsPerGB = 180
		maxIopsPerVol = 225000
	case "100_vpus":
		iopsPerGB = 195
		maxIopsPerVol = 250000
	case "110_vpus":
		iopsPerGB = 210
		maxIopsPerVol = 275000
	case "120_vpus":
		iopsPerGB = 225
		maxIopsPerVol = 300000
	}

	if instStorage.DriveCapacityGiB*uint64(iopsPerGB) > uint64(maxIopsPerVol) {
		return uint64(maxIopsPerVol)
	}
	return instStorage.DriveCapacityGiB * uint64(iopsPerGB)
}

func init() {
	cloudops.RegisterStorageManager(cloudops.Oracle, NewStorageManager)
}
