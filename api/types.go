package api

import (
	"time"
)

// CloudVolumeSpec defines volume spec.
type CloudVolumeSpec struct {
	// VolumeType the volume type (e.g. GP2 on AWS)
	VolumeType *string `type:"string" enum:"VolumeType"`
	// Size in GiBs.
	Size *int64 `type:"integer"`
	// Availability zone in which to create the volume.
	AvailabilityZone *string `type:"string" required:"true"`
	// IOPS desired IOPS. Optional.
	Iops *int64 `type:"integer"`
	// Encrypted specifies whether the volume should be encrypted. Optional
	Encrypted *bool `locationName:"encrypted" type:"boolean"`
	// KmsKeyID identifier for the Key Management Service and specifies customer master
	// key (CMK) to use when creating the encrypted volume.  Optional
	KmsKeyID *string `type:"string"`
	// SnapshotID the snapshot from which to create the volume. Optional
	SnapshotID *string `type:"string"`
	// The tags to apply to the volume during creation. Optional
	Labels map[string]string `locationName:"Labels"`
}

// VolumeAttachmentState enum for current volume attachment state.
type VolumeAttachmentState string

const (
	// VolumeAttachmentStateAttaching volume is attaching.
	VolumeAttachmentStateAttaching VolumeAttachmentState = "attaching"
	// VolumeAttachmentStateAttached volume is attached.
	VolumeAttachmentStateAttached VolumeAttachmentState = "attached"
	// VolumeAttachmentStateDetaching volume is detaching.
	VolumeAttachmentStateDetaching VolumeAttachmentState = "detaching"
	// VolumeAttachmentStateDetached volume is detached.
	VolumeAttachmentStateDetached VolumeAttachmentState = "detached"
)

// CloudVolumeAttachment runtime volume attachment status.
type CloudVolumeAttachment struct {
	// AttachTime the time stamp when the attachment initiated.
	AttachTime *time.Time `locationName:"attachTime" type:"timestamp"`
	// DeviceName the device name.
	DeviceName *string `locationName:"device" type:"string"`
	// InstanceID unique instance identifier.
	InstanceID *string `locationName:"instanceID" type:"string"`
	// The attachment state of the volume.
	State *string `locationName:"status" type:"string" enum:"VolumeAttachmentState"`
	// The ID of the volume.
	VolumeID *string `locationName:"volumeID" type:"string"`
}

// CloudVolume runtime status of CloudVolume.
type CloudVolume struct {
	// VolumeID unique identifier for the volume.
	VolumeID *string `locationName:"volumeID" type:"string"`
	// Attachement information
	Attachment *CloudVolumeAttachment `locationName:"attachment"`
	// AvailabilityZone for the volume.
	AvailabilityZone *string `locationName:"availabilityZone" type:"string"`
	// CreateTime the time stamp when volume creation was initiated.
	CreateTime *time.Time `locationName:"createTime" type:"timestamp"`
	// Encrypted indicates whether the volume will be encrypted.
	Encrypted *bool `locationName:"encrypted" type:"boolean"`
	// Iops provisioned for this volume.
	Iops *int64 `locationName:"iops" type:"integer"`
	// KmsKeyID the full ARN of the Key Management Service, customer master
	// key (CMK) that was used to protect the volume encryption key for the volume.
	KmsKeyID *string `locationName:"kmsKeyID" type:"string"`
	// Size in GiBs.
	Size *int64 `locationName:"size" type:"integer"`
	// SnapshotID from which the volume was created, if applicable.
	SnapshotID *string `locationName:"snapshotID" type:"string"`
	// State The volume state.
	State *string `locationName:"status" type:"string" enum:"VolumeState"`
	// Labels assigned to the volume.
	Labels map[string]string `locationName:"labels" locationNameList:"item" type:"list"`
	// VolumeType the type of the volume e.g. GP2j
	VolumeType *string `locationName:"volumeType" type:"string" enum:"VolumeType"`
	// contains filtered or unexported fields
}
