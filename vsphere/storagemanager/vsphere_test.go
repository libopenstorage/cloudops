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
	testSpecPath = "testspecs/vsphere-storage-decision-matrix.yaml"
)

var (
	storageManager cloudops.StorageManager
)

func TestVsphereStorageManager(t *testing.T) {
	t.Run("setup", setup)
	t.Run("storageDistribution", storageDistribution)
	t.Run("storageUpdate", storageUpdate)
}

func setup(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	decisionMatrix, err := parser.NewStorageDecisionMatrixParser().UnmarshalFromYaml(testSpecPath)
	require.NoError(t, err, "Unexpected error on yaml parser")

	storageManager, err = newVsphereStorageManager(*decisionMatrix)
	require.NoError(t, err, "Unexpected error on creating vsphere storage manager")
}

func storageDistribution(t *testing.T) {
	testMatrix := []struct {
		expectedErr error
		request     *cloudops.StorageDistributionRequest
		response    *cloudops.StorageDistributionResponse
	}{
		{
			// Test1: Distribute 9TiB across 3 zones with each zone having 3 instances
			request: &cloudops.StorageDistributionRequest{
				UserStorageSpec: []*cloudops.StorageSpec{
					&cloudops.StorageSpec{
						MinCapacity: 9216,
						MaxCapacity: 102400,
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
						DriveType:        "thin",
						InstancesPerZone: 3,
						DriveCount:       1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test2: Distribute 18TiB across 3 zones with each zone having 3 instances
			request: &cloudops.StorageDistributionRequest{
				UserStorageSpec: []*cloudops.StorageSpec{
					&cloudops.StorageSpec{
						MinCapacity: 18432,
						MaxCapacity: 102400,
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
						DriveType:        "thin",
						InstancesPerZone: 3,
						DriveCount:       1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// Test3: Distribute 18TiB across 3 zones with each zone having 3 instances
			request: &cloudops.StorageDistributionRequest{
				UserStorageSpec: []*cloudops.StorageSpec{
					&cloudops.StorageSpec{
						MinCapacity: 18432,
						MaxCapacity: 102400,
						DriveType:   "eagerzeroedthick",
					},
				},
				InstancesPerZone: 3,
				ZoneCount:        2,
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 3072,
						DriveType:        "eagerzeroedthick",
						InstancesPerZone: 3,
						DriveCount:       1,
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
			require.True(t, reflect.DeepEqual(*response, *test.response),
				"Expected Response: %+v . Actual Response %+v",
				test.response.InstanceStorage[0], response.InstanceStorage[0])
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
			//        Instance has 3 x 100 GiB
			//        Update from 300GiB to 600 GiB by resizing disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         600,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 2
			//        Instance has 2 x 150 GiB
			//        Update from 300GiB to 400 GiB by resizing disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         400,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 150,
						DriveType:        "thin",
						DriveCount:       2,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "thin",
						DriveCount:       2,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 3
			//        Instance has 3 x 100 GiB
			//        Update from 300GiB to 600 GiB by adding disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         600,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       1,
					},
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       1,
					},
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       1,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 4
			//		  Instances has 3 x 100 GiB
			//        Update from 300GiB to 600 GiB by adding disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         600,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 100,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 5
			//		  Instances has 3 x 200 GiB
			//        Update from 600 GiB to 700 GiB by adding disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         700,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentInstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "thin",
						DriveCount:       3,
					},
				},
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "thin",
						DriveCount:       1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 6
			//		  Instances has no existing drives
			//        Update from 0 GiB to 700 GiB by adding disks
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         700,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 700,
						DriveType:        "thin",
						DriveCount:       1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 7
			//		  Instances has no existing drives
			//        Update from 0 GiB to 8193 GiB by adding disks. 8193 is higher
			//        than the maximum drive in the matrix
			request: &cloudops.StorageUpdateRequest{
				NewCapacity:         8193,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
			},
			response: &cloudops.StorageUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 4097,
						DriveType:        "thin",
						DriveCount:       2,
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
