package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	// "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	// "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	// "github.com/Azure/go-autorest/autorest"
)

type scaleSetVMsClient struct {
	scaleSetName      string
	resourceGroupName string
	client            *armcompute.VirtualMachineScaleSetVMsClient
}

func newScaleSetVMsClient(
	config Config,
	baseURI string,
	credential azcore.TokenCredential,
) vmsClient {
	// vmsClient := compute.NewVirtualMachineScaleSetVMsClientWithBaseURI(baseURI, config.SubscriptionID)
	vmsClient, _ := armcompute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID, credential, nil)

	// vmsClient.PollingDelay = clientPollingDelay
	// vmsClient.AddToUserAgent(config.UserAgent)
	return &scaleSetVMsClient{
		scaleSetName:      config.ScaleSetName,
		resourceGroupName: config.ResourceGroupName,
		client:            vmsClient,
	}
}

func (s *scaleSetVMsClient) name(instanceID string) string {
	return s.scaleSetName + "_" + instanceID
}

func (s *scaleSetVMsClient) describe(
	instanceID string,
) (interface{}, error) {
	return s.describeInstance(instanceID)
}

func (s *scaleSetVMsClient) getDataDisks(
	instanceID string,
) ([]*armcompute.DataDisk, error) {
	vm, err := s.describeInstance(instanceID)
	if err != nil {
		return nil, err
	}

	return retrieveDataDisks(*vm), nil
}

func (s *scaleSetVMsClient) updateDataDisks(
	instanceID string,
	dataDisks []*armcompute.DataDisk,
) error {
	vm, err := s.describeInstance(instanceID)
	if err != nil {
		return err
	}

	vm.Properties = &armcompute.VirtualMachineScaleSetVMProperties{
		StorageProfile: &armcompute.StorageProfile{
			DataDisks: dataDisks,
		},
	}

	ctx := context.Background()

	poller, err := s.client.BeginUpdate(
		ctx,
		s.resourceGroupName,
		s.scaleSetName,
		instanceID,
		*vm,
		nil,
	)

	//future, err := s.client.Update(
	//	ctx,
	//	s.resourceGroupName,
	//	s.scaleSetName,
	//	instanceID,
	//	vm,
	//)
	if err != nil {
		return err
	}

	// err = future.WaitForCompletionRef(ctx, s.client.Client)
	_, err = poller.PollUntilDone(context.Background(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *scaleSetVMsClient) describeInstance(
	instanceID string,
) (*armcompute.VirtualMachineScaleSetVM, error) {
	viewType := armcompute.InstanceViewTypesInstanceView
	resp, err := s.client.Get(
		context.Background(),
		s.resourceGroupName,
		s.scaleSetName,
		instanceID,
		&armcompute.VirtualMachineScaleSetVMsClientGetOptions{
			Expand: &viewType,
		},
	)
	if err != nil {
		return nil, err
	}
	return &resp.VirtualMachineScaleSetVM, nil
}

func retrieveDataDisks(vm armcompute.VirtualMachineScaleSetVM) []*armcompute.DataDisk {
	if vm.Properties == nil ||
		vm.Properties.StorageProfile == nil ||
		vm.Properties.StorageProfile.DataDisks == nil {
		return []*armcompute.DataDisk{}
	}

	return vm.Properties.StorageProfile.DataDisks
}
