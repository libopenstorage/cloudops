
# Overview

Selecting storage drives depends on a number of factors:

-  *Workload* (random/sequential) determines the drive category: spinning media or solid state.
-  *IOPS*  Minimum drives size is dictated by required IOPS 
-  *Number of Drives per instance* drives sometimes have individual network connetion. Striping across two drives is sometimes a better decision than allocating a large single drive. This property holds true only uptopa certain number of drives per instance. It also depends upon the instance type and drive type.
-  *Instance Type* Not all drive types are supported on all instance types
-  *Zone/Region* Not all zones or regions can support all drivs a minimmum drive size is right drive type, size, the number of drives per instance, the instance type, the zone and the region. An example is EBS volume matrix:

References: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSVolumeTypes.html 

In distributed heterogeneous application deployments, the requirements are at a cluster level and not at a node level. The requirements also differ based on the category of applications. They also evolve as the cluster grows or as the deployment grows.

The Cloud drive manangement library insulates the operator from cloud specific nuances and translates high level requirements for storage capacity and performance to cloud specific storage resource management.


# Goal

- Provide a drive management library that takes in high level requirements for cluster wide capacity and performance and recommends drive configuration.

- The input parameters should be cloud agnostic. 

- The library would be extensible to all clouds.

- The cloud matrix definition should be configurable

- The capacity management should also include cost analysis


# Cloud Storage Decision Matrix

The cloud storage decision matrix dictates the drive configuration choices. This configuration is provided as Yaml/JSON to the cloud management library. There will be a cloud matrix per provider.

```
type CloudStorage struct {
        IOPS              uint32   `json:"iops" yaml:"iops"`
        InstanceType      string   `json:"instance_type" yaml:"instance_type"`
        InstanceMaxDrives uint32   `json:"instance_max_drives" yaml:"instance_max_drives"`
        InstanceMinDrives uint32   `json:"instance_min_drives" yaml:"instance_min_drives"`
        Region            string   `json:"region" yaml:"region"`
        MinSize           uint64   `json:"min_size" yaml:"min_size"`
        MaxSize           uint64   `json:"max_size" yaml:"max_size"`
        Priority          string   `json:"priority" yaml:"priority"`
        ThinProvisioning  bool     `json:"thin_provisioning" yaml:"thin_provisioning"`
}
```

This Cloud Storage Decision Matrix is stored in a cluster wide accessible key/value store (e.g ConfigMap in k8s)

# Cloud Storage Initial Allocation

The input for storage allocation is a provider specific `CloudStorage` in addition to CloudStorageSpec defined below

```
type CloudUserSpec struct {
        IOPS                  uint32   `json:"iops" yaml:"iops"`
        MinCapacity           uint64   `json:"min_capacity" yaml:"min_capacity"`
        MaxCapacity           uint64   `json:"max_capacity" yaml:"max_capacity"`
}

type CloudStorageSpec struct {
      UserSpec               CloudUserSpec
      InstanceType           string
      InstancesPerZone       int
      ZoneCount              int
}

```

Its output will be the distribution of drives across zones and nodes.

```
type CloudStorageDistribution struct {
      InstanceStorage struct {
            DriveCapacityGiB          int64
            DriveCount                int   
            DriveType                 string
      }
      InstancesPerZone                int
}

```

Assumption: Storage nodes instance type is homogeneous. 





