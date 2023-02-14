package azure

import (
	"github.com/Azure/go-autorest/autorest/to"
	"testing"

	// "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	// "github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/stretchr/testify/require"
)

func TestRetrieveDataDisks(t *testing.T) {
	var nilDiskSlice []*armcompute.DataDisk
	testDisks := []*armcompute.DataDisk{
		{
			Name: to.StringPtr("disk1"),
		},
		{
			Name: to.StringPtr("disk2"),
		},
	}

	testCases := []struct {
		name        string
		input       armcompute.VirtualMachineScaleSetVM
		expectedRes []*armcompute.DataDisk
	}{
		{
			name:  "nil vm properties",
			input: armcompute.VirtualMachineScaleSetVM{},
			expectedRes: []*armcompute.DataDisk{},
		},
		{
			name: "nil storage profile",
			input: armcompute.VirtualMachineScaleSetVM{
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{},
			},
			expectedRes: []*armcompute.DataDisk{},
		},
		{
			name: "nil data disks reference",
			input: armcompute.VirtualMachineScaleSetVM{
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{
					StorageProfile: &armcompute.StorageProfile{},
				},
			},
			expectedRes: []*armcompute.DataDisk{},
		},
		{
			name: "nil data disks slice",
			input: armcompute.VirtualMachineScaleSetVM{
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{
					StorageProfile: &armcompute.StorageProfile{
						DataDisks: nilDiskSlice,
					},
				},
			},
			expectedRes: []*armcompute.DataDisk{},
		},
		{
			name: "empty data disks slice",
			input: armcompute.VirtualMachineScaleSetVM{
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{
					StorageProfile: &armcompute.StorageProfile{
						DataDisks: []*armcompute.DataDisk{},
					},
				},
			},
			expectedRes: []*armcompute.DataDisk{},
		},
		{
			name: "test data disks",
			input: armcompute.VirtualMachineScaleSetVM{
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{
					StorageProfile: &armcompute.StorageProfile{
						DataDisks: testDisks,
					},
				},
			},
			expectedRes: testDisks,
		},
	}

	for _, tc := range testCases {
		res := retrieveDataDisks(tc.input)
		require.Equalf(t, tc.expectedRes, res, "TC: %s", tc.name)
	}
}
