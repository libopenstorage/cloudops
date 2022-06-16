package storagemanager

import (
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/csi/storagemanager"
)

const (
	// DriveType3IOPSTier is a constant for 3 IOPS tier drive types
	DriveType3IOPSTier = "3iops-tier"
	// DriveType5IOPSTier is a constant for 5 IOPS tier drive types
	DriveType5IOPSTier = "5iops-tier"
	// DriveType10IOPSTier is a constant for 10 IOPS tier drive types
	DriveType10IOPSTier = "10iops-tier"
	// DriveTypeGeneralPurpose is a constant for general-purpose drive types
	DriveTypeGeneralPurpose = "general-purpose"
	// DriveType3IOPSTierMultiplier is the IOPS multiplier for each GiB for 3 IOPS tier drive type
	DriveType3IOPSTierMultiplier = 3
	// DriveType5IOPSTierMultiplier is the IOPS multiplier for each GiB for 5 IOPS tier drive type
	DriveType5IOPSTierMultiplier = 5
	// DriveType10IOPSTierMultiplier is the IOPS multiplier for each GiB for 10 IOPS tier drive type
	DriveType10IOPSTierMultiplier = 10
	// DriveTypeGeneralPurposeMultiplier is the IOPS multiplier for each GiB for general-purpose drive type
	DriveTypeGeneralPurposeMultiplier = 3
)

type ibmStorageManager struct {
	cloudops.StorageManager
	decisionMatrix *cloudops.StorageDecisionMatrix
}

// newIBMStorageManager returns an IBM implementation for Storage Management
func newIBMStorageManager(
	decisionMatrix cloudops.StorageDecisionMatrix,
) (cloudops.StorageManager, error) {
	return storagemanager.NewCSIStorageManager(decisionMatrix)
}

func init() {
	cloudops.RegisterStorageManager(cloudops.IBM, newIBMStorageManager)
}
