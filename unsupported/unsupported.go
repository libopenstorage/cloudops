package unsupported

import (
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
