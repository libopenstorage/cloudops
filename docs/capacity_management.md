# Overview

Selecting storage drives depends on a number of factors:

-  *Workload* (random/sequential) determines the drive category: spinning media or solid state.
-  *IOPS*  Cloud providers usually dictate the minimum drive size required to achieve certain IOPS
-  *Number of Drives per instance* drives may have individual network connetion. Striping across two drives is sometimes a better decision than allocating a large single drive. This property holds true only upto a certain number of drives per instance. It also depends upon the instance type and drive type.
-  *Instance Type* Not all drive types are supported on all instance types
-  *Zone/Region* Not all zones or regions support all types of drives.

In summary, to determine a set of storage drives to provision depends on the right drive type, size, the number of drives per instance, the instance type, the zone and the region.

An example is the EBS volume matrix:https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSVolumeTypes.html

In distributed heterogeneous application deployments, the requirements are at a cluster level and not at a node level. The requirements also differ based on the category of applications. They also evolve as the cluster grows or as the deployment grows.

The *CloudOps Drive Manangement* library insulates the operator from cloud specific nuances and translates high level requirements for storage capacity and performance to cloud specific storage resource management.


# Goal

- Provide a drive management library that takes in high level requirements for cluster wide capacity and performance and recommends drive configuration.

- The input parameters should be cloud agnostic.

- The library should be extensible to all clouds.

- The cloud storage matrix definition should be configurable.

- The capacity management should include cost analysis.


# Cloud Storage Decision Matrix

The cloud storage decision matrix dictates the drive configuration choices. This configuration is provided as Yaml/JSON to the cloud management library. There will be a cloud matrix per provider.

A typical entry in the decision matrix for a cloud will involve the following fields:

```go
// CloudStorage defines an entry in the cloud storage decision matrix.
type CloudStorage struct {
        // IOPS is the desired iops from the underlying cloud storage.
        IOPS              uint32   `json:"iops" yaml:"iops"`
        // InstanceType is the type of instance on which the cloud storage will
        // be attached.
        InstanceType      string   `json:"instance_type" yaml:"instance_type"`
        // InstanceMaxDrives is the maximum number of drives that can be attached
        // to an instance without a performance hit.
        InstanceMaxDrives uint32   `json:"instance_max_drives" yaml:"instance_max_drives"`
        // InstanceMinDrives is the minimum number of drives that need to be
        // attached to an instance to achieve maximum performance.
        InstanceMinDrives uint32   `json:"instance_min_drives" yaml:"instance_min_drives"`
        // Region of the instance.
        Region            string   `json:"region" yaml:"region"`
        // MinSize is the minimum size of the drive that needs to be provisioned
        // to achieve the desired IOPS on the provided instance types.
        MinSize           uint64   `json:"min_size" yaml:"min_size"`
        // MaxSize is the maximum size of the drive that can be provisioned
        // without affecting performance on the provided instance type.
        MaxSize           uint64   `json:"max_size" yaml:"max_size"`
        // Priority for this entry in the decision matrix.
        Priority          string   `json:"priority" yaml:"priority"`
        // ThinProvisioning is set/unset if the cloud provider supports such a
        // storage device.
        ThinProvisioning  bool     `json:"thin_provisioning" yaml:"thin_provisioning"`
}
```

This Cloud Storage Decision Matrix is stored in a cluster wide accessible key/value store (e.g ConfigMap in k8s)

# Cloud Storage Initial Allocation

The input for storage allocation is a provider specific `CloudStorage` in addition to `CloudStorageSpec` defined below

```go
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

```go
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
