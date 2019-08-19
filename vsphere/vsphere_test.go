package vsphere

import (
	"fmt"
	"testing"
	"time"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/vim25/types"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere/vclib"
)

const (
	newDiskSizeInKB    = 2097152 // 2GB
	newDiskPrefix      = "openstorage-test"
	newDiskDescription = "Disk created by Openstorage tests"
)

var (
	diskName             = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())
	drivers              = make(map[string]cloudops.Ops)
	diskTemplates        = make(map[string]map[string]interface{})
	vmCreateOptsByDriver = make(map[string][]interface{})
	cfg                  *VSphereConfig
)

func initVsphere(t *testing.T) (cloudops.Ops, map[string]interface{}, []interface{}) {
	var err error
	cfg, err = ReadVSphereConfigFromEnv()
	require.NoError(t, err, "failed to get vsphere config from env")

	cfg.VMUUID, err = cloudops.GetEnvValueStrict("VSPHERE_VM_UUID")
	require.NoError(t, err, "failed to get vsphere config from env variable VSPHERE_VM_UUID")

	datastoreForTest, err := cloudops.GetEnvValueStrict("VSPHERE_TEST_DATASTORE")
	require.NoError(t, err, "failed to get datastore from env variable VSPHERE_TEST_DATASTORE")

	hostForTest, err := cloudops.GetEnvValueStrict("VSPHERE_TEST_HOST")
	require.NoError(t, err, "failed to get host  from env variable VSPHERE_TEST_HOST")

	driver, err := NewClient(cfg)
	require.NoError(t, err, "failed to instantiate storage ops driver")

	tags := map[string]string{
		"foo": "bar",
	}

	timestamp := time.Now().Format("20060102150405")

	diskOptions := &vclib.VolumeOptions{
		Name:       diskName,
		Tags:       tags,
		CapacityKB: newDiskSizeInKB,
		Datastore:  datastoreForTest,
	}

	vmsToCreate := []interface{}{
		&VirtualMachineCreateOpts{
			Spec: &types.VirtualMachineConfigSpec{
				Name:       fmt.Sprintf("cloudops-test-vm-1-%s", timestamp),
				NumCPUs:    1,
				MemoryMB:   1024,
				Annotation: "created-by-cloudops-test",
				GuestId:    "freebsd64Guest",
				Version:    "vmx-14",
			},
			Datastore: datastoreForTest,
			PowerOn:   false,
			Host:      hostForTest,
		},
	}

	return driver, map[string]interface{}{
		diskName: diskOptions,
	}, vmsToCreate
}

func TestAll(t *testing.T) {
	if IsDevMode() {
		t.Run("setup", setup)
		test.RunTest(drivers, diskTemplates, vmCreateOptsByDriver, t)
	} else {
		fmt.Printf("skipping vSphere tests as environment is not set...\n")
		t.Skip("skipping vSphere tests as environment is not set...")
	}
}

func TestDeviceMappings(t *testing.T) {
	if IsDevMode() {
		t.Run("setup", setup)
		for _, driver := range drivers {
			test.DeviceMappings(t, driver, cfg.VMUUID)
		}
	} else {
		fmt.Printf("skipping vSphere tests as environment is not set...\n")
		t.Skip("skipping vSphere tests as environment is not set...")
	}
}

func setup(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	d, disks, vmCreateOpts := initVsphere(t)
	drivers[d.Name()] = d
	diskTemplates[d.Name()] = disks
	vmCreateOptsByDriver[d.Name()] = vmCreateOpts
}
