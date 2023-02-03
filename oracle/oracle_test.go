package oracle

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

const (
	// Minimum size supported by oracle cloud is 50 GB
	newDiskSizeInGB = 50
	newDiskPrefix   = "openstorage-test"
	envKmsKeyID     = "KMS_KEY_ID"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func TestAll(t *testing.T) {
	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})

	d, err := NewClient()
	if err != nil {
		fmt.Printf("err : %+v", err)
		t.Skipf("skipping Oracle tests as environment is not set...\n")
	}

	compartmentID, _ := cloudops.GetEnvValueStrict(fmt.Sprintf("%s", envCompartmentID))
	availabilityDomain, _ := cloudops.GetEnvValueStrict(fmt.Sprintf("%s", envAvailabilityDomain))
	kmsKeyID, _ := cloudops.GetEnvValueStrict(fmt.Sprintf("%s", envKmsKeyID))
	oracleVol := &core.Volume{
		SizeInGBs:          common.Int64(newDiskSizeInGB),
		CompartmentId:      common.String(compartmentID),
		DisplayName:        &diskName,
		VpusPerGB:          common.Int64(10),
		KmsKeyId:           common.String(kmsKeyID),
		AvailabilityDomain: common.String(availabilityDomain),
	}
	drivers[d.Name()] = d
	diskTemplates[d.Name()] = map[string]interface{}{
		diskName: oracleVol,
	}
	test.RunTest(drivers, diskTemplates, sizeCheck, t)
}

func sizeCheck(template interface{}, targetSize uint64) bool {
	// TODO: implement it right way
	return true
}
