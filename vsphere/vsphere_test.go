package vsphere

import (
	"fmt"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere/vclib"
)

const (
	newDiskSizeInKB    = 2097152 // 2GB
	newDiskPrefix      = "openstorage-test"
	newDiskDescription = "Disk created by Openstorage tests"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func initVsphere(t *testing.T) (cloudops.Ops, map[string]interface{}) {
	cfg, err := ReadVSphereConfigFromEnv()
	require.NoError(t, err, "failed to get vsphere config from env")

	cfg.VMUUID, err = cloudops.GetEnvValueStrict("VSPHERE_VM_UUID")
	require.NoError(t, err, "failed to get vsphere config from env variable VSPHERE_VM_UUID")

	datastoreForTest, err := cloudops.GetEnvValueStrict("VSPHERE_TEST_DATASTORE")
	require.NoError(t, err, "failed to get datastore from env variable VSPHERE_TEST_DATASTORE")

	driver, err := NewClient(cfg)
	require.NoError(t, err, "failed to instantiate storage ops driver")

	tags := map[string]string{
		"foo": "bar",
	}
	diskOptions := &vclib.VolumeOptions{
		Name:       diskName,
		Tags:       tags,
		CapacityKB: newDiskSizeInKB,
		Datastore:  datastoreForTest,
	}

	return driver, map[string]interface{}{
		diskName: diskOptions,
	}
}

func TestAll(t *testing.T) {
	if IsDevMode() {
		drivers := make(map[string]cloudops.Ops)
		diskTemplates := make(map[string]map[string]interface{})

		d, disks := initVsphere(t)
		drivers[d.Name()] = d
		diskTemplates[d.Name()] = disks

		test.RunTest(drivers, diskTemplates, nil, t)
	} else {
		fmt.Printf("skipping vSphere tests as environment is not set...\n")
		t.Skip("skipping vSphere tests as environment is not set...")
	}
}
