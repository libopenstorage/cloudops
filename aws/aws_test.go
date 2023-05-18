package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	newDiskSizeInGB = 10
	newDiskPrefix   = "openstorage-test"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func TestAll(t *testing.T) {
	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})

	if d, err := NewClient(); err == nil {
		volType := opsworks.VolumeTypeGp2
		volSize := int64(newDiskSizeInGB)
		zone, _ := cloudops.GetEnvValueStrict("AWS_ZONE")
		ebsVol := &ec2.Volume{
			AvailabilityZone: &zone,
			VolumeType:       &volType,
			Size:             &volSize,
		}
		drivers[d.Name()] = d
		diskTemplates[d.Name()] = map[string]interface{}{
			diskName: ebsVol,
		}
	} else {
		t.Skipf("skipping AWS tests as environment is not set...\n")
	}

	test.RunTest(drivers, diskTemplates, sizeCheck, t)
}

func sizeCheck(template interface{}, targetSize uint64) bool {
	disk, ok := template.(*ec2.Volume)
	if !ok {
		return false
	}
	if disk.Size == nil {
		return false
	}
	return targetSize == uint64(*disk.Size)
}

func TestAwsGetPrefixFromRootDeviceName(t *testing.T) {
	a := &awsOps{}
	tests := []struct {
		deviceName     string
		expectedPrefix string
		expectError    bool
	}{
		{
			deviceName:     "/dev/sdb",
			expectedPrefix: "/dev/sd",
			expectError:    false,
		},
		{
			deviceName:     "/dev/xvdd",
			expectedPrefix: "/dev/xvd",
			expectError:    false,
		},
		{
			deviceName:     "/dev/xvdda",
			expectedPrefix: "/dev/xvd",
			expectError:    false,
		},
		{
			deviceName:     "/dev/dda",
			expectedPrefix: "",
			expectError:    true,
		},
		{
			deviceName:     "/dev/hdf",
			expectedPrefix: "/dev/hd",
			expectError:    false,
		},
		{
			deviceName:     "/dev/sys/dev/asdfasdfasdf",
			expectedPrefix: "",
			expectError:    true,
		},
		{
			deviceName:     "",
			expectedPrefix: "",
			expectError:    true,
		},
	}

	for _, test := range tests {
		prefix, err := a.getPrefixFromRootDeviceName(test.deviceName)
		assert.Equal(t, err != nil, test.expectError)
		assert.Equal(t, test.expectedPrefix, prefix)
	}
}

type mockEC2Client struct {
	ec2iface.EC2API
	Vol *ec2.Volume
}

func (m mockEC2Client) CreateVolume(*ec2.CreateVolumeInput) (*ec2.Volume, error) {
	return m.Vol, nil
}

func TestAwsCreate(t *testing.T) {
	cases := []struct {
		name           string
		volumeTemplate interface{}
		volResult      interface{}
		expectedErr    error
	}{
		{
			"illegal volume template from input is handled",
			"I am anything but a valid ec2 volume template",
			nil,
			cloudops.NewStorageError(cloudops.ErrVolInval,
				"Invalid volume template given", ""),
		},
	}
	for _, c := range cases {
		expectedRes, _ := c.volResult.(*ec2.Volume)
		s := &awsOps{
			ec2: &ec2Wrapper{
				Client: mockEC2Client{Vol: expectedRes},
			},
		}
		res, err := s.Create(c.volumeTemplate, nil, nil)
        assert.Equal(t, err.Error(), c.expectedErr.Error(), "%s failed, expected error: \n%v\n but got: \n%v", c.name, c.expectedErr, err)
        assert.Equal(t, res, c.volResult, "%s failed, expected result %s but got %s", c.name, c.volResult, res)
	}
}
