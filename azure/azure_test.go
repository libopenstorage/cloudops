package azure_test

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/go-autorest/autorest/to"
	"os"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/azure"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
)

const (
	newDiskSizeInGB = 10
	newDiskPrefix   = "openstorage-test"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func initAzure(t *testing.T) (cloudops.Ops, map[string]interface{}) {
	driver, err := azure.NewEnvClient()
	if err != nil {
		t.Skipf("skipping Azure tests as environment is not set...\n")
	}

	region, present := os.LookupEnv("AZURE_INSTANCE_REGION")
	if !present {
		t.Skipf("skipping Azure tests as AZURE_INSTANCE_REGION is not set...\n")
	}

	size := int32(newDiskSizeInGB)
	name := diskName

	premiumLRS := armcompute.DiskStorageAccountTypesPremiumLRS
	template := &armcompute.Disk{
		Name:     &name,
		Location: &region,
		Properties: &armcompute.DiskProperties{
			DiskSizeGB:        &size,
			DiskIOPSReadWrite: to.Int64Ptr(1350),
			DiskMBpsReadWrite: to.Int64Ptr(550),
		},
		SKU: &armcompute.DiskSKU{
			Name: &premiumLRS,
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
	disk, ok := template.(*armcompute.Disk)
	if !ok {
		return false
	}
	if disk.Properties == nil || disk.Properties.DiskSizeGB == nil {
		return false
	}
	return targetSize == uint64(*disk.Properties.DiskSizeGB)
}
