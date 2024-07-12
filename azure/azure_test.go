package azure

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-08-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
)

const (
	newDiskSizeInGB = 10
	newDiskPrefix   = "openstorage-test"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func initAzure(t *testing.T) (cloudops.Ops, map[string]interface{}) {
	driver, err := NewEnvClient()
	if err != nil {
		t.Skipf("skipping Azure tests as environment is not set...\n")
	}

	region, present := os.LookupEnv("AZURE_INSTANCE_REGION")
	if !present {
		t.Skipf("skipping Azure tests as AZURE_INSTANCE_REGION is not set...\n")
	}

	size := int32(newDiskSizeInGB)
	name := diskName

	template := &compute.Disk{
		Name:     &name,
		Location: &region,
		DiskProperties: &compute.DiskProperties{
			DiskSizeGB:        &size,
			DiskIOPSReadWrite: to.Int64Ptr(1350),
			DiskMBpsReadWrite: to.Int64Ptr(550),
		},
		Sku: &compute.DiskSku{
			Name: compute.PremiumLRS,
		},
	}

	return driver, map[string]interface{}{
		diskName: template,
	}
}

func TestAll(t *testing.T) {
	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})
	d, disks := initAzure(t)
	drivers[d.Name()] = d
	diskTemplates[d.Name()] = disks
	test.RunTest(drivers, diskTemplates, sizeCheck, t)
}

func sizeCheck(template interface{}, targetSize uint64) bool {
	disk, ok := template.(*compute.Disk)
	if !ok {
		return false
	}
	if disk.DiskProperties == nil || disk.DiskProperties.DiskSizeGB == nil {
		return false
	}
	return targetSize == uint64(*disk.DiskProperties.DiskSizeGB)
}

func TestUpdateUltraIopsThroughput(t *testing.T) {
	testCases := []struct {
		size         int32
		reqIops      *int64
		reqTP        *int64
		expectedIops int64
		expectedTP   int64
	}{

		{
			//TEST1 : where requested IOPS and TP is nil
			size: 100, reqIops: nil, reqTP: nil, expectedIops: 100, expectedTP: 1,
		},
		{
			//TEST2 : Check that minimum IOPS is atleast 100
			size: 32, reqIops: to.Int64Ptr(50), reqTP: to.Int64Ptr(5), expectedIops: 100, expectedTP: 5,
		},
		{
			//TEST3 : If TP is not in range - move to minimum
			size: 100, reqIops: to.Int64Ptr(1000), reqTP: to.Int64Ptr(4000), expectedIops: 1000, expectedTP: 4,
		},
		{
			//TEST4 : If IOPS is not in range - move to minimum
			size: 200, reqIops: to.Int64Ptr(150), reqTP: to.Int64Ptr(8), expectedIops: 200, expectedTP: 8,
		},
		{
			//TEST5 : If  IOPS and TP are in range, use same
			size: 10000, reqIops: to.Int64Ptr(10000), reqTP: to.Int64Ptr(40), expectedIops: 10000, expectedTP: 40,
		},
		{
			//TEST6 : If IOPS is greater than 400000, move to minimum
			size: 20000, reqIops: to.Int64Ptr(420000), reqTP: to.Int64Ptr(16000), expectedIops: 20000, expectedTP: 79,
		},
	}
	for _, tc := range testCases {
		size := tc.size
		reqIops := tc.reqIops
		reqTP := tc.reqTP
		updateUltraIopsThroughput(size, reqIops, reqTP)

		if reqIops != nil && *reqIops != tc.expectedIops {
			t.Errorf("For size %d, expected reqIops to be %d, but got %d", size, tc.expectedIops, *reqIops)
		}
		if reqTP != nil && *reqTP != tc.expectedTP {
			t.Errorf("For size %d, expected reqIops to be %d, but got %d", size, tc.expectedTP, *reqTP)
		}
	}
}
func TestUpdatePremiumv2IopsThroughput(t *testing.T) {
	testCases := []struct {
		size         int32
		reqIops      *int64
		reqTP        *int64
		expectedIops int64
		expectedTP   int64
	}{

		{
			//TEST1 : where requested IOPS and TP is nil
			size: 100, reqIops: nil, reqTP: nil, expectedIops: 3000, expectedTP: 125,
		},
		{
			//TEST2 : Check that minimum IOPS is atleast 3000
			size: 32, reqIops: to.Int64Ptr(2500), reqTP: to.Int64Ptr(125), expectedIops: 3000, expectedTP: 125,
		},
		{
			//TEST3 : If TP is not in range - move to minimum
			size: 100, reqIops: to.Int64Ptr(1000), reqTP: to.Int64Ptr(12), expectedIops: 3000, expectedTP: 125,
		},
		{
			//TEST4 : If  IOPS and TP are in range, use same
			size: 10000, reqIops: to.Int64Ptr(10000), reqTP: to.Int64Ptr(40), expectedIops: 10000, expectedTP: 125,
		},
		{
			//TEST5 : If IOPS is greater than 80000, move to minimum
			size: 20000, reqIops: to.Int64Ptr(85000), reqTP: to.Int64Ptr(125), expectedIops: 3000, expectedTP: 125,
		},
	}
	for _, tc := range testCases {
		size := tc.size
		reqIops := tc.reqIops
		reqTP := tc.reqTP
		updatePremiumv2IopsThroughput(size, reqIops, reqTP)

		if reqIops != nil && *reqIops != tc.expectedIops {
			t.Errorf("For size %d, expected reqIops to be %d, but got %d", size, tc.expectedIops, *reqIops)
		}
		if reqTP != nil && *reqTP != tc.expectedTP {
			t.Errorf("For size %d, expected reqTP to be %d, but got %d", size, tc.expectedTP, *reqTP)
		}
	}
}
func TestCalculateMinThroughput(t *testing.T) {
	testCases := []struct {
		iops     int64
		expected int64
	}{
		{iops: 0, expected: 1},
		{iops: 100, expected: 1},
		{iops: 1000, expected: 4},
		{iops: 2000, expected: 8},
		{iops: 4000, expected: 16},
		{iops: 8000, expected: 32},
		{iops: 10000, expected: 40},
		{iops: 20000, expected: 79},
		{iops: 40000, expected: 157},
	}

	for _, tc := range testCases {
		iops := tc.iops
		expected := tc.expected
		minThroughput := calculateMinThroughput(iops)

		if minThroughput != expected {
			t.Errorf("For IOPS %d, expected minimum throughput to be %d, but got %d", iops, expected, minThroughput)
		}
	}
}
