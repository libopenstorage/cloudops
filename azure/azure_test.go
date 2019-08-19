package azure_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
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

	template := &compute.Disk{
		Name:     &name,
		Location: &region,
		DiskProperties: &compute.DiskProperties{
			DiskSizeGB:        &size,
			DiskIOPSReadWrite: to.Int64Ptr(1350),
			DiskMBpsReadWrite: to.Int32Ptr(550),
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
	test.RunTest(drivers, diskTemplates, nil, t)
}
