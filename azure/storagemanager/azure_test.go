package storagemanager

import (
	"reflect"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/parser"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	testSpecPath = "testspecs/azure-storage-decision-matrix.yaml"
)

func getMatrixFromYaml(t *testing.T) *cloudops.StorageDecisionMatrix {
	decisionMatrix, err := parser.NewStorageDecisionMatrixParser().UnmarshalFromYaml(testSpecPath)
	require.NoError(t, err, "Unexpected error on yaml parser")
	return decisionMatrix
}

func TestAzureStorageDistribution(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	decisionMatrix := getMatrixFromYaml(t)
	azureManager, err := NewAzureStorageManager(*decisionMatrix)
	require.NoError(t, err, "Unexpected error on creating azure storage manager")

	// Test1: always use the upper bound on IOPS if there is no drive type
	// that provides that exact amount of requested IOPS")

	request := &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        1000,
				MinCapacity: 1024,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        2,
	}
	response, err := azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse := &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 256,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 2,
				DriveCount:       1,
				IOPS:             1100,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test2: choose the right size of the disk by updating the instances per zone
	//        in case of a conflict with two configurations providing the same IOPS
	//        and min capacity choose based of priority
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        500,
				MinCapacity: 1000,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        3,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 333,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 1,
				DriveCount:       1,
				IOPS:             1100,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test3: user wants 1TiB on all the nodes
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        5000,
				MinCapacity: 9216,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        3,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 1024,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 3,
				DriveCount:       1,
				IOPS:             5000,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test4: choose the configuration which is closest to the requested IOPS
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        2000,
				MinCapacity: 16384,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 2,
		ZoneCount:        2,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 4096,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 2,
				DriveCount:       1,
				IOPS:             2300,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test5: choose upper bound IOPS when you cannot uniformly distribute storage
	// across nodes for the provided IOPS
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        2000,
				MinCapacity: 16384,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 2,
		ZoneCount:        3,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 2730,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 2,
				DriveCount:       1,
				IOPS:             2300,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test6: reduce the number of instances per zone if the IOPS and min capacity are not met
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        7500,
				MinCapacity: 4096,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 2,
		ZoneCount:        2,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 2048,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 1,
				DriveCount:       1,
				IOPS:             7500,
			},
		},
	}
	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test7: if storage cannot be distributed equally across zones return an error
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        7500,
				MinCapacity: 2048,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        3,
	}
	response, err = azureManager.GetStorageDistribution(request)
	require.EqualError(t, cloudops.ErrStorageDistributionCandidateNotFound, err.Error(), "Unexpected error")
	require.Nil(t, response, "Expected a nil response")

	// Test8: Multiple user storage specs in a single request
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        500,
				MinCapacity: 1000,
				MaxCapacity: 100000,
			},
			&cloudops.StorageSpec{
				IOPS:        5000,
				MinCapacity: 9216,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        3,
	}

	response, err = azureManager.GetStorageDistribution(request)
	require.NoError(t, err, "Unexpected error on GetStorageDistribution")

	expectedResponse = &cloudops.StorageDistributionResponse{
		InstanceStorage: []*cloudops.StoragePoolSpec{
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 333,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 1,
				DriveCount:       1,
				IOPS:             1100,
			},
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: 1024,
				DriveType:        "Premium_LRS",
				InstancesPerZone: 3,
				DriveCount:       1,
				IOPS:             5000,
			},
		},
	}

	require.True(t, reflect.DeepEqual(*response, *expectedResponse),
		"Expected Response: %+v . Actual Response %+v",
		expectedResponse.InstanceStorage[0], response.InstanceStorage[0])

	// Test9: Fail the request even if one of the user specs fails
	request = &cloudops.StorageDistributionRequest{
		UserStorageSpec: []*cloudops.StorageSpec{
			&cloudops.StorageSpec{
				IOPS:        500,
				MinCapacity: 1000,
				MaxCapacity: 100000,
			},
			&cloudops.StorageSpec{
				IOPS:        7500,
				MinCapacity: 2048,
				MaxCapacity: 100000,
			},
		},
		InstanceType:     "foo",
		InstancesPerZone: 3,
		ZoneCount:        3,
	}

	response, err = azureManager.GetStorageDistribution(request)
	require.EqualError(t, cloudops.ErrStorageDistributionCandidateNotFound, err.Error(), "Unexpected error")
	require.Nil(t, response, "Expected a nil response")

}
