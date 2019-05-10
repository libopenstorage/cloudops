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

func (u *unsupportedCompute) InspectSelf() (*cloudops.InstanceInfo, error) {
	return nil, cloudops.ErrUnsupported
}

// InspectInstanceGroup returns instanceGroupInfo matching labels.
func (u *unsupportedCompute) InspectSelfInstanceGroup() (*cloudops.InstanceGroupInfo, error) {
	return nil, cloudops.ErrUnsupported
}
