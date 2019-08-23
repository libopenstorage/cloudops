// +build unittest

package storagemanager

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/parser"
	"github.com/libopenstorage/openstorage/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	testSpecPath = "testspecs/azure-storage-decision-matrix.yaml"
)

var (
	storageManager cloudops.StorageManager
)

func TestAzureStorageManager(t *testing.T) {
	t.Run("setup", setup)
	t.Run("storageDistribution", storageDistribution)
	t.Run("storageUpdate", storageUpdate)
}

func setup(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	decisionMatrix, err := parser.NewStorageDecisionMatrixParser().UnmarshalFromYaml(testSpecPath)
	require.NoError(t, err, "Unexpected error on yaml parser")

	storageManager, err = NewAzureStorageManager(*decisionMatrix)
	require.NoError(t, err, "Unexpected error on creating Azure storage manager")
}

func storageDistribution(t *testing.T) {
	testMatrix := []struct {
		expectedErr error
		request     *cloudops.StorageDistributionRequest
		response    *cloudops.StorageDistributionResponse
	}{
		{
			// Test1: always use the upper bound on IOPS if there is no drive type
			// that provides that exact amount of requested IOPS")
			request: &cloudops.StorageDistributionRequest{
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
			},

			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 256,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 2,
						DriveCount:       1,
						IOPS:             1100,
					},
				},
			},
			expectedErr: nil,
		},
		// Test2: choose the right size of the disk by updating the instances per zone
		//        in case of a conflict with two configurations providing the same IOPS
		//        and min capacity choose based of priority
		{
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 256,
						DriveType:        "Standard_LRS",
						InstancesPerZone: 2,
						DriveCount:       1,
						IOPS:             500,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test3: user wants 1TiB on all the nodes
			request: &cloudops.StorageDistributionRequest{
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
			},

			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 1024,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 3,
						DriveCount:       1,
						IOPS:             5000,
					},
				},
			},
			expectedErr: nil,
		},

		{
			// Test4: choose the configuration which is closest to the requested IOPS
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 8192,
						DriveType:        "StandardSSD_LRS",
						InstancesPerZone: 1,
						DriveCount:       1,
						IOPS:             2000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test5: choose upper bound IOPS when you cannot uniformly distribute storage
			// across nodes for the provided IOPS
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 8192,
						DriveType:        "StandardSSD_LRS",
						InstancesPerZone: 1,
						DriveCount:       1,
						IOPS:             2000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test6: reduce the number of instances per zone if the IOPS and min capacity are not met
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 2048,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 1,
						DriveCount:       1,
						IOPS:             7500,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test7: if storage cannot be distributed equally across zones return an error
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 2048,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 1,
						DriveCount:       1,
						IOPS:             7500,
					},
				},
			},

			expectedErr: nil,
		},
		{
			// Test8: Multiple user storage specs in a single request
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 256,
						DriveType:        "Standard_LRS",
						InstancesPerZone: 2,
						DriveCount:       1,
						IOPS:             500,
					},
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 1024,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 3,
						DriveCount:       1,
						IOPS:             5000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test9: Fail the request even if one of the user specs fails
			request: &cloudops.StorageDistributionRequest{
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
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 256,
						DriveType:        "Standard_LRS",
						InstancesPerZone: 2,
						DriveCount:       1,
						IOPS:             500,
					},
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 2048,
						DriveType:        "Premium_LRS",
						InstancesPerZone: 1,
						DriveCount:       1,
						IOPS:             7500,
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, test := range testMatrix {
		response, err := storageManager.GetStorageDistribution(test.request)
		if test.expectedErr == nil {
			require.NoError(t, err, "Unexpected error on GetStorageDistribution")
			require.NotNil(t, response, "got nil response from GetStorageDistribution")
			require.Equal(t, len(test.response.InstanceStorage), len(response.InstanceStorage), "unequal response lengths")
			for i := range test.response.InstanceStorage {
				require.True(t, reflect.DeepEqual(*response.InstanceStorage[i], *test.response.InstanceStorage[i]),
					"Expected Response: %+v . Actual Response %+v",
					test.response.InstanceStorage[i], response.InstanceStorage[i])
			}
		} else {
			require.NotNil(t, err, "GetStorageDistribution should have returned an error")
			require.Equal(t, test.expectedErr, err, "received unexpected type of error")
		}
	}

}

func storageUpdate(t *testing.T) {
	testMatrix := []struct {
		expectedErr error
		request     *cloudops.StorageUpdateRequest
		response    *cloudops.StorageUpdateResponse
	}{
		{
			// ***** TEST: 1
			//        Instance has 3 x 256 GiB
			//        Update from 768GiB to 1536 GiB by resizing disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         1536,
				NewIOPS:             1000,
				ResizeOperationType: api.StoragePoolResizeOperationType_RESIZE_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 256,
						DriveType:        "Premium_LRS",
						IOPS:             1000,
						DriveCount:       3,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.StoragePoolResizeOperationType_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 512,
						DriveType:        "Premium_LRS",
						DriveCount:       3,
						IOPS:             1100,
					},
				},
			},
			expectedErr: nil,
		},
	}

	for _, test := range testMatrix {
		response, err := storageManager.RecommendInstanceStorageUpdate(test.request)
		if test.expectedErr == nil {
			require.Nil(t, err, "RecommendInstanceStorageUpdate returned an error")
			require.NotNil(t, response, "RecommendInstanceStorageUpdate returned empty response")
			require.Equal(t, len(test.response.InstanceStorage), len(response.InstanceStorage), "length of expected and actual response not equal")
			// ensure response contains test.response
			for _, instStorage := range response.InstanceStorage {
				matched := false
				for _, expectedInstStorage := range test.response.InstanceStorage {
					matched = (expectedInstStorage.DriveCapacityGiB == instStorage.DriveCapacityGiB) &&
						(expectedInstStorage.DriveType == instStorage.DriveType) &&
						(expectedInstStorage.DriveCount == instStorage.DriveCount)

					if expectedInstStorage.IOPS > 0 {
						matched = matched && (expectedInstStorage.IOPS >= instStorage.IOPS)
					}

					if matched {
						break
					}

				}

				require.True(t, matched, fmt.Sprintf("response didn't match. expected: %v actual: %v", test.response, response))
			}
		} else {
			require.NotNil(t, err, "RecommendInstanceStorageUpdate should have returned an error")
			require.Equal(t, test.expectedErr, err, "received unexpected type of error")
		}
	}
}
