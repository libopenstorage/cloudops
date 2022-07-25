package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/libopenstorage/cloudops"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/sirupsen/logrus"
)

const (
	v1MetadataAPIEndpoint         = "http://169.254.169.254/opc/v1/instance/"
	v2MetadataAPIEndpoint         = "http://169.254.169.254/opc/v2/instance/"
	metadataInstanceIDkey         = "id"
	metadataRegionKey             = "region"
	metadataAvailabilityDomainKey = "availabilityDomain"
	metadataCompartmentIDkey      = "compartmentId"
	metadataKey                   = "metadata"
	metadataTenancyIDKey          = "oke-tenancy-id"
	metadataPoolIDKey             = "oke-pool-id"
	envInstanceID                 = "ORACLE_INSTANCE_ID"
	envRegion                     = "ORACLE_INSTANCE_REGION"
	envAvailabilityDomain         = "ORACLE_INSTNACE_AVAILABILITY_DOMAIN"
	envCompartmentID              = "ORACLE_COMPARTMENT_ID"
	envTenancyID                  = "ORACLE_TENANCY_ID"
	envPoolID                     = "ORACLE_POOL_ID"
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
	storage            core.BlockstorageClient
	compute            core.ComputeClient
	containerEngine    containerengine.ContainerEngineClient
}

// NewClient creates a new cloud operations client for Oracle cloud
func NewClient() (cloudops.Ops, error) {
	instance, region, availabilityDomain, compartmentID, tenancyID, poolID, err := getInfoFromMetadata()
	if err != nil {
		instance, region, availabilityDomain, compartmentID, tenancyID, poolID, err = getInfoFromEnv()
		if err != nil {
			return nil, err
		}
	}
	os.Setenv("ORACLE_tenancy_ocid", tenancyID)
	os.Setenv("ORACLE_region", region)
	configProvider := common.ConfigurationProviderEnvironmentVariables("ORACLE", "")
	storage, err := core.NewBlockstorageClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	compute, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	containerEngine, err := containerengine.NewContainerEngineClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}

	// TODO: wrap around exponentialBackoffOps
	return &oracleOps{
		compute:            compute,
		storage:            storage,
		containerEngine:    containerEngine,
		instance:           instance,
		region:             region,
		availabilityDomain: availabilityDomain,
		compartmentID:      compartmentID,
		tenancyID:          tenancyID,
		poolID:             poolID,
	}, nil
}

func getInfoFromEnv() (string, string, string, string, string, string, error) {
	instance, err := cloudops.GetEnvValueStrict(envInstanceID)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	region, err := cloudops.GetEnvValueStrict(envRegion)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	availabilityDomain, err := cloudops.GetEnvValueStrict(envAvailabilityDomain)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	compartmentID, err := cloudops.GetEnvValueStrict(envCompartmentID)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	tenancyID, err := cloudops.GetEnvValueStrict(envTenancyID)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	poolID, err := cloudops.GetEnvValueStrict(envPoolID)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	return instance, region, availabilityDomain, compartmentID, tenancyID, poolID, nil
}

func getMetadata() (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	client := &http.Client{}
	v2req, err := http.NewRequest("GET", v2MetadataAPIEndpoint, nil)
	if err != nil {
		return metadata, err
	}
	v2req.Header.Add("Authorization", "Bearer Oracle")
	q := v2req.URL.Query()
	v2req.URL.RawQuery = q.Encode()

	resp, err := client.Do(v2req)
	if err != nil {
		logrus.Warnf("metadata lookup from IMDS v1 endpoint failed")
		return metadata,
			fmt.Errorf("error occured while getting instance metadata from Oracle Metadata API. Error:[%v]", err)
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			// v2 IMDS endpoint not enabled, try v1
			v1req, err := http.NewRequest("GET", v1MetadataAPIEndpoint, nil)
			if err != nil {
				return metadata, err
			}
			q := v2req.URL.Query()
			v1req.URL.RawQuery = q.Encode()

			resp, err = client.Do(v1req)
			if err != nil {
				logrus.Warnf("metadata lookup from IMDS v1 endpoint failed")
				return metadata,
					fmt.Errorf("error occured while getting instance metadata from Oracle Metadata API. Error:[%v]", err)
			}
			if resp.StatusCode != 200 {
				return metadata,
					fmt.Errorf("error querying Oracle metadata service: Code %d returned for url %s", resp.StatusCode, v1req.URL)
			}
		} else {
			return metadata,
				fmt.Errorf("error querying Oracle metadata service: Code %d returned for url %s", resp.StatusCode, v2req.URL)
		}
	}

	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return metadata,
				fmt.Errorf("error while reading Oracle metadata response: [%v]", err)
		}
		if len(respBody) == 0 {
			return metadata,
				fmt.Errorf("error querying Oracle metadata: Empty response")
		}

		err = json.Unmarshal(respBody, &metadata)
		if err != nil {
			return metadata,
				fmt.Errorf("error parsing Oracle metadata: %v", err)
		}
	}
	return metadata, nil
}
func getInfoFromMetadata() (string, string, string, string, string, string, error) {
	var tenancyID, poolID string
	var ok bool
	metadata, err := getMetadata()
	if err != nil {
		return "", "", "", "", "", "", err
	}
	if metadata[metadataKey] != nil {
		if okeMetadata, ok := metadata[metadataKey].(map[string]interface{}); ok {
			if okeMetadata[metadataTenancyIDKey] != nil {
				if tenancyID, ok = okeMetadata[metadataTenancyIDKey].(string); !ok {
					return "", "", "", "", "", "",
						fmt.Errorf("can not get tenancy ID from oracle metadata service. error: [%v]", err)
				}
				if poolID, ok = okeMetadata[metadataPoolIDKey].(string); !ok {
					return "", "", "", "", "", "",
						fmt.Errorf("can not get tenancy ID from oracle metadata service. error: [%v]", err)
				}
			}
		} else {
			return "", "", "", "", "", "",
				fmt.Errorf("can not get OKE metadata from oracle metadata service. error: [%v]", err)
		}
	}

	var instanceID, region, availabilityDomain, compartmentID string

	if instanceID, ok = metadata[metadataInstanceIDkey].(string); !ok {
		return "", "", "", "", "", "",
			fmt.Errorf("can not get instance id from oracle metadata service. error: [%v]", err)
	}
	if region, ok = metadata[metadataRegionKey].(string); !ok {
		return "", "", "", "", "", "",
			fmt.Errorf("can not get region from oracle metadata service. error: [%v]", err)
	}
	if availabilityDomain, ok = metadata[metadataAvailabilityDomainKey].(string); !ok {
		return "", "", "", "", "", "",
			fmt.Errorf("can not get instance availability domain from oracle metadata service. error: [%v]", err)
	}
	if compartmentID, ok = metadata[metadataCompartmentIDkey].(string); !ok {
		return "", "", "", "", "", "",
			fmt.Errorf("can not get compartment ID from oracle metadata service. error: [%v]", err)
	}

	return instanceID, region,
		availabilityDomain,
		compartmentID,
		tenancyID, poolID,
		nil
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

	// TODO: get labels from tags
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
	// TODO: Populate labels from tags
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
	return resp.Instance, err
}
