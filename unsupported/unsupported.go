package unsupported

import (
	"github.com/libopenstorage/cloudops"
)

type unsupportedCompte struct {
}

// NewUnsupportedCompute return wrapper for cloudOps
func NewUnsupportedCompute() cloudops.Compute {
	return &unsupportedCompte{}
}

func (u *unsupportedCompte) InstanceID() string {
	return "Unsupported"
}

// Inspect instance identified by ID.
func (u *unsupportedCompte) InspectIntance(ID string) (*cloudops.InstanceInfo, error) {
	return nil, cloudops.ErrUnsupported
}

// InspectInstanceGroup returns instanceGroupInfo matching labels.
func (u *unsupportedCompte) InspectInstanceGroup(
	labels map[string]string,
) (*cloudops.InstanceInfo, error) {
	return nil, cloudops.ErrUnsupported
}
