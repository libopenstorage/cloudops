package azure

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	// "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	// "github.com/Azure/go-autorest/autorest"
)

type baseVMsClient struct {
	resourceGroupName string
	client            *armcompute.VirtualMachinesClient
}

func newBaseVMsClient(
	config Config,
	baseURI string,
	credential azcore.TokenCredential,
) vmsClient {
	//options := arm.ClientOptions {
	//	ClientOptions: azcore.ClientOptions {
	//		Cloud: cloud.AzureChina,
	//	},
	//}
	vmsClient, err := armcompute.NewVirtualMachinesClient(config.SubscriptionID, credential, nil)
	if err != nil {

	}
	// vmsClient, err := armcompute.NewVirtualMachinesClient(config.SubscriptionID, credential, &options)
	// vmsClient.Authorizer = authorizer
	// vmsClient.PollingDelay = clientPollingDelay
	// vmsClient.AddToUserAgent(config.UserAgent)

	return &baseVMsClient{
		resourceGroupName: config.ResourceGroupName,
		client:            vmsClient,
	}
}

func (b *baseVMsClient) name(instanceName string) string {
	return instanceName
}

func (b *baseVMsClient) describe(
	instanceName string,
) (interface{}, error) {
	return b.describeInstance(instanceName)
}

func (b *baseVMsClient) getDataDisks(instanceName string, ) ([]*armcompute.DataDisk, error) {
	vm, err := b.describeInstance(instanceName)
	if err != nil {
		return nil, err
	}

	if vm.Properties.StorageProfile == nil ||
		vm.Properties.StorageProfile.DataDisks == nil {
		return nil, fmt.Errorf("vm storage profile is invalid")
	}

	return vm.Properties.StorageProfile.DataDisks, nil
}

func (b *baseVMsClient) updateDataDisks(
	instanceName string,
	dataDisks []*armcompute.DataDisk,
) error {
	updatedVM := armcompute.VirtualMachineUpdate{
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile: &armcompute.StorageProfile{
				DataDisks: dataDisks,
			},
		},
	}

	poller, err := b.client.BeginUpdate(
		context.TODO(),
		b.resourceGroupName,
		instanceName,
		updatedVM,
		nil,
		)

	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(context.Background(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (b *baseVMsClient) describeInstance(
	instanceName string,
) (*armcompute.VirtualMachine, error) {
	viewType := armcompute.InstanceViewTypesInstanceView
	resp, err := b.client.Get(
		context.Background(),
		b.resourceGroupName,
		instanceName,
		&armcompute.VirtualMachinesClientGetOptions{
			Expand: &viewType,
		},
	)
	if err != nil {
		return nil, err
	}
	return &resp.VirtualMachine, nil
}
