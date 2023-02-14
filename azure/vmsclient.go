package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	// "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	// "github.com/Azure/go-autorest/autorest"
)

// vmsClient is an interface for azure vm client operations
type vmsClient interface {
	// name returns the name of the instance
	name(instanceID string) string
	// describe returns the VM instance object
	describe(instanceID string) (interface{}, error)
	// getDataDisks returns a list of data disks attached to the given VM
	getDataDisks(instanceID string) ([]*armcompute.DataDisk, error)
	// updateDataDisks update the data disks for the given VM
	updateDataDisks(instanceID string, dataDisks []*armcompute.DataDisk) error
}

func newVMsClient(
	config Config,
	baseURI string,
	credential azcore.TokenCredential,
	// authorizer autorest.Authorizer,
) vmsClient {
	if config.ScaleSetName == "" {
		return newBaseVMsClient(config, baseURI, credential)
	}
	return newScaleSetVMsClient(config, baseURI, credential)
}
