package vsphere

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/unsupported"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere/vclib"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere/vclib/diskmanagers"
)

const (
	diskDirectory  = "osd-provisioned-disks"
	dummyDiskName  = "kube-dummyDisk.vmdk"
	diskByIDPath   = "/dev/disk/by-id/"
	diskSCSIPrefix = "wwn-0x"
)

type vsphereOps struct {
	cloudops.Compute
	vm   *vclib.VirtualMachine
	conn *vclib.VSphereConnection
	cfg  *VSphereConfig
}

type VirtualMachineCreateOpts struct {
	// Spec is the config spec for the virtual machine
	Spec *types.VirtualMachineConfigSpec
	// StoragePod is the datastore cluster (aka storage pod to use for the VM)
	StoragePod string
	// Datastore is the VMFS datastore to use for the VM
	Datastore string
	// ResourcePool is the resource pool to use to place to VM
	ResourcePool string
	// Host is the ESXi host for the VM
	Host string
	// Folder is the folder for the VM
	Folder string
	// If true, VM will be powered on after create
	PowerOn bool
	// Datacenter is the datacenter to create the VM in
	Datacenter string
}

// VirtualDisk encapsulates the existing virtual disk object to add a managed object
// reference to the datastore of the disk
type VirtualDisk struct {
	diskmanagers.VirtualDisk
	// DatastoreRef is the managed object reference of the datastore on which the disk belongs
	DatastoreRef types.ManagedObjectReference
}

// NewClient creates a new vsphere cloudops instance
func NewClient(cfg *VSphereConfig) (cloudops.Ops, error) {
	vSphereConn := &vclib.VSphereConnection{
		Username:          cfg.User,
		Password:          cfg.Password,
		Hostname:          cfg.VCenterIP,
		Insecure:          cfg.InsecureFlag,
		RoundTripperCount: cfg.RoundTripperCount,
		Port:              cfg.VCenterPort,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vmObj, err := GetVMObject(ctx, vSphereConn, cfg.VMUUID)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Using following configuration for vsphere:")
	logrus.Debugf("  vCenter: %s:%s", cfg.VCenterIP, cfg.VCenterPort)
	logrus.Debugf("  Datacenter: %s", vmObj.Datacenter.Name())
	logrus.Debugf("  VMUUID: %s", cfg.VMUUID)

	return &vsphereOps{
		Compute: unsupported.NewUnsupportedCompute(),
		cfg:     cfg,
		vm:      vmObj,
		conn:    vSphereConn,
	}, nil
}

func (ops *vsphereOps) Name() string { return string(cloudops.Vsphere) }

func (ops *vsphereOps) InstanceID() string { return ops.cfg.VMUUID }

func (ops *vsphereOps) Create(opts interface{}, labels map[string]string) (interface{}, error) {
	volumeOptions, ok := opts.(*vclib.VolumeOptions)
	if !ok {
		return nil, fmt.Errorf("invalid volume options specified to create: %v", opts)
	}

	if len(volumeOptions.Tags) == 0 {
		volumeOptions.Tags = labels
	} else {
		for k, v := range labels {
			volumeOptions.Tags[k] = v
		}
	}

	if len(volumeOptions.Datastore) == 0 {
		return nil, fmt.Errorf("datastore is required for the create call")
	}

	datastore := strings.TrimSpace(volumeOptions.Datastore)
	logrus.Infof("Given datastore/datastore cluster: %s for new disk", datastore)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vmObj, err := ops.renewVM(ctx, ops.vm)
	if err != nil {
		return nil, err
	}

	isPod, storagePod, err := IsStoragePod(ctx, vmObj, volumeOptions.Datastore)
	if err != nil {
		return nil, err
	}

	if isPod {
		datastore, err = ops.getDatastoreToUseInStoragePod(ctx, vmObj, volumeOptions, storagePod)
		if err != nil {
			return nil, err
		}
	}

	logrus.Infof("Using datastore: %s for new disk", datastore)

	ds, err := vmObj.Datacenter.GetDatastoreByName(ctx, datastore)
	if err != nil {
		logrus.Errorf("Failed to get datastore: %s due to: %v", datastore, err)
		return nil, err
	}

	volumeOptions.Datastore = datastore

	diskBasePath := filepath.Clean(ds.Path(diskDirectory)) + "/"
	err = ds.CreateDirectory(ctx, diskBasePath, false)
	if err != nil && err != vclib.ErrFileAlreadyExist {
		logrus.Errorf("Cannot create dir %#v. err %s", diskBasePath, err)
		return nil, err
	}

	diskPath := diskBasePath + volumeOptions.Name + ".vmdk"
	disk := diskmanagers.VirtualDisk{
		DiskPath:      diskPath,
		VolumeOptions: volumeOptions,
	}

	diskPath, err = disk.Create(ctx, ds)
	if err != nil {
		logrus.Errorf("Failed to create a vsphere volume with volumeOptions: %+v on "+
			"datastore: %s. err: %+v", volumeOptions, datastore, err)
		return nil, err
	}

	// Get the canonical path for the volume path.
	canonicalVolumePath, err := getCanonicalVolumePath(ctx, vmObj.Datacenter, diskPath)
	if err != nil {
		logrus.Errorf("Failed to get canonical vsphere disk path for: %s with "+
			"volumeOptions: %+v on datastore: %s. err: %+v", diskPath, volumeOptions, datastore, err)
		return nil, err
	}

	disk.DiskPath = canonicalVolumePath

	return &VirtualDisk{
		VirtualDisk:  disk,
		DatastoreRef: ds.Reference(),
	}, nil
}

func (ops *vsphereOps) GetDeviceID(vDisk interface{}) (string, error) {
	disk, ok := vDisk.(*VirtualDisk)
	if !ok {
		return "", fmt.Errorf("invalid input: %v to GetDeviceID", vDisk)
	}

	return disk.DiskPath, nil
}

// Attach takes in the path of the vmdk file and returns where it is attached inside the vm instance
// The disk is attach on the VM where is the package is running
func (ops *vsphereOps) Attach(diskPath string, options map[string]string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vmObj, err := ops.renewVM(ctx, ops.vm)
	if err != nil {
		return "", err
	}

	return ops.attachDiskToVM(ctx, vmObj, diskPath, options)

}

func (ops *vsphereOps) attachDiskToVM(
	ctx context.Context,
	vmObj *vclib.VirtualMachine,
	diskPath string,
	options map[string]string) (string, error) {
	devices, err := vmObj.Device(ctx)
	if err != nil {
		return "", err
	}

	ds, err := vmObj.Datacenter.GetDatastoreByPath(ctx, diskPath)
	if err != nil {
		logrus.Errorf("Failed to get datastore from diskPath: %q. err: %+v", diskPath, err)
		return "", err
	}

	disk, newSCSIController, err := vmObj.CreateDiskSpec(
		ctx,
		diskPath,
		ds,
		&vclib.VolumeOptions{SCSIControllerType: vclib.PVSCSIControllerType})
	if err != nil {
		logrus.Errorf("Failed to create disk spec for: %s due to err: %v", diskPath, err)
		return "", err
	}

	if options != nil {
		if sharingMode, present := options["sharingMode"]; present && len(sharingMode) > 0 {
			backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
			backing.Sharing = sharingMode
		}
	}

	vmName, err := vmObj.ObjectName(ctx)
	if err != nil {
		return "", err
	}

	//disk = devices.ChildDisk(disk)
	if err = vmObj.AddDevice(ctx, disk); err != nil {
		logrus.Errorf("Failed to attach vsphere disk: %s for VM: %s. err: +%v", diskPath, vmName, err)
		return "", err
	}

	// Once disk is attached, get the disk UUID.
	diskUUID, err := vmObj.Datacenter.GetVirtualDiskPage83Data(ctx, diskPath)
	if err != nil {
		logrus.Errorf("Error occurred while getting Disk Info from VM: %q. err: %v", vmObj.InventoryPath, err)
		vmObj.DetachDisk(ctx, diskPath)
		if newSCSIController != nil {
			ops.deleteController(ctx, vmObj, newSCSIController, devices)
		}

		return "", err
	}

	return path.Join(diskByIDPath, diskSCSIPrefix+diskUUID), nil
}

func (ops *vsphereOps) AttachByInstanceID(
	instanceID, diskPath string,
	options map[string]string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vmObj, err := GetVMObject(ctx, ops.conn, instanceID)
	if err != nil {
		return "", err
	}

	return ops.attachDiskToVM(ctx, vmObj, diskPath, options)
}

// deleteController removes latest added SCSI controller from VM.
func (ops *vsphereOps) deleteController(
	ctx context.Context,
	vm *vclib.VirtualMachine,
	controllerDevice types.BaseVirtualDevice,
	vmDevices object.VirtualDeviceList) error {
	controllerDeviceList := vmDevices.SelectByType(controllerDevice)
	if len(controllerDeviceList) < 1 {
		return vclib.ErrNoDevicesFound
	}
	device := controllerDeviceList[len(controllerDeviceList)-1]
	err := vm.RemoveDevice(ctx, true, device)
	if err != nil {
		logrus.Errorf("Error occurred while removing device on VM: %q. err: %+v", vm.InventoryPath, err)
		return err
	}
	return nil
}

func (ops *vsphereOps) Detach(diskPath string) error {
	return ops.detachInternal(diskPath, ops.cfg.VMUUID)
}

func (ops *vsphereOps) DetachFrom(diskPath, instanceID string) error {
	return ops.detachInternal(diskPath, instanceID)
}

func (ops *vsphereOps) detachInternal(diskPath, instanceID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var vmObj *vclib.VirtualMachine
	var err error
	if instanceID == ops.cfg.VMUUID {
		vmObj, err = ops.renewVM(ctx, ops.vm)
		if err != nil {
			return err
		}
	} else {
		vmObj, err = GetVMObject(ctx, ops.conn, instanceID)
		if err != nil {
			return err
		}
	}

	vmName, err := vmObj.ObjectName(ctx)
	if err != nil {
		return err
	}

	if err := vmObj.DetachDisk(ctx, diskPath); err != nil {
		err = fmt.Errorf("failed to detach vsphere disk: %s for VM: %s. err: +%v", diskPath, vmName, err)
		return err
	}

	return nil
}

// Delete virtual disk at given path
func (ops *vsphereOps) Delete(diskPath string) error {
	return ops.deleteInternal(diskPath, ops.cfg.VMUUID)
}

func (ops *vsphereOps) DeleteFrom(diskPath, instanceID string) error {
	return ops.deleteInternal(diskPath, instanceID)
}

func (ops *vsphereOps) deleteInternal(diskPath, instanceID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var vmObj *vclib.VirtualMachine
	var err error
	if instanceID == ops.cfg.VMUUID {
		vmObj, err = ops.renewVM(ctx, ops.vm)
		if err != nil {
			return err
		}
	} else {
		vmObj, err = GetVMObject(ctx, ops.conn, instanceID)
		if err != nil {
			return err
		}
	}

	disk := diskmanagers.VirtualDisk{
		DiskPath:      diskPath,
		VolumeOptions: &vclib.VolumeOptions{},
		VMOptions:     &vclib.VMOptions{},
	}

	err = disk.Delete(ctx, vmObj.Datacenter)
	if err != nil {
		logrus.Errorf("Failed to delete vsphere disk: %s. err: %+v", diskPath, err)
	}

	return err
}

// Desribe an instance of the virtual machine object to which ops is connected to
func (ops *vsphereOps) Describe() (interface{}, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return ops.renewVM(ctx, ops.vm)
}

// FreeDevices is not supported by this provider
func (ops *vsphereOps) FreeDevices(blockDeviceMappings []interface{}, rootDeviceName string) ([]string, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "FreeDevices",
	}
}

func (ops *vsphereOps) Inspect(diskPaths []*string) ([]interface{}, error) {
	// TODO find a way to map diskPaths to unattached/attached virtual disks and query info
	// currently returning the disks directly

	return nil, &cloudops.ErrNotSupported{
		Operation: "Inspect",
	}
}

// DeviceMappings returns map[local_attached_volume_path]->volume ID/NAME
func (ops *vsphereOps) DeviceMappings(instanceID string) (map[string]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		vmObj *vclib.VirtualMachine
		err   error
	)

	if len(instanceID) == 0 {
		vmObj, err = ops.renewVM(ctx, ops.vm)
	} else {
		vmObj, err = GetVMObject(ctx, ops.conn, instanceID)
	}
	if err != nil {
		return nil, err
	}

	vmName, err := vmObj.ObjectName(ctx)
	if err != nil {
		return nil, err
	}

	vmDevices, err := vmObj.Device(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices for vm: %s", vmName)
	}

	// Go over all the devices attached on this vm and create a map of just the virtual disks and where
	// they are attached on the vm
	m := make(map[string]string)
	for _, device := range vmDevices {
		if vmDevices.TypeName(device) == "VirtualDisk" {
			virtualDevice := device.GetVirtualDevice()
			backing, ok := virtualDevice.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
			if ok {
				devicePath, err := ops.DevicePath(backing.FileName)
				if err == nil && len(devicePath) != 0 { // TODO can ignore errors?
					m[devicePath] = backing.FileName
				}
			}
		}
	}

	return m, nil
}

// DevicePath for the given volume i.e path where it's attached
func (ops *vsphereOps) DevicePath(diskPath string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vmObj, err := ops.renewVM(ctx, ops.vm)
	if err != nil {
		return "", err
	}

	vmName, err := vmObj.ObjectName(ctx)
	if err != nil {
		return "", err
	}

	attached, err := vmObj.IsDiskAttached(ctx, diskPath)
	if err != nil {
		return "", fmt.Errorf("failed to check if disk: %s is attached on vm: %s. err: %v",
			diskPath, vmName, err)
	}

	if !attached {
		return "", cloudops.NewStorageError(cloudops.ErrVolDetached,
			"disk is not attached on current VM", diskPath)
	}

	diskUUID, err := vmObj.Datacenter.GetVirtualDiskPage83Data(ctx, diskPath)
	if err != nil {
		logrus.Errorf("failed to get device path for disk: %s on vm: %s", diskPath, vmName)
		return "", err
	}

	return path.Join(diskByIDPath, diskSCSIPrefix+diskUUID), nil
}

func (ops *vsphereOps) Enumerate(volumeIds []*string,
	labels map[string]string,
	setIdentifier string,
) (map[string][]interface{}, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "Enumerate",
	}
}

// Snapshot the volume with given volumeID
func (ops *vsphereOps) Snapshot(volumeID string, readonly bool) (interface{}, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "Snapshot",
	}
}

// SnapshotDelete deletes the snapshot with given ID
func (ops *vsphereOps) SnapshotDelete(snapID string) error {
	return &cloudops.ErrNotSupported{
		Operation: "SnapshotDelete",
	}
}

// ApplyTags will apply given labels/tags on the given volume
func (ops *vsphereOps) ApplyTags(volumeID string, labels map[string]string) error {
	return &cloudops.ErrNotSupported{
		Operation: "ApplyTags",
	}
}

// RemoveTags removes labels/tags from the given volume
func (ops *vsphereOps) RemoveTags(volumeID string, labels map[string]string) error {
	return &cloudops.ErrNotSupported{
		Operation: "RemoveTags",
	}
}

// Tags will list the existing labels/tags on the given volume
func (ops *vsphereOps) Tags(volumeID string) (map[string]string, error) {
	return nil, &cloudops.ErrNotSupported{
		Operation: "Tags",
	}
}

func (ops *vsphereOps) CreateInstance(opts interface{}) (*cloudops.InstanceInfo, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createOpts, ok := opts.(*VirtualMachineCreateOpts)
	if !ok {
		return nil, &cloudops.ErrInvalidArgument{
			Operation: "CreateInstance",
			Reason:    fmt.Sprintf("Invalid create options for VM create: %v", opts),
		}
	}

	devices, err := ops.addStorageToVM(nil, "scsi")
	if err != nil {
		return nil, err
	}

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}

	spec := createOpts.Spec
	spec.DeviceChange = deviceChange

	finder, dc, err := ops.getFinderAndDC(ctx, createOpts.Datacenter)
	if err != nil {
		return nil, err
	}

	var (
		host         *object.HostSystem
		resourcePool *object.ResourcePool
		datastore    *object.Datastore
	)

	if len(createOpts.Host) > 0 {
		host, err = finder.HostSystem(ctx, createOpts.Host)
		if err != nil {
			return nil, err
		}

		resourcePool, err = host.ResourcePool(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		resourcePool, err = finder.ResourcePoolOrDefault(ctx, createOpts.ResourcePool)
		if err != nil {
			return nil, err
		}
	}

	// If storage pod is specified, collect placement recommendations
	if len(createOpts.StoragePod) > 0 {
		storagePod, err := finder.DatastoreCluster(ctx, createOpts.StoragePod)
		if err != nil {
			return nil, err
		}

		datastore, err = recommendDatastore(ctx, ops.conn.GoVmomiClient.Client, resourcePool, storagePod, spec)
		if err != nil {
			return nil, err
		}
	} else {
		datastore, err = finder.DatastoreOrDefault(ctx, createOpts.Datastore)
		if err != nil {
			return nil, err
		}

	}

	vmxPath := fmt.Sprintf("%s/%s.vmx", spec.Name, spec.Name)
	_, err = datastore.Stat(ctx, vmxPath)
	if err == nil {
		dsPath := datastore.Path(vmxPath)
		return nil, fmt.Errorf("File %s already exists", dsPath)
	}

	folder, err := ops.getFolder(ctx, finder, createOpts.Folder)
	if err != nil {
		return nil, err
	}

	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastore.Name()),
	}

	task, err := folder.CreateVM(ctx, *spec, resourcePool, host)
	if err != nil {
		return nil, err
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, err
	}

	vm := object.NewVirtualMachine(ops.conn.GoVmomiClient.Client, info.Result.(types.ManagedObjectReference))
	if createOpts.PowerOn {
		task, err := vm.PowerOn(ctx)
		if err != nil {
			return nil, err
		}

		_, err = task.WaitForResult(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	inst := &cloudops.InstanceInfo{
		CloudResourceInfo: cloudops.CloudResourceInfo{
			Name: createOpts.Spec.Name,
			ID:   vm.Reference().Value,
			Zone: dc.Name(),
		},
	}
	return inst, nil
}

func (ops *vsphereOps) InspectInstance(vmNameOrUUID string) (*cloudops.InstanceInfo, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vm, err := ops.getVMByNameOrUUID(ctx, vmNameOrUUID)
	if err != nil {
		return nil, err
	}

	vmUUID, err := ops.getUUIDForVM(ctx, vm)
	if err != nil {
		return nil, err
	}

	vmName, err := ops.getNameForVM(ctx, vm)
	if err != nil {
		return nil, err
	}

	vmZone, err := ops.getZoneForVM(ctx, vm)
	if err != nil {
		return nil, err
	}

	return &cloudops.InstanceInfo{
		CloudResourceInfo: cloudops.CloudResourceInfo{
			Name: vmName,
			ID:   vmUUID,
			Zone: vmZone,
		},
	}, nil
}

func (ops *vsphereOps) DeleteInstance(
	vmNameOrUUID string,
	// TODO check if we can use the zone for vmware. VM names are unique
	// across the vcenter. So mostly we don't need this.
	zone string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vm, err := ops.getVMByNameOrUUID(ctx, vmNameOrUUID)
	if err != nil {
		return err
	}

	task, err := vm.PowerOff(ctx)
	if err != nil {
		return err
	}

	// Ignore error since the VM may already been in powered off state.
	// vm.Destroy will fail if the VM is still powered on.
	_ = task.Wait(ctx)

	task, err = vm.Destroy(ctx)
	if err != nil {
		return err
	}

	err = task.Wait(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (ops *vsphereOps) getVMByNameOrUUID(
	ctx context.Context,
	vmNameOrUUID string) (*object.VirtualMachine, error) {
	// first try looking up by name
	vm, err := ops.getVMByName(ctx, vmNameOrUUID)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, err
		}

		// lookup by uuid
		vmObj, err := GetVMObject(ctx, ops.conn, vmNameOrUUID)
		if err != nil {
			return nil, err
		}

		return vmObj.VirtualMachine, nil
	}

	return vm, nil

}

func (ops *vsphereOps) getVMByName(ctx context.Context, name string) (*object.VirtualMachine, error) {
	finder, _, err := ops.getFinderAndDC(ctx, "")
	if err != nil {
		return nil, err
	}

	vms, err := finder.VirtualMachineList(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(vms) == 0 {
		err = fmt.Errorf("given name: %s matched no vms. Please provide"+
			" a valid vm name", name)
		return nil, err
	}

	if len(vms) > 1 {
		err = fmt.Errorf("given name: %s matches multiple vms. Please provide"+
			" a specific vm name", name)
		return nil, err
	}

	return vms[0], nil
}

func (ops *vsphereOps) ListInstances(opts *cloudops.ListInstancesOpts) (
	[]*cloudops.InstanceInfo, error) {
	resp := make([]*cloudops.InstanceInfo, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var datacenter string
	if opts != nil && opts.LabelSelector != nil {
		datacenter = opts.LabelSelector["datacenter"]
	}

	finder, dc, err := ops.getFinderAndDC(ctx, datacenter)
	if err != nil {
		return nil, err
	}

	client, err := ops.getClient(ctx)
	if err != nil {
		return nil, err
	}

	root := client.ServiceContent.RootFolder
	if len(datacenter) > 0 {
		root = dc.Reference()
	}

	// Create filter for ops.ListInstancesOpts
	filter := property.Filter{}
	if opts != nil {
		for k, v := range opts.LabelSelector {
			// skip certain filter options
			if k == "datacenter" {
				continue
			}
			filter[k] = v
		}
	}

	recurse := true
	m := view.NewManager(client.Client)
	kinds := []string{"VirtualMachine"}
	v, err := m.CreateContainerView(ctx, root, kinds, recurse)
	if err != nil {
		return nil, err
	}

	defer v.Destroy(ctx)

	objs, err := v.Find(ctx, kinds, filter)
	if err != nil {
		return nil, err
	}

	for _, o := range objs {
		e, err := finder.Element(ctx, o)
		if err != nil {
			return nil, err
		}

		vm, err := finder.VirtualMachine(ctx, e.Path)
		if err != nil {
			return nil, err
		}

		if opts != nil {
			if len(opts.NamePrefix) > 0 {
				if !strings.HasPrefix(vm.Name(), opts.NamePrefix) {
					continue
				}
			}
		}

		isValid, err := ops.isVMValid(ctx, vm)
		if err != nil {
			return nil, err
		}

		if !isValid {
			continue
		}

		vmUUID, err := ops.getUUIDForVM(ctx, vm)
		if err != nil {
			return nil, err
		}

		vmZone, err := ops.getZoneForVM(ctx, vm)
		if err != nil {
			return nil, err
		}

		resp = append(resp, &cloudops.InstanceInfo{
			CloudResourceInfo: cloudops.CloudResourceInfo{
				Name: vm.Name(),
				ID:   vmUUID,
				Zone: vmZone,
			},
		})
	}

	return resp, nil
}

func (ops *vsphereOps) getUUIDForVM(ctx context.Context, vm *object.VirtualMachine) (string, error) {
	var o mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"config.uuid"}, &o)
	if err != nil {
		return "", err
	}

	if o.Config != nil {
		return o.Config.Uuid, nil
	}

	return "", fmt.Errorf("failed to get UUID for vm: %v", vm)
}

func (ops *vsphereOps) isVMValid(ctx context.Context, vm *object.VirtualMachine) (bool, error) {
	var o mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"runtime.connectionState"}, &o)
	if err != nil {
		return false, err
	}

	return o.Runtime.ConnectionState != "invalid", nil
}

func (ops *vsphereOps) getNameForVM(ctx context.Context, vm *object.VirtualMachine) (string, error) {
	var o mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"config.name"}, &o)
	if err != nil {
		return "", err
	}

	return o.Config.Name, nil
}

// We treat a VM's zone as the vSphere cluster in which the vm resizes.
// If there is no cluster, then it's the datacenter
func (ops *vsphereOps) getZoneForVM(
	ctx context.Context,
	vm *object.VirtualMachine) (string, error) {

	host, err := vm.HostSystem(ctx)
	if err != nil {
		return "", err
	}

	var o mo.HostSystem
	err = host.Properties(ctx, host.Reference(), []string{"parent"}, &o)
	if err != nil {
		return "", err
	}

	var zone string
	switch o.Parent.Type {
	case "ClusterComputeResource":
		ccr := object.NewClusterComputeResource(ops.conn.GoVmomiClient.Client, *o.Parent)
		zone, err = ccr.ObjectName(ctx)
		if err != nil {
			return "", err
		}
		return zone, nil
	case "ComputeResource":
		// zone is the datacenter ( o -> parent -> parent -> parent )
		cr := object.NewComputeResource(ops.conn.GoVmomiClient.Client, *o.Parent)
		var o mo.ComputeResource
		err = cr.Properties(ctx, cr.Reference(), []string{"parent"}, &o)
		if err != nil {
			return "", err
		}

		if o.Parent.Type == "Folder" {
			folder := object.NewFolder(ops.conn.GoVmomiClient.Client, *o.Parent)
			var o mo.Folder
			err = folder.Properties(ctx, folder.Reference(), []string{"parent"}, &o)
			if err != nil {
				return "", err
			}

			if o.Parent.Type == "Datacenter" {
				dc := object.NewDatacenter(ops.conn.GoVmomiClient.Client, *o.Parent)
				zone, err = dc.ObjectName(ctx)
				if err != nil {
					return "", err
				}
				return zone, nil
			}
		}
	}

	return "", fmt.Errorf("failed to get zone for vm: %s", vm.Name())
}

// addStorageToVM Current this just adds disk controllers. In future, add support
// for iso, vmdks here
func (ops *vsphereOps) addStorageToVM(
	devices object.VirtualDeviceList,
	controller string) (object.VirtualDeviceList, error) {
	if controller != "ide" {
		if controller == "nvme" {
			nvme, err := devices.CreateNVMEController()
			if err != nil {
				return nil, err
			}

			devices = append(devices, nvme)
			controller = devices.Name(nvme)
		} else {
			scsi, err := devices.CreateSCSIController(controller)
			if err != nil {
				return nil, err
			}

			devices = append(devices, scsi)
			controller = devices.Name(scsi)
		}
	}

	// If controller is specified to be IDE, add IDE controller.
	if controller == "ide" {
		ide, err := devices.CreateIDEController()
		if err != nil {
			return nil, err
		}

		devices = append(devices, ide)
	}

	return devices, nil
}

// GetVMObject fetches the VirtualMachine object corresponding to the given virtual machine uuid
func GetVMObject(ctx context.Context, conn *vclib.VSphereConnection, vmUUID string) (*vclib.VirtualMachine, error) {
	// TODO change impl below using multiple goroutines and sync.WaitGroup to make it faster
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := conn.Connect(ctx); err != nil {
		return nil, err
	}

	if len(vmUUID) == 0 {
		return nil, fmt.Errorf("virtual machine uuid is required")
	}

	datacenterObjs, err := vclib.GetAllDatacenter(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Lookup in each vsphere datacenter for this virtual machine
	for _, dc := range datacenterObjs {
		vm, err := dc.GetVMByUUID(ctx, vmUUID)
		if err != nil {
			if err != vclib.ErrNoVMFound {
				logrus.Warnf("failed to find vm with uuid: %s in datacenter: %s due to err: %v", vmUUID, dc.Name(), err)
				// don't let one bad egg fail entire search. keep looking.
			} else {
				logrus.Debugf("did not find vm with uuid: %s in datacenter: %s", vmUUID, dc.Name())
			}
			continue
		}

		if vm != nil {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("failed to find vm with uuid: %s in any datacenter for vc: %s", vmUUID, conn.Hostname)
}

func (ops *vsphereOps) renewVM(ctx context.Context, vm *vclib.VirtualMachine) (*vclib.VirtualMachine, error) {
	client, err := ops.getClient(ctx)
	if err != nil {
		return nil, err
	}

	vmObj := vm.RenewVM(client)
	return &vmObj, nil
}

func (ops *vsphereOps) getClient(ctx context.Context) (*govmomi.Client, error) {
	var client *govmomi.Client
	err := ops.conn.Connect(ctx)
	if err != nil {
		client, err = ops.conn.NewClient(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		client = ops.conn.GoVmomiClient
	}

	return client, nil
}

// getDatastoreToUseInStoragePod asks the storage resource manager to recommend a datastore
// in the given storage pod (datastore cluster) for the required disk spec
func (ops *vsphereOps) getDatastoreToUseInStoragePod(
	ctx context.Context, vmObj *vclib.VirtualMachine,
	volumeOptions *vclib.VolumeOptions, storagePod *object.StoragePod) (string, error) {
	logrus.Infof("Using storage pod: %s", storagePod.Name())

	// devices is a list of devices in the virtual machine (disks and disk controllers) that
	// will be part of the request spec to storage resource manager
	var devices object.VirtualDeviceList
	scsi, err := devices.CreateSCSIController("scsi")
	if err != nil {
		return "", err
	}

	devices = append(devices, scsi)

	controller, err := devices.FindDiskController("scsi")
	if err != nil {
		return "", err
	}

	disk := &types.VirtualDisk{
		VirtualDevice: types.VirtualDevice{
			Key: devices.NewKey(),
			Backing: &types.VirtualDiskFlatVer2BackingInfo{
				DiskMode:        string(types.VirtualDiskModePersistent),
				ThinProvisioned: types.NewBool(true),
			},
		},
		CapacityInKB: int64(volumeOptions.CapacityKB),
	}

	devices = append(devices, disk)
	devices.AssignController(disk, controller)
	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return "", err
	}

	vmName, err := vmObj.ObjectName(ctx)
	if err != nil {
		return "", err
	}

	spec := &types.VirtualMachineConfigSpec{
		Name: vmName,
	}

	spec.DeviceChange = deviceChange
	resourcePool, err := vmObj.ResourcePool(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get vm resource pool due to: %v", err)
	}

	recommendedDatastore, err := recommendDatastore(ctx, vmObj.Client(), resourcePool, storagePod, spec)
	if err != nil {
		return "", err
	}

	return recommendedDatastore.Name(), nil
}

// recommendedDatastore recommends a datastore to use for the given storage pod by
// quering the storage resource manager
// logic borrowwed from recommendDatastore() at https://github.com/vmware/govmomi/blob/master/govc/vm/create.go#L455
func recommendDatastore(
	ctx context.Context,
	client *vim25.Client,
	resourcePool *object.ResourcePool,
	storagePod *object.StoragePod,
	spec *types.VirtualMachineConfigSpec) (*object.Datastore, error) {
	sp := storagePod.Reference()

	// Build pod selection spec from config spec
	podSelectionSpec := types.StorageDrsPodSelectionSpec{
		StoragePod: &sp,
	}

	for _, deviceConfigSpec := range spec.DeviceChange {
		s := deviceConfigSpec.GetVirtualDeviceConfigSpec()
		if s.Operation != types.VirtualDeviceConfigSpecOperationAdd {
			continue
		}

		if s.FileOperation != types.VirtualDeviceConfigSpecFileOperationCreate {
			continue
		}

		d, ok := s.Device.(*types.VirtualDisk)
		if !ok {
			continue
		}

		podConfigForPlacement := types.VmPodConfigForPlacement{
			StoragePod: sp,
			Disk: []types.PodDiskLocator{
				{
					DiskId:          d.Key,
					DiskBackingInfo: d.Backing,
				},
			},
		}

		podSelectionSpec.InitialVmConfig = append(podSelectionSpec.InitialVmConfig, podConfigForPlacement)
	}

	if resourcePool == nil {
		return nil, fmt.Errorf("failed to get vm resource pool")
	}

	resourcePoolRef := resourcePool.Reference()

	sps := types.StoragePlacementSpec{
		Type:             string(types.StoragePlacementSpecPlacementTypeCreate),
		PodSelectionSpec: podSelectionSpec,
		ConfigSpec:       spec,
		ResourcePool:     &resourcePoolRef,
	}

	srm := object.NewStorageResourceManager(client)
	result, err := srm.RecommendDatastores(ctx, sps)
	if err != nil {
		logrus.Errorf("failed to get datastore recommendations due to: %v", err)
		return nil, err
	}

	// Use result to pin disks to recommended datastores
	recs := result.Recommendations
	if len(recs) == 0 {
		return nil, fmt.Errorf("no datastores recommendations")
	}

	ds := recs[0].Action[0].(*types.StoragePlacementAction).Destination

	var mds mo.Datastore
	err = property.DefaultCollector(client).RetrieveOne(ctx, ds, []string{"name"}, &mds)
	if err != nil {
		return nil, err
	}

	datastore := object.NewDatastore(client, ds)
	datastore.InventoryPath = mds.Name

	return datastore, nil
}

// IsStoragePod checks if the object with given name is a StoragePod (Datastore cluster)
func IsStoragePod(ctx context.Context, vmObj *vclib.VirtualMachine, name string) (bool, *object.StoragePod, error) {
	f := find.NewFinder(vmObj.Client(), true)
	f.SetDatacenter(vmObj.Datacenter.Datacenter)
	sp, err := f.DatastoreCluster(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil, nil
		}

		logrus.Errorf("got error: %v fetching datastore cluster: %s", err, name)
		return false, nil, err
	}

	if sp == nil {
		return false, nil, nil
	}

	return true, sp, nil
}

func (ops *vsphereOps) getFolder(
	ctx context.Context,
	finder *find.Finder,
	folderName string) (*object.Folder, error) {
	if finder == nil {
		return nil, fmt.Errorf("no finder provided to getFolder call")
	}

	folder, err := finder.FolderOrDefault(ctx, folderName)
	if err != nil {
		return nil, err
	}

	return folder, nil
}

func (ops *vsphereOps) getFinderAndDC(ctx context.Context, datacenter string) (
	*find.Finder, *object.Datacenter, error) {
	client, err := ops.getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	finder := find.NewFinder(client.Client, false)

	var dc *object.Datacenter
	if datacenter == "" {
		dc, err = finder.DefaultDatacenter(ctx)
	} else {
		if dc, err = finder.Datacenter(ctx, datacenter); err != nil {
			return nil, nil, err
		}
	}

	finder.SetDatacenter(dc)
	return finder, dc, nil
}
