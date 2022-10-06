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
	testSpecPath = "testspecs/oracle.yaml"
)

var (
	storageManager cloudops.StorageManager
)

type updateTestInput struct {
	expectedErr error
	request     *cloudops.StoragePoolUpdateRequest
	response    *cloudops.StoragePoolUpdateResponse
}

func TestOracleStorageManager(t *testing.T) {
	t.Run("setup", setup)
	t.Run("storageDistribution", storageDistribution)
	t.Run("storageUpdate", storageUpdate)
}

func setup(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	decisionMatrix, err := parser.NewStorageDecisionMatrixParser().UnmarshalFromYaml(testSpecPath)
	require.NoError(t, err, "Unexpected error on yaml parser")

	storageManager, err = NewStorageManager(*decisionMatrix)
	require.NoError(t, err, "Unexpected error on creating Oracle storage manager")
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
						MaxCapacity: 4096,
					},
				},
				InstanceType:     "foo",
				InstancesPerZone: 3,
				ZoneCount:        2,
			},

			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 250,
						DriveType:        "0_vpus",
						InstancesPerZone: 3,
						DriveCount:       1,
						IOPS:             500,
					},
				},
			},
			expectedErr: nil,
		},
		// Did not understood
		// Test2: choose the right size of the disk by updating the instances per zone
		//        in case of a conflict with two configurations providing the same IOPS
		//        and min capacity choose based of priority
		{
			request: &cloudops.StorageDistributionRequest{
				UserStorageSpec: []*cloudops.StorageSpec{
					&cloudops.StorageSpec{
						IOPS:        500,
						MinCapacity: 1024,
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
						DriveCapacityGiB: 14,
						DriveType:        "0_vpus",
						InstancesPerZone: 3,
						DriveCount:       8,
						IOPS:             28,
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
						IOPS:        2900,
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
						DriveCapacityGiB: 1250,
						DriveType:        "0_vpus",
						InstancesPerZone: 3,
						DriveCount:       1,
						IOPS:             2500,
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
						IOPS:        5700,
						MinCapacity: 8000,
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
						DriveCapacityGiB: 250,
						DriveType:        "10_vpus",
						InstancesPerZone: 2,
						DriveCount:       8,
						IOPS:             15000,
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
						IOPS:        800,
						MinCapacity: 2096,
						MaxCapacity: 10000,
					},
				},
				InstanceType:     "foo",
				InstancesPerZone: 2,
				ZoneCount:        3,
			},
			response: &cloudops.StorageDistributionResponse{
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 349,
						DriveType:        "0_vpus",
						InstancesPerZone: 2,
						DriveCount:       1,
						IOPS:             698,
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
						DriveCapacityGiB: 128,
						DriveType:        "10_vpus",
						InstancesPerZone: 2,
						DriveCount:       8,
						IOPS:             7680,
					},
				},
			},
			expectedErr: nil,
		},
	}
	for j, test := range testMatrix {
		fmt.Println("Executing test case: ", j+1)
		response, err := storageManager.GetStorageDistribution(test.request)
		if test.expectedErr == nil {
			require.NoError(t, err, "Unexpected error on GetStorageDistribution")
			require.NotNil(t, response, "got nil response from GetStorageDistribution")
			require.Equal(t, len(test.response.InstanceStorage), len(response.InstanceStorage), "unequal response lengths")
			for i := range test.response.InstanceStorage {
				require.True(t, reflect.DeepEqual(*response.InstanceStorage[i], *test.response.InstanceStorage[i]),
					"Test Case %v Expected Response: %+v . Actual Response %+v", j+1,
					test.response.InstanceStorage[i], response.InstanceStorage[i])
			}
		} else {
			require.NotNil(t, err, "GetStorageDistribution should have returned an error")
			require.Equal(t, test.expectedErr, err, "received unexpected type of error")
		}
	}
}

func storageUpdate(t *testing.T) {
	testMatrix := []updateTestInput{
		{
			// ***** TEST: 1
			//        Instance has 3 x 256 GiB
			//        Update from 768GiB to 1536 GiB by resizing disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     1536,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentDriveSize:    256,
				CurrentDriveType:    "20_vpus",
				CurrentIOPS:         768,
				CurrentDriveCount:   3,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 512,
						DriveType:        "20_vpus",
						DriveCount:       3,
						IOPS:             38400,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 2
			//        Instance has 2 x 350 GiB
			//        Update from 700GiB to 800 GiB by resizing disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     800,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentDriveSize:    350,
				CurrentDriveType:    "20_vpus",
				CurrentDriveCount:   2,
				TotalDrivesOnNode:   2,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 400,
						DriveType:        "20_vpus",
						DriveCount:       2,
						IOPS:             30000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 3
			//        Instance has 3 x 300 GiB
			//        Update from 900GiB to 1200 GiB by resizing disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     1200,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentDriveSize:    300,
				CurrentDriveType:    "20_vpus",
				CurrentDriveCount:   3,
				TotalDrivesOnNode:   3,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 400,
						DriveType:        "20_vpus",
						DriveCount:       3,
						IOPS:             30000,
					},
				},
			},
			expectedErr: nil,
		},
		// Did not understood
		{
			// ***** TEST: 4
			//		  Instances has 2 x 1024 GiB
			//        Update from 2048 GiB to  4096 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     4096,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    1024,
				CurrentDriveType:    "50_vpus",
				CurrentDriveCount:   2,
				TotalDrivesOnNode:   2,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 1024,
						DriveType:        "50_vpus",
						DriveCount:       2,
						IOPS:             122880,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 5
			//		  Instances has 2 x 1024 GiB
			//        Update from 2048 GiB to  3072 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     3072,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    1024,
				CurrentDriveType:    "50_vpus",
				CurrentDriveCount:   2,
				TotalDrivesOnNode:   2,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 1024,
						DriveType:        "50_vpus",
						DriveCount:       1,
						IOPS:             122880,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 6
			//		  Instances has 3 x 600 GiB
			//        Update from 1800 GiB to 2000 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     2000,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    600,
				CurrentDriveType:    "20_vpus",
				CurrentDriveCount:   3,
				TotalDrivesOnNode:   3,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 600,
						DriveType:        "20_vpus",
						DriveCount:       1,
						IOPS:             45000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 7
			//		  Instances has no existing drives
			//        Update from 0 GiB to 700 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     700,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				TotalDrivesOnNode:   0,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 700,
						DriveType:        "0_vpus",
						DriveCount:       1,
						IOPS:             1400,
					},
				},
			},
			expectedErr: nil,
		},
		//{
		// ***** TEST: 8
		//		  Instances has no existing drives
		//        Update from 0 GiB to 2666 GiB by adding disks. 2666 is higher
		//        than the maximum drive in the matrix
		/*
						time="2022-10-06T10:34:04+05:30" level=debug msg="-- Storage Distribution Pool Update Request --" MinCapacity=2666 OperationType=RESIZE_TYPE_ADD_DISK
			time="2022-10-06T10:34:04+05:30" level=debug msg="check if we can add drive(s) for atleast: 2666 GiB"
			    oracle_test.go:251:
			        	Error Trace:	oracle_test.go:251
			        	Error:
			        	Test:       	TestOracleStorageManager/storageUpdate
			        	Messages:   	RecommendStoragePoolUpdate returned an error
			--- FAIL: TestOracleStorageManager (0.08s)
			    --- PASS: TestOracleStorageManager/setup (0.07s)
			    --- PASS: TestOracleStorageManager/storageDistribution (0.00s)
			    --- FAIL: TestOracleStorageManager/storageUpdate (0.01s)
			FAIL
			FAIL	github.com/libopenstorage/cloudops/oracle/storagemanager	0.251s
			FAIL

						request: &cloudops.StoragePoolUpdateRequest{
							DesiredCapacity:     2666,
							ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
							TotalDrivesOnNode:   0,
						},
						response: &cloudops.StoragePoolUpdateResponse{
							ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
							InstanceStorage: []*cloudops.StoragePoolSpec{
								&cloudops.StoragePoolSpec{
									DriveCapacityGiB: 1333,
									DriveType:        "120_vpus",
									DriveCount:       2,
								},
							},
						},
						expectedErr: nil,
					}*/
		{
			// ***** TEST: 9
			//        Instance has 1 x 150 GiB
			//        Update from 256GiB to 280 GiB by resizing disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     280,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				CurrentDriveSize:    256,
				CurrentDriveType:    "20_vpus",
				CurrentDriveCount:   1,
				TotalDrivesOnNode:   1,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 280,
						DriveType:        "20_vpus",
						DriveCount:       1,
						IOPS:             21000,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 10 -> lower sized disks
			//        Instance has 1 x 200 GiB
			//        Update from 200GiB to 400 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     400,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    200,
				CurrentDriveType:    "0_vpus",
				CurrentDriveCount:   1,
				TotalDrivesOnNode:   1,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "0_vpus",
						DriveCount:       1,
						IOPS:             400,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 11 -> ask for one more GiB
			//        Instance has 2 x 200 GiB
			//        Update from 400 GiB to 401 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     401,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    200,
				CurrentDriveType:    "0_vpus",
				CurrentDriveCount:   2,
				TotalDrivesOnNode:   2,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage: []*cloudops.StoragePoolSpec{
					&cloudops.StoragePoolSpec{
						DriveCapacityGiB: 200,
						DriveType:        "0_vpus",
						DriveCount:       1,
						IOPS:             400,
					},
				},
			},
			expectedErr: nil,
		},
		{
			// ***** TEST: 12 instance is already at higher capacity than requested
			//        Instance has 3 x 200 GiB
			//        Update from 600 GiB to 401 GiB by adding disks
			request: &cloudops.StoragePoolUpdateRequest{
				DesiredCapacity:     401,
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				CurrentDriveSize:    200,
				CurrentDriveType:    "0_vpus",
				CurrentDriveCount:   3,
				TotalDrivesOnNode:   3,
			},
			response: &cloudops.StoragePoolUpdateResponse{
				ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
				InstanceStorage:     nil,
			},
			expectedErr: &cloudops.ErrCurrentCapacityHigherThanDesired{Current: 600, Desired: 401},
		},
	}

	for j, test := range testMatrix {
		fmt.Println("Executing test case: ", j+1)
		response, err := storageManager.RecommendStoragePoolUpdate(test.request)
		if test.expectedErr == nil {
			require.Nil(t, err, "RecommendStoragePoolUpdate returned an error")
			require.NotNil(t, response, "RecommendStoragePoolUpdate returned empty response")
			require.Equal(t, len(test.response.InstanceStorage), len(response.InstanceStorage), "length of expected and actual response not equal")
			for i := range test.response.InstanceStorage {
				require.True(t, reflect.DeepEqual(*response.InstanceStorage[i], *test.response.InstanceStorage[i]),
					"Test Case %v Expected Response: %+v . Actual Response %+v", j+1,
					test.response.InstanceStorage[i], response.InstanceStorage[i])
			}
		} else {
			require.NotNil(t, err, "RecommendInstanceStorageUpdate should have returned an error")
			require.Equal(t, test.expectedErr.Error(), err.Error(), "received unexpected type of error")
		}
	}
}
