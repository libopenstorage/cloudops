package parser

import (
	"reflect"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/stretchr/testify/require"
)

const (
	testYamlFilePath     = "/tmp/cloudops-test.yaml"
	existingYamlFilePath = "testspecs/test.yaml"
)

func TestStorageDecisionMatrixParser(t *testing.T) {
	inputMatrix := cloudops.StorageDecisionMatrix{
		Rows: []cloudops.StorageDecisionMatrixRow{
			cloudops.StorageDecisionMatrixRow{
				IOPS:         uint32(1000),
				MinSize:      uint64(100),
				MaxSize:      uint64(200),
				InstanceType: "foo",
			},
			cloudops.StorageDecisionMatrixRow{
				IOPS:         uint32(2000),
				MinSize:      uint64(200),
				MaxSize:      uint64(400),
				InstanceType: "bar",
			},
		},
	}
	p := NewStorageDecisionMatrixParser()
	err := p.MarshalToYaml(&inputMatrix, testYamlFilePath)
	require.NoError(t, err, "Unexpected error on MarshalToYaml")

	actualMatrix, err := p.UnmarshalFromYaml(testYamlFilePath)
	require.NoError(t, err, "Unexpected error on UnmarshalFromYaml")
	require.True(t, reflect.DeepEqual(inputMatrix, *actualMatrix), "Unequal matrices %v %v", inputMatrix, *actualMatrix)
}

func TestStorageDecisionMatrixParserWithExistingYaml(t *testing.T) {

	expectedMatrix := cloudops.StorageDecisionMatrix{
		Rows: []cloudops.StorageDecisionMatrixRow{
			cloudops.StorageDecisionMatrixRow{
				IOPS:              uint32(1100),
				MinSize:           uint64(256),
				MaxSize:           uint64(8192),
				InstanceType:      "*",
				Region:            "*",
				InstanceMaxDrives: uint32(8),
				InstanceMinDrives: uint32(1),
				Priority:          0,
				ThinProvisioning:  false,
				DriveType:         "Premium_LRS",
			},
			cloudops.StorageDecisionMatrixRow{
				IOPS:              uint32(500),
				MinSize:           uint64(256),
				MaxSize:           uint64(4096),
				InstanceType:      "*",
				Region:            "*",
				InstanceMaxDrives: uint32(8),
				InstanceMinDrives: uint32(1),
				Priority:          1,
				ThinProvisioning:  false,
				DriveType:         "StandardSSD_LRS",
			},
			cloudops.StorageDecisionMatrixRow{
				IOPS:              uint32(1300),
				MinSize:           uint64(8192),
				MaxSize:           uint64(8192),
				InstanceType:      "*",
				Region:            "*",
				InstanceMaxDrives: uint32(8),
				InstanceMinDrives: uint32(1),
				Priority:          2,
				ThinProvisioning:  false,
				DriveType:         "Standard_LRS",
			},
		},
	}
	p := NewStorageDecisionMatrixParser()
	actualMatrix, err := p.UnmarshalFromYaml(existingYamlFilePath)
	require.NoError(t, err, "Unexpected error on UnmarshalFromYaml")
	require.True(t, reflect.DeepEqual(expectedMatrix, *actualMatrix), "Unequal matrices %v %v", expectedMatrix, *actualMatrix)

}
