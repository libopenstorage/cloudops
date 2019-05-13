package gce

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/unsupported"
	"github.com/libopenstorage/openstorage/pkg/parser"
	"github.com/portworx/sched-ops/task"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var notFoundRegex = regexp.MustCompile(`.*notFound`)

const googleDiskPrefix = "/dev/disk/by-id/google-"

// StatusReady ready status
const StatusReady = "ready"

const (
	devicePathMaxRetryCount = 3
	devicePathRetryInterval = 2 * time.Second
	clusterNameKey          = "cluster-name"
	clusterLocationKey      = "cluster-location"
	kubeLabelsKey           = "kube-labels"
	nodePoolKey             = "cloud.google.com/gke-nodepool"
	instanceTemplateKey     = "instance-template"
)

type gceOps struct {
	cloudops.Compute
	inst             *instance
	computeService   *compute.Service
	containerService *container.Service
	mutex            sync.Mutex
}

// instance stores the metadata of the running GCE instance
type instance struct {
	name     string
	hostname string
	zone     string
	region   string
	project  string
}

// IsDevMode checks if the pkg is invoked in developer mode where GCE credentials
// are set as env variables
func IsDevMode() bool {
	var i = new(instance)
	err := gceInfoFromEnv(i)
	return err == nil
}

// NewClient creates a new GCE operations client
func NewClient() (cloudops.Ops, error) {
	var i = new(instance)
	var err error
	if metadata.OnGCE() {
		err = gceInfo(i)
	} else if ok := IsDevMode(); ok {
		err = gceInfoFromEnv(i)
	} else {
		return nil, fmt.Errorf("instance is not running on GCE")
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching instance info. Err: %v", err)
	}

	ctx := context.Background()
	computeService, err := compute.NewService(ctx, option.WithScopes(compute.ComputeScope))
	if err != nil {
		return nil, fmt.Errorf("unable to create Compute service: %v", err)
	}

	containerService, err := container.NewService(ctx, option.WithScopes(compute.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("unable to create Container service: %v", err)
	}

	return &gceOps{
		Compute:          unsupported.NewUnsupportedCompute(),
		inst:             i,
		computeService:   computeService,
		containerService: containerService,
	}, nil
}

func (s *gceOps) Name() string { return "gce" }

func (s *gceOps) InstanceID() string { return s.inst.name }

func (s *gceOps) InspectInstance(instanceID string) (*cloudops.InstanceInfo, error) {
	inst, err := s.computeService.Instances.Get(s.inst.project, s.inst.zone, instanceID).Do()
	if err != nil {
		return nil, err
	}

	instInfo := &cloudops.InstanceInfo{
		CloudResourceInfo: cloudops.CloudResourceInfo{
			Name:   inst.Name,
			ID:     fmt.Sprintf("%d", inst.Id),
			Zone:   inst.Zone,
			Region: s.inst.region,
			Labels: inst.Labels,
		},
	}
	return instInfo, nil
}

func (s *gceOps) InspectInstanceGroupForInstance(instanceID string) (*cloudops.InstanceGroupInfo, error) {
	inst, err := s.computeService.Instances.Get(s.inst.project, s.inst.zone, instanceID).Do()
	if err != nil {
		return nil, err
	}

	meta := inst.Metadata
	if meta == nil {
		return nil, fmt.Errorf("instance doesn't have metadata set")
	}

	var (
		gkeClusterName   string
		instanceTemplate string
		clusterLocation  string
		kubeLabels       map[string]string
	)

	for _, item := range meta.Items {
		if item == nil {
			continue
		}

		if item.Key == clusterNameKey {
			if item.Value == nil {
				return nil, fmt.Errorf("instance has %s key in metadata but has invalid value", clusterNameKey)
			}

			gkeClusterName = *item.Value
		}

		if item.Key == instanceTemplateKey {
			if item.Value == nil {
				return nil, fmt.Errorf("instance has %s key in metadata but has invalid value", instanceTemplateKey)
			}

			instanceTemplate = *item.Value
		}

		if item.Key == clusterLocationKey {
			if item.Value == nil {
				return nil, fmt.Errorf("instance has %s key in metadata but has invalid value", clusterLocationKey)
			}

			clusterLocation = *item.Value
		}

		if item.Key == kubeLabelsKey {
			if item.Value == nil {
				return nil, fmt.Errorf("instance has %s key in metadata but has invalid value", kubeLabelsKey)
			}

			kubeLabels, err = parser.LabelsFromString(*item.Value)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(gkeClusterName) == 0 ||
		len(instanceTemplate) == 0 ||
		len(clusterLocation) == 0 ||
		len(kubeLabels) == 0 {
		return nil, &cloudops.ErrNotSupported{
			Operation: "InspectInstanceGroupForInstance",
			Reason:    "API is currently only supported on the GKE platform",
		}
	}

	for labelKey, labelValue := range kubeLabels {
		if labelKey == nodePoolKey {
			nodePoolPath := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s",
				s.inst.project, clusterLocation, gkeClusterName, labelValue)
			nodePool, err := s.containerService.Projects.Locations.Clusters.NodePools.Get(nodePoolPath).Do()
			if err != nil {
				logrus.Errorf("failed to get node pool")
				return nil, err
			}

			zones := make([]string, 0)
			for _, igURL := range nodePool.InstanceGroupUrls {
				// e.g https://www.googleapis.com/compute/v1/projects/portworx-eng/zones/us-east1-b/instanceGroupManagers/gke-harsh-regional-asg-t-default-pool-a8750fe9-grp
				parts := strings.Split(igURL, "/")
				if len(parts) < 3 {
					return nil, fmt.Errorf("failed to parse zones for a node pool")
				}

				zones = append(zones, parts[len(parts)-3])
			}

			retval := &cloudops.InstanceGroupInfo{
				CloudResourceInfo: cloudops.CloudResourceInfo{
					Name:   nodePool.Name,
					Zone:   s.inst.zone,
					Region: s.inst.region,
				},
				Zones:              zones,
				AutoscalingEnabled: nodePool.Autoscaling.Enabled,
				Min:                &nodePool.Autoscaling.MinNodeCount,
				Max:                &nodePool.Autoscaling.MaxNodeCount,
			}

			if nodePool.Config != nil {
				retval.Labels = nodePool.Config.Labels
			}

			return retval, nil
		}
	}

	return nil, fmt.Errorf("instance doesn't belong to a GKE node pool")
}

func (s *gceOps) ApplyTags(
	diskName string,
	labels map[string]string) error {
	d, err := s.computeService.Disks.Get(s.inst.project, s.inst.zone, diskName).Do()
	if err != nil {
		return err
	}

	var currentLabels map[string]string
	if len(d.Labels) == 0 {
		currentLabels = make(map[string]string)
	} else {
		currentLabels = d.Labels
	}

	for k, v := range formatLabels(labels) {
		currentLabels[k] = v
	}

	rb := &compute.ZoneSetLabelsRequest{
		LabelFingerprint: d.LabelFingerprint,
		Labels:           currentLabels,
	}

	_, err = s.computeService.Disks.SetLabels(s.inst.project, s.inst.zone, d.Name, rb).Do()
	return err
}

func (s *gceOps) Attach(diskName string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var d *compute.Disk
	d, err := s.computeService.Disks.Get(s.inst.project, s.inst.zone, diskName).Do()
	if err != nil {
		return "", err
	}

	if len(d.Users) != 0 {
		return "", fmt.Errorf("disk %s is already in use by %s", diskName, d.Users)
	}

	diskURL := d.SelfLink
	rb := &compute.AttachedDisk{
		DeviceName: d.Name,
		Source:     diskURL,
	}

	_, err = s.computeService.Instances.AttachDisk(
		s.inst.project,
		s.inst.zone,
		s.inst.name,
		rb).Do()
	if err != nil {
		return "", err
	}

	devicePath, err := s.waitForAttach(d, time.Minute)
	if err != nil {
		return "", err
	}

	return devicePath, nil
}

func (s *gceOps) Create(
	template interface{},
	labels map[string]string,
) (interface{}, error) {
	v, ok := template.(*compute.Disk)
	if !ok {
		return nil, cloudops.NewStorageError(cloudops.ErrVolInval,
			"Invalid volume template given", "")
	}

	newDisk := &compute.Disk{
		Description:    "Disk created by openstorage",
		Labels:         formatLabels(labels),
		Name:           v.Name,
		SizeGb:         v.SizeGb,
		SourceImage:    v.SourceImage,
		SourceSnapshot: v.SourceSnapshot,
		Type:           v.Type,
		Zone:           path.Base(v.Zone),
	}

	resp, err := s.computeService.Disks.Insert(s.inst.project, newDisk.Zone, newDisk).Do()
	if err != nil {
		return nil, err
	}

	if err = s.checkDiskStatus(newDisk.Name, newDisk.Zone, StatusReady); err != nil {
		return nil, s.rollbackCreate(resp.Name, err)
	}

	d, err := s.computeService.Disks.Get(s.inst.project, newDisk.Zone, newDisk.Name).Do()
	if err != nil {
		return nil, err
	}

	return d, err
}

func (s *gceOps) DeleteFrom(id, _ string) error {
	return s.Delete(id)
}

func (s *gceOps) Delete(id string) error {
	ctx := context.Background()
	found := false
	req := s.computeService.Disks.AggregatedList(s.inst.project)
	if err := req.Pages(ctx, func(page *compute.DiskAggregatedList) error {
		for _, diskScopedList := range page.Items {
			for _, disk := range diskScopedList.Disks {
				if disk.Name == id {
					found = true
					_, err := s.computeService.Disks.Delete(s.inst.project, path.Base(disk.Zone), id).Do()
					return err
				}
			}
		}
		return nil
	}); err != nil {
		logrus.Errorf("failed to list disks: %v", err)
		return err
	}

	if !found {
		return fmt.Errorf("failed to delete disk: %s as it wasn't found", id)
	}

	return nil
}

func (s *gceOps) Detach(devicePath string) error {
	return s.detachInternal(devicePath, s.inst.name)
}

func (s *gceOps) DetachFrom(devicePath, instanceName string) error {
	return s.detachInternal(devicePath, instanceName)
}

func (s *gceOps) detachInternal(devicePath, instanceName string) error {
	_, err := s.computeService.Instances.DetachDisk(
		s.inst.project,
		s.inst.zone,
		instanceName,
		devicePath).Do()
	if err != nil {
		return err
	}

	var d *compute.Disk
	d, err = s.computeService.Disks.Get(s.inst.project, s.inst.zone, devicePath).Do()
	if err != nil {
		return err
	}

	err = s.waitForDetach(d.SelfLink, time.Minute)
	if err != nil {
		return err
	}

	return err
}

func (s *gceOps) DeviceMappings() (map[string]string, error) {
	instance, err := s.describeinstance()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, d := range instance.Disks {
		if d.Boot {
			continue
		}

		pathByID := fmt.Sprintf("%s%s", googleDiskPrefix, d.DeviceName)
		devPath, err := s.diskIDToBlockDevPath(pathByID)
		if err != nil {
			return nil, cloudops.NewStorageError(
				cloudops.ErrInvalidDevicePath,
				fmt.Sprintf("unable to find block dev path for %s. %v", pathByID, err),
				s.inst.name)
		}
		m[devPath] = path.Base(d.Source)
	}

	return m, nil
}

func (s *gceOps) DevicePath(diskName string) (string, error) {
	d, err := s.computeService.Disks.Get(s.inst.project, s.inst.zone, diskName).Do()
	if gerr, ok := err.(*googleapi.Error); ok &&
		gerr.Code == http.StatusNotFound {
		return "", cloudops.NewStorageError(
			cloudops.ErrVolNotFound,
			fmt.Sprintf("Disk: %s not found in zone %s", diskName, s.inst.zone),
			s.inst.name)
	} else if err != nil {
		return "", err
	}

	if len(d.Users) == 0 {
		err = cloudops.NewStorageError(cloudops.ErrVolDetached,
			fmt.Sprintf("Disk: %s is detached", d.Name), s.inst.name)
		return "", err
	}

	var inst *compute.Instance
	inst, err = s.describeinstance()
	if err != nil {
		return "", err
	}

	for _, instDisk := range inst.Disks {
		if instDisk.Source == d.SelfLink {
			pathByID := fmt.Sprintf("%s%s", googleDiskPrefix, instDisk.DeviceName)
			devPath, err := s.diskIDToBlockDevPathWithRetry(pathByID)
			if err == nil {
				return devPath, nil
			}
			return "", cloudops.NewStorageError(
				cloudops.ErrInvalidDevicePath,
				fmt.Sprintf("unable to find block dev path for %s. %v", devPath, err),
				s.inst.name)
		}
	}

	return "", cloudops.NewStorageError(
		cloudops.ErrVolAttachedOnRemoteNode,
		fmt.Sprintf("disk %s is not attached on: %s (Attached on: %v)",
			d.Name, s.inst.name, d.Users),
		s.inst.name)
}

func (s *gceOps) Enumerate(
	volumeIds []*string,
	labels map[string]string,
	setIdentifier string,
) (map[string][]interface{}, error) {
	sets := make(map[string][]interface{})

	allDisks, err := s.getDisksFromAllZones(formatLabels(labels))
	if err != nil {
		return nil, err
	}

	for _, disk := range allDisks {
		if len(setIdentifier) == 0 {
			cloudops.AddElementToMap(sets, disk, cloudops.SetIdentifierNone)
		} else {
			found := false
			for key := range disk.Labels {
				if key == setIdentifier {
					cloudops.AddElementToMap(sets, disk, key)
					found = true
					break
				}
			}

			if !found {
				cloudops.AddElementToMap(sets, disk, cloudops.SetIdentifierNone)
			}
		}
	}

	return sets, nil
}

func (s *gceOps) FreeDevices(
	blockDeviceMappings []interface{},
	rootDeviceName string,
) ([]string, error) {
	return nil, fmt.Errorf("function not implemented")
}

func (s *gceOps) GetDeviceID(disk interface{}) (string, error) {
	if d, ok := disk.(*compute.Disk); ok {
		return d.Name, nil
	} else if d, ok := disk.(*compute.Snapshot); ok {
		return d.Name, nil
	} else {
		return "", fmt.Errorf("invalid type: %v given to GetDeviceID", disk)
	}
}

func (s *gceOps) Inspect(diskNames []*string) ([]interface{}, error) {
	allDisks, err := s.getDisksFromAllZones(nil)
	if err != nil {
		return nil, err
	}

	var disks []interface{}
	for _, id := range diskNames {
		if d, ok := allDisks[*id]; ok {
			disks = append(disks, d)
		} else {
			return nil, fmt.Errorf("disk %s not found", *id)
		}
	}

	return disks, nil
}

func (s *gceOps) RemoveTags(
	diskName string,
	labels map[string]string,
) error {
	d, err := s.computeService.Disks.Get(s.inst.project, s.inst.zone, diskName).Do()
	if err != nil {
		return err
	}

	if len(d.Labels) != 0 {
		currentLabels := d.Labels
		for k := range formatLabels(labels) {
			delete(currentLabels, k)
		}

		rb := &compute.ZoneSetLabelsRequest{
			LabelFingerprint: d.LabelFingerprint,
			Labels:           currentLabels,
		}

		_, err = s.computeService.Disks.SetLabels(s.inst.project, s.inst.zone, d.Name, rb).Do()
	}

	return err
}

func (s *gceOps) Snapshot(
	disk string,
	readonly bool,
) (interface{}, error) {
	rb := &compute.Snapshot{
		Name: fmt.Sprintf("snap-%d%02d%02d", time.Now().Year(), time.Now().Month(), time.Now().Day()),
	}

	_, err := s.computeService.Disks.CreateSnapshot(s.inst.project, s.inst.zone, disk, rb).Do()
	if err != nil {
		return nil, err
	}

	if err = s.checkSnapStatus(rb.Name, StatusReady); err != nil {
		return nil, err
	}

	snap, err := s.computeService.Snapshots.Get(s.inst.project, rb.Name).Do()
	if err != nil {
		return nil, err
	}

	return snap, err
}

func (s *gceOps) SnapshotDelete(snapID string) error {
	_, err := s.computeService.Snapshots.Delete(s.inst.project, snapID).Do()
	return err
}

func (s *gceOps) Tags(diskName string) (map[string]string, error) {
	d, err := s.computeService.Disks.Get(s.inst.project, s.inst.zone, diskName).Do()
	if err != nil {
		return nil, err
	}

	return d.Labels, nil
}

func (s *gceOps) available(v *compute.Disk) bool {
	return strings.ToLower(v.Status) == StatusReady
}

func (s *gceOps) checkDiskStatus(id string, zone string, desired string) error {
	_, err := task.DoRetryWithTimeout(
		func() (interface{}, bool, error) {
			d, err := s.computeService.Disks.Get(s.inst.project, zone, id).Do()
			if err != nil {
				return nil, true, err
			}

			actual := strings.ToLower(d.Status)
			if len(actual) == 0 {
				return nil, true, fmt.Errorf("nil volume state for %v", id)
			}

			if actual != desired {
				return nil, true,
					fmt.Errorf("invalid status: %s for disk: %s. expected: %s",
						actual, id, desired)
			}

			return nil, false, nil
		},
		cloudops.ProviderOpsTimeout,
		cloudops.ProviderOpsRetryInterval)

	return err
}

func (s *gceOps) checkSnapStatus(id string, desired string) error {
	_, err := task.DoRetryWithTimeout(
		func() (interface{}, bool, error) {
			snap, err := s.computeService.Snapshots.Get(s.inst.project, id).Do()
			if err != nil {
				return nil, true, err
			}

			actual := strings.ToLower(snap.Status)
			if len(actual) == 0 {
				return nil, true, fmt.Errorf("nil snapshot state for %v", id)
			}

			if actual != desired {
				return nil, true,
					fmt.Errorf("invalid status: %s for snapshot: %s. expected: %s",
						actual, id, desired)
			}

			return nil, false, nil
		},
		cloudops.ProviderOpsTimeout,
		cloudops.ProviderOpsRetryInterval)

	return err
}

// Describe current instance.
func (s *gceOps) Describe() (interface{}, error) {
	return s.describeinstance()
}

func (s *gceOps) describeinstance() (*compute.Instance, error) {
	return s.computeService.Instances.Get(s.inst.project, s.inst.zone, s.inst.name).Do()
}

// gceInfo fetches the GCE instance metadata from the metadata server
func gceInfo(inst *instance) error {
	var err error
	inst.zone, err = metadata.Zone()
	if err != nil {
		return err
	}

	inst.region = inst.zone[:len(inst.zone)-2]

	inst.name, err = metadata.InstanceName()
	if err != nil {
		return err
	}

	inst.hostname, err = metadata.Hostname()
	if err != nil {
		return err
	}

	inst.project, err = metadata.ProjectID()
	if err != nil {
		return err
	}

	return nil
}

func gceInfoFromEnv(inst *instance) error {
	var err error
	inst.name, err = cloudops.GetEnvValueStrict("GCE_INSTANCE_NAME")
	if err != nil {
		return err
	}

	inst.zone, err = cloudops.GetEnvValueStrict("GCE_INSTANCE_ZONE")
	if err != nil {
		return err
	}

	inst.region = inst.zone[:len(inst.zone)-2]

	inst.project, err = cloudops.GetEnvValueStrict("GCE_INSTANCE_PROJECT")
	if err != nil {
		return err
	}

	return nil
}

func (s *gceOps) rollbackCreate(id string, createErr error) error {
	logrus.Warnf("Rollback create volume %v, Error %v", id, createErr)
	err := s.Delete(id)
	if err != nil {
		logrus.Warnf("Rollback failed volume %v, Error %v", id, err)
	}
	return createErr
}

// waitForDetach checks if given disk is detached from the local instance
func (s *gceOps) waitForDetach(
	diskURL string,
	timeout time.Duration,
) error {

	_, err := task.DoRetryWithTimeout(
		func() (interface{}, bool, error) {
			inst, err := s.describeinstance()
			if err != nil {
				return nil, true, err
			}

			for _, d := range inst.Disks {
				if d.Source == diskURL {
					return nil, true,
						fmt.Errorf("disk: %s is still attached to instance: %s",
							diskURL, s.inst.name)
				}
			}

			return nil, false, nil

		},
		cloudops.ProviderOpsTimeout,
		cloudops.ProviderOpsRetryInterval)

	return err
}

// waitForAttach checks if given disk is attached to the local instance
func (s *gceOps) waitForAttach(
	disk *compute.Disk,
	timeout time.Duration,
) (string, error) {
	devicePath, err := task.DoRetryWithTimeout(
		func() (interface{}, bool, error) {
			devicePath, err := s.DevicePath(disk.Name)
			if se, ok := err.(*cloudops.StorageError); ok &&
				se.Code == cloudops.ErrVolAttachedOnRemoteNode {
				return "", false, err
			} else if err != nil {
				return "", true, err
			}

			return devicePath, false, nil
		},
		cloudops.ProviderOpsTimeout,
		cloudops.ProviderOpsRetryInterval)
	if err != nil {
		return "", err
	}

	return devicePath.(string), nil
}

// generateListFilterFromLabels create a filter string based off --filter documentation at
// https://cloud.google.com/sdk/gcloud/reference/compute/disks/list
func generateListFilterFromLabels(labels map[string]string) string {
	var filter string
	for k, v := range labels {
		filter = fmt.Sprintf("%s(labels.%s eq %s)", filter, k, v)
	}

	return filter
}

func (s *gceOps) getDisksFromAllZones(labels map[string]string) (map[string]*compute.Disk, error) {
	ctx := context.Background()
	response := make(map[string]*compute.Disk)
	var req *compute.DisksAggregatedListCall

	if len(labels) > 0 {
		filter := generateListFilterFromLabels(labels)
		req = s.computeService.Disks.AggregatedList(s.inst.project).Filter(filter)
	} else {
		req = s.computeService.Disks.AggregatedList(s.inst.project)
	}

	if err := req.Pages(ctx, func(page *compute.DiskAggregatedList) error {
		for _, diskScopedList := range page.Items {
			for _, disk := range diskScopedList.Disks {
				response[disk.Name] = disk
			}
		}

		return nil
	}); err != nil {
		logrus.Errorf("failed to list disks: %v", err)
		return nil, err
	}

	return response, nil
}

func (s *gceOps) diskIDToBlockDevPathWithRetry(devPath string) (string, error) {
	var (
		retryCount int
		path       string
		err        error
	)

	for {
		if path, err = s.diskIDToBlockDevPath(devPath); err == nil {
			return path, nil
		}
		logrus.Warnf(err.Error())
		retryCount++
		if retryCount >= devicePathMaxRetryCount {
			break
		}
		time.Sleep(devicePathRetryInterval)
	}
	return "", err
}

func (s *gceOps) diskIDToBlockDevPath(devPath string) (string, error) {
	// check if path is a sym link. If yes, return pointee
	fi, err := os.Lstat(devPath)
	if err != nil {
		return "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		output, err := filepath.EvalSymlinks(devPath)
		if err != nil {
			return "", fmt.Errorf("failed to read symlink due to: %v", err)
		}

		devPath = strings.TrimSpace(string(output))
	} else {
		return "", fmt.Errorf("%s was expected to be a symlink to actual "+
			"device path", devPath)
	}

	return devPath, nil
}

func formatLabels(labels map[string]string) map[string]string {
	newLabels := make(map[string]string)
	for k, v := range labels {
		newLabels[strings.ToLower(k)] = strings.ToLower(v)
	}
	return newLabels
}
