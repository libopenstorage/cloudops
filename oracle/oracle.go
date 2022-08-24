package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/libopenstorage/cloudops"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/portworx/sched-ops/task"
	"github.com/sirupsen/logrus"
)

const (
	v1MetadataAPIEndpoint = "http://169.254.169.254/opc/v1/instance/"
	v2MetadataAPIEndpoint = "http://169.254.169.254/opc/v2/instance/"
	metadataInstanceIDkey = "id"
	// MetadataRegionKey key name for region in metadata JSON
	// returned by IMDS service
	MetadataRegionKey = "canonicalRegionName"
	// MetadataAvailabilityDomainKey key name for availability domain
	// in metadata JSON returned by IMDS service
	MetadataAvailabilityDomainKey = "availabilityDomain"
	// MetadataCompartmentIDkey key name for compartmentID
	// in metadata JSON returned by IMDS service
	MetadataCompartmentIDkey = "compartmentId"
	// MetadataKey key name in metadata json for metadata returned by IMDS service
	MetadataKey = "metadata"
	// MetadataUserDataKey key name in metadata json for user data
	MetadataUserDataKey   = "user_data"
	metadataTenancyIDKey  = "oke-tenancy-id"
	metadataPoolIDKey     = "oke-pool-id"
	metadataClusterIDKey  = "oke-cluster-id"
	envPrefix             = "PX_ORACLE"
	envInstanceID         = "INSTANCE_ID"
	envRegion             = "INSTANCE_REGION"
	envAvailabilityDomain = "INSTNACE_AVAILABILITY_DOMAIN"
	envCompartmentID      = "COMPARTMENT_ID"
	envTenancyID          = "TENANCY_ID"
	envPoolID             = "POOL_ID"
	envClusterID          = "CLUSTER_ID"
)

type oracleOps struct {
	cloudops.Compute
	cloudops.Storage
	instance           string
	region             string
	availabilityDomain string
	compartmentID      string
	tenancyID          string
	poolID             string
	clusterID          string
	storage            core.BlockstorageClient
	compute            core.ComputeClient
	containerEngine    containerengine.ContainerEngineClient
}

// NewClient creates a new cloud operations client for Oracle cloud
func NewClient() (cloudops.Ops, error) {
	oracleOps := &oracleOps{}
	err := getInfoFromMetadata(oracleOps)
	if err != nil {
		fmt.Printf("Got error [%v] from metadata\n", err)
		err = getInfoFromEnv(oracleOps)
		if err != nil {
			return nil, err
		}
	}
	os.Setenv(fmt.Sprintf("%s_tenancy_ocid", envPrefix), oracleOps.tenancyID)
	os.Setenv(fmt.Sprintf("%s_region", envPrefix), oracleOps.region)
	configProvider := common.ConfigurationProviderEnvironmentVariables(envPrefix, "")
	oracleOps.storage, err = core.NewBlockstorageClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	oracleOps.compute, err = core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	oracleOps.containerEngine, err = containerengine.NewContainerEngineClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}

	// TODO: [PWX-18717] wrap around exponentialBackoffOps
	return oracleOps, nil
}

func getInfoFromEnv(oracleOps *oracleOps) error {
	var err error
	oracleOps.instance, err = cloudops.GetEnvValueStrict(envInstanceID)
	if err != nil {
		return err
	}

	oracleOps.region, err = cloudops.GetEnvValueStrict(envRegion)
	if err != nil {
		return err
	}

	oracleOps.availabilityDomain, err = cloudops.GetEnvValueStrict(envAvailabilityDomain)
	if err != nil {
		return err
	}

	oracleOps.compartmentID, err = cloudops.GetEnvValueStrict(envCompartmentID)
	if err != nil {
		return err
	}

	oracleOps.tenancyID, err = cloudops.GetEnvValueStrict(envTenancyID)
	if err != nil {
		return err
	}

	oracleOps.poolID, err = cloudops.GetEnvValueStrict(envPoolID)
	if err != nil {
		return err
	}

	oracleOps.clusterID, err = cloudops.GetEnvValueStrict(envClusterID)
	if err != nil {
		return err
	}
	return nil
}

func getRequest(endpoint string, headers map[string]string) (map[string]interface{}, int, error) {
	metadata := make(map[string]interface{})
	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return metadata, 0, err
	}

	for headerKey, headerValue := range headers {
		req.Header.Add(headerKey, headerValue)
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Errorf("metadata lookup from [%s] endpoint failed with error:[%v]", endpoint, err)
		if resp != nil {
			return metadata, resp.StatusCode, errMsg
		}
		return metadata, http.StatusNotFound, errMsg
	}
	if resp.StatusCode != http.StatusOK {
		return metadata, resp.StatusCode, nil
	}
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return metadata, resp.StatusCode,
				fmt.Errorf("error while reading Oracle metadata response: [%v]", err)
		}
		if len(respBody) == 0 {
			return metadata, resp.StatusCode,
				fmt.Errorf("error querying Oracle metadata: Empty response")
		}

		err = json.Unmarshal(respBody, &metadata)
		if err != nil {
			return metadata, resp.StatusCode,
				fmt.Errorf("error parsing Oracle metadata: %v", err)
		}
	}
	return metadata, resp.StatusCode, nil
}

// GetMetadata returns metadata from IMDS
func GetMetadata() (map[string]interface{}, error) {
	httpHeaders := map[string]string{}
	httpHeaders["Authorization"] = "Bearer Oracle"
	var httpStatusCode int
	var err error
	var metadata map[string]interface{}
	metadata, httpStatusCode, err = getRequest(v2MetadataAPIEndpoint, httpHeaders)
	if err != nil {
		return nil, err
	}

	if httpStatusCode != http.StatusOK {
		logrus.Warnf("Trying %s endpoint as got %d http response from %s\n",
			v1MetadataAPIEndpoint, httpStatusCode, v2MetadataAPIEndpoint)
		metadata, httpStatusCode, err = getRequest(v1MetadataAPIEndpoint, map[string]string{})
		if err != nil {
			return nil, err
		}
	}
	if httpStatusCode != http.StatusOK {
		return metadata,
			fmt.Errorf("error:[%v] got HTTP Code %d", err, httpStatusCode)
	}
	return metadata, nil
}

func getInfoFromMetadata(oracleOps *oracleOps) error {
	var tenancyID, poolID, clusterID string
	var ok bool
	metadata, err := GetMetadata()
	if err != nil {
		return err
	}
	if metadata[MetadataKey] != nil {
		if okeMetadata, ok := metadata[MetadataKey].(map[string]interface{}); ok {
			if okeMetadata[metadataTenancyIDKey] != nil {
				if tenancyID, ok = okeMetadata[metadataTenancyIDKey].(string); !ok {
					return fmt.Errorf("can not get tenancy ID from oracle metadata service. error: [%v]", err)
				}
				if poolID, ok = okeMetadata[metadataPoolIDKey].(string); !ok {
					return fmt.Errorf("can not get pool ID from oracle metadata service. error: [%v]", err)
				}
				if clusterID, ok = okeMetadata[metadataClusterIDKey].(string); !ok {
					return fmt.Errorf("can not get cluster ID from oracle metadata service. error: [%v]", err)
				}
			}
		} else {
			return fmt.Errorf("can not get OKE metadata from oracle metadata service. error: [%v]", err)
		}
	}
	oracleOps.tenancyID = tenancyID
	oracleOps.poolID = poolID
	oracleOps.clusterID = clusterID
	if oracleOps.instance, ok = metadata[metadataInstanceIDkey].(string); !ok {
		return fmt.Errorf("can not get instance id from oracle metadata service. error: [%v]", err)
	}
	if oracleOps.region, ok = metadata[MetadataRegionKey].(string); !ok {
		return fmt.Errorf("can not get region from oracle metadata service. error: [%v]", err)
	}
	if oracleOps.availabilityDomain, ok = metadata[MetadataAvailabilityDomainKey].(string); !ok {
		return fmt.Errorf("can not get instance availability domain from oracle metadata service. error: [%v]", err)
	}
	if oracleOps.compartmentID, ok = metadata[MetadataCompartmentIDkey].(string); !ok {
		return fmt.Errorf("can not get compartment ID from oracle metadata service. error: [%v]", err)
	}
	return nil
}

func (o *oracleOps) Name() string { return string(cloudops.Oracle) }

func (o *oracleOps) InstanceID() string { return o.instance }

func (o *oracleOps) InspectInstance(instanceID string) (*cloudops.InstanceInfo, error) {

	instance := core.GetInstanceRequest{
		InstanceId: &instanceID,
	}
	resp, err := o.compute.GetInstance(context.Background(), instance)
	if err != nil {
		return nil, err
	}

	return &cloudops.InstanceInfo{
		CloudResourceInfo: cloudops.CloudResourceInfo{
			Name:   string(*resp.DisplayName),
			ID:     instanceID,
			Region: *resp.Region,
			Zone:   *resp.AvailabilityDomain,
		},
	}, nil
}

func (o *oracleOps) InspectInstanceGroupForInstance(instanceID string) (*cloudops.InstanceGroupInfo, error) {
	getNodePoolReq := containerengine.GetNodePoolRequest{
		NodePoolId: &o.poolID,
	}

	nodePoolDetails, err := o.containerEngine.GetNodePool(context.Background(), getNodePoolReq)
	if err != nil {
		return nil, err
	}

	zones := []string{}
	for _, placementConfig := range nodePoolDetails.NodePool.NodeConfigDetails.PlacementConfigs {
		zones = append(zones, *placementConfig.AvailabilityDomain)
	}
	size := int64(*nodePoolDetails.NodeConfigDetails.Size)

	return &cloudops.InstanceGroupInfo{
		CloudResourceInfo: cloudops.CloudResourceInfo{
			Name:   *nodePoolDetails.Name,
			ID:     o.poolID,
			Region: o.region,
		},
		Min:   &size,
		Max:   &size,
		Zones: zones,
	}, nil
}

func (o *oracleOps) Describe() (interface{}, error) {
	getInstanceReq := core.GetInstanceRequest{
		InstanceId: &o.instance,
	}
	resp, err := o.compute.GetInstance(context.Background(), getInstanceReq)
	if err != nil {
		return nil, err
	}
	return resp.Instance, nil
}

func (o *oracleOps) DeviceMappings() (map[string]string, error) {
	m := make(map[string]string)
	var devicePath, volID string
	volumeAttachmentReq := core.ListVolumeAttachmentsRequest{
		InstanceId: common.String(o.instance),
	}
	volumeAttachmentResp, err := o.compute.ListVolumeAttachments(context.Background(), volumeAttachmentReq)
	if err != nil {
		return m, err
	}
	for _, va := range volumeAttachmentResp.Items {
		if va.GetDevice() != nil && va.GetVolumeId() != nil {
			devicePath = *va.GetDevice()
			volID = *va.GetVolumeId()
		} else {
			logrus.Warnf("Device path or volume id for [%+v] volume attachment not found", va)
			continue
		}
		m[devicePath] = volID
	}
	return m, nil
}

func (o *oracleOps) DevicePath(volumeID string) (string, error) {
	volumeAttachmentReq := core.ListVolumeAttachmentsRequest{
		VolumeId: common.String(volumeID),
	}
	volumeAttachmentResp, err := o.compute.ListVolumeAttachments(context.Background(), volumeAttachmentReq)
	if err != nil {
		return "", err
	}

	if volumeAttachmentResp.Items == nil || len(volumeAttachmentResp.Items) == 0 {
		return "", cloudops.NewStorageError(cloudops.ErrVolDetached,
			"Volume is detached", volumeID)
	}
	volumeAttachment := volumeAttachmentResp.Items[0]
	if volumeAttachment.GetInstanceId() == nil {
		return "", cloudops.NewStorageError(cloudops.ErrVolInval,
			"Unable to determine volume instance attachment", "")
	}
	if o.instance != *volumeAttachment.GetInstanceId() {
		return "", cloudops.NewStorageError(cloudops.ErrVolAttachedOnRemoteNode,
			fmt.Sprintf("Volume attached on %q current instance %q",
				*volumeAttachment.GetInstanceId(), o.instance),
			*volumeAttachment.GetInstanceId())
	}

	if volumeAttachment.GetLifecycleState() != core.VolumeAttachmentLifecycleStateAttached {
		return "", cloudops.NewStorageError(cloudops.ErrVolInval,
			fmt.Sprintf("Invalid state %q, volume is not attached",
				volumeAttachment.GetLifecycleState()), "")
	}
	if volumeAttachment.GetDevice() == nil {
		return "", cloudops.NewStorageError(cloudops.ErrVolInval,
			"Unable to determine volume attachment path", "")
	}
	return *volumeAttachment.GetDevice(), nil
}

// Inspect volumes specified by volumeID
func (o *oracleOps) Inspect(volumeIds []*string) ([]interface{}, error) {
	oracleVols := []interface{}{}
	for _, volID := range volumeIds {
		getVolReq := core.GetVolumeRequest{
			VolumeId: volID,
		}
		getVolResp, err := o.storage.GetVolume(context.Background(), getVolReq)
		if err != nil {
			return nil, err
		}
		oracleVols = append(oracleVols, getVolResp.Volume)
	}
	return oracleVols, nil
}

// Create volume based on input template volume and also apply given labels.
func (o *oracleOps) Create(template interface{}, labels map[string]string) (interface{}, error) {
	vol, ok := template.(core.Volume)
	if !ok {
		return nil, cloudops.NewStorageError(cloudops.ErrVolInval,
			"Invalid volume template given", "")
	}

	createVolReq := core.CreateVolumeRequest{
		CreateVolumeDetails: core.CreateVolumeDetails{
			CompartmentId:      &o.compartmentID,
			AvailabilityDomain: &o.availabilityDomain,
			SizeInGBs:          vol.SizeInGBs,
			VpusPerGB:          vol.VpusPerGB,
			DisplayName:        vol.DisplayName,
			FreeformTags:       labels,
		},
	}
	createVolResp, err := o.storage.CreateVolume(context.Background(), createVolReq)
	if err != nil {
		return nil, err
	}

	oracleVol, err := o.waitVolumeStatus(*createVolResp.Id, core.VolumeLifecycleStateAvailable)
	if err != nil {
		return nil, o.rollbackCreate(*createVolResp.Id, err)
	}
	return oracleVol, nil
}

func (o *oracleOps) waitVolumeStatus(volID string, desiredStatus core.VolumeLifecycleStateEnum) (interface{}, error) {
	getVolReq := core.GetVolumeRequest{
		VolumeId: &volID,
	}
	f := func() (interface{}, bool, error) {
		getVolResp, err := o.storage.GetVolume(context.Background(), getVolReq)
		if err != nil {
			return nil, true, err
		}
		if getVolResp.Volume.LifecycleState == core.VolumeLifecycleStateAvailable {
			return getVolResp.Volume, false, nil
		}

		logrus.Debugf("volume [%s] is still in [%s] state", volID, getVolResp.Volume.LifecycleState)
		return nil, true, fmt.Errorf("volume [%s] is still in [%s] state", volID, getVolResp.Volume.LifecycleState)
	}
	oracleVol, err := task.DoRetryWithTimeout(f, cloudops.ProviderOpsTimeout, cloudops.ProviderOpsRetryInterval)
	return oracleVol, err
}

func (o *oracleOps) rollbackCreate(id string, createErr error) error {
	logrus.Warnf("Rollback create volume %v, Error %v", id, createErr)
	err := o.Delete(id)
	if err != nil {
		logrus.Warnf("Rollback failed volume %v, Error %v", id, err)
	}
	return createErr
}

// Delete volumeID.
func (o *oracleOps) Delete(volumeID string) error {
	delVolReq := core.DeleteVolumeRequest{
		VolumeId: &volumeID,
	}
	delVolResp, err := o.storage.DeleteVolume(context.Background(), delVolReq)
	if err != nil {
		logrus.Errorf("failed to delete volume [%s]. Response: [%v], Error: [%v]", volumeID, delVolResp, err)
		return err
	}
	return nil
}

func (o *oracleOps) SetInstanceGroupSize(instanceGroupID string, count int64, timeout time.Duration) error {

	if timeout == 0*time.Second {
		timeout = 5 * time.Minute
	}

	instanceGroupSize := int(count)

	//get nodepool by ID to be updated
	nodePoolReq := containerengine.ListNodePoolsRequest{CompartmentId: &o.compartmentID, Name: &instanceGroupID, ClusterId: &o.clusterID}
	nodePools, err := o.containerEngine.ListNodePools(context.Background(), nodePoolReq)
	if err != nil {
		return err
	}

	if len(nodePools.Items) == 0 {
		return errors.New("No node pool found with name " + instanceGroupID)
	}
	numberOfDomains := len(nodePools.Items[0].NodeConfigDetails.PlacementConfigs)
	totalClusterSize := numberOfDomains * instanceGroupSize
	logrus.Println("Setting instanceGroupSize to ", totalClusterSize, " in total ", numberOfDomains, " regions.")

	//get all availabliity domain
	nodePoolPlacementConfigDetails := make([]containerengine.NodePoolPlacementConfigDetails, numberOfDomains)

	for i, placementConfigs := range nodePools.Items[0].NodeConfigDetails.PlacementConfigs {
		nodePoolPlacementConfigDetails[i].AvailabilityDomain = placementConfigs.AvailabilityDomain
		nodePoolPlacementConfigDetails[i].SubnetId = placementConfigs.SubnetId
	}

	//update node pools
	req := containerengine.UpdateNodePoolRequest{
		NodePoolId: nodePools.Items[0].Id, //get node pool id
		UpdateNodePoolDetails: containerengine.UpdateNodePoolDetails{
			NodeConfigDetails: &containerengine.UpdateNodePoolNodeConfigDetails{
				Size:             &totalClusterSize,
				PlacementConfigs: nodePoolPlacementConfigDetails,
			},
		},
	}

	resp, err := o.containerEngine.UpdateNodePool(context.Background(), req)
	if err != nil {
		return err
	}

	err = o.waitTillWorkStatusIsSucceeded(resp.OpcRequestId, resp.OpcWorkRequestId, timeout)
	if err != nil {
		return err
	}

	return nil
}

func (o *oracleOps) waitTillWorkStatusIsSucceeded(opcRequestID, opcWorkRequestID *string, timeout time.Duration) error {
	workReq := containerengine.GetWorkRequestRequest{OpcRequestId: opcRequestID,
		WorkRequestId: opcWorkRequestID}

	f := func() (interface{}, bool, error) {
		workResp, err := o.containerEngine.GetWorkRequest(context.Background(), workReq)
		if err != nil {
			return nil, true, err
		}

		if workResp.Status == containerengine.WorkRequestStatusSucceeded {
			return workResp.Status, false, nil
		}

		logrus.Debugf("Work status is in [%s] state", workResp.Status)
		return nil, true, fmt.Errorf("Work status is in [%s] state", workResp.Status)
	}
	_, err := task.DoRetryWithTimeout(f, timeout, 10*time.Second)
	return err
}

func (o *oracleOps) GetInstanceGroupSize(instanceGroupID string) (int64, error) {

	var count int64

	nodePoolReq := containerengine.ListNodePoolsRequest{CompartmentId: &o.compartmentID, Name: &instanceGroupID, ClusterId: &o.clusterID}
	nodePools, err := o.containerEngine.ListNodePools(context.Background(), nodePoolReq)
	if err != nil {
		return 0, err
	}

	if len(nodePools.Items) == 0 {
		return 0, errors.New("No node pool found with name " + instanceGroupID)
	}

	req := containerengine.GetNodePoolRequest{NodePoolId: nodePools.Items[0].Id}

	resp, err := o.containerEngine.GetNodePool(context.Background(), req)

	if err != nil {
		return 0, err
	}

	for _, node := range resp.Nodes {
		if node.LifecycleState == containerengine.NodeLifecycleStateActive {
			count++
		}
	}

	return count, nil
}
