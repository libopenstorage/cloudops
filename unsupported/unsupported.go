package unsupported

import (
	"time"

	"github.com/libopenstorage/cloudops"
)

type unsupportedCompute struct {
}

// NewUnsupportedCompute return wrapper for cloudOps where all methods are not supported
func NewUnsupportedCompute() cloudops.Compute {
	return &unsupportedCompute{}
}

func (u *unsupportedCompute) InstanceID() string {
	return "Unsupported"
}

func (u *unsupportedCompute) InspectInstance(instanceID string) (*cloudops.InstanceInfo, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "InspectInstance",
	}
}

func (u *unsupportedCompute) InspectInstanceGroupForInstance(instanceID string) (*cloudops.InstanceGroupInfo, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "InspectInstanceGroupForInstance",
	}
}

func (u *unsupportedCompute) SetInstanceGroupSize(instanceID string, count int64, timeout time.Duration) error {
	return &cloudops.ErrNotSupported{
		Operation: "SetInstanceGroupSize",
	}
}

func (u *unsupportedCompute) GetClusterSize(instanceID string) (int64, error) {
	return 0, &cloudops.ErrNotSupported{
		Operation: "GetClusterSize",
	}
}

func (u *unsupportedCompute) GetClusterStatus(instanceID string) (cloudops.ClusterState, error) {
	return cloudops.StatusUnspecified, &cloudops.ErrNotSupported{
		Operation: "GetClusterStatus",
	}
}
