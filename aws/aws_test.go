package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
	"github.com/pborman/uuid"
	"github.com/portworx/sched-ops/k8s/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	newDiskSizeInGB = 10
	newDiskPrefix   = "openstorage-test"
)

var diskName = fmt.Sprintf("%s-%s", newDiskPrefix, uuid.New())

func TestAll(t *testing.T) {
	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})

	if d, err := NewClient("", ""); err == nil {
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

func TestAllWithKubernetes(t *testing.T) {

	// Create a new fake clientset
	client := fake.NewSimpleClientset()
	schedClient := core.New(client)
	core.SetInstance(schedClient)

	k8sSecretName := "px-aws"
	k8sSecretNamespace := "portworx"

	// Test Case: Valid AWS credentials

	// Fetch the caller provided credentials from the env variables
	// and put them into a kubernetes secret.
	_, err := core.Instance().CreateSecret(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sSecretName,
			Namespace: k8sSecretNamespace,
		},
		Data: map[string][]byte{
			awsAccessKeyName:       []byte(os.Getenv(awsAccessKeyName)),
			awsSecretAccessKeyName: []byte(os.Getenv(awsSecretAccessKeyName)),
		},
	})
	require.NoError(t, err, "failed to create fake secret")

	// Unset the aws credentials from the environment variables
	// so that the static credentials from the k8s secret are used.
	os.Unsetenv(awsAccessKeyName)
	os.Unsetenv(awsSecretAccessKeyName)

	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})

	if d, err := NewClient(k8sSecretName, k8sSecretNamespace); err == nil {
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

	// Test Case: Missing aws access key
	_, err = core.Instance().UpdateSecret(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sSecretName,
			Namespace: k8sSecretNamespace,
		},
	})
	require.NoError(t, err, "failed to create fake secret")

	c, err := NewClient(k8sSecretName, k8sSecretNamespace)
	require.Contains(t, err.Error(), fmt.Sprintf("%v not found in k8s secret", awsAccessKeyName))
	require.Nil(t, c)

	// Test Case: Missing aws secret key
	_, err = core.Instance().UpdateSecret(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sSecretName,
			Namespace: k8sSecretNamespace,
		},
		Data: map[string][]byte{
			awsAccessKeyName: []byte("access-key"),
		},
	})
	require.NoError(t, err, "failed to create fake secret")

	c, err = NewClient(k8sSecretName, k8sSecretNamespace)
	require.Contains(t, err.Error(), fmt.Sprintf("%v not found in k8s secret", awsSecretAccessKeyName))
	require.Nil(t, c)

	// Test Case: Invalid aws credentials
	_, err = core.Instance().UpdateSecret(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sSecretName,
			Namespace: k8sSecretNamespace,
		},
		Data: map[string][]byte{
			awsAccessKeyName:       []byte("access-key"),
			awsSecretAccessKeyName: []byte("secret-key"),
		},
	})
	require.NoError(t, err, "failed to create fake secret")

	c, err = NewClient(k8sSecretName, k8sSecretNamespace)
	require.NoError(t, err)
	require.NotNil(t, c)

	vols, err := c.Enumerate(nil, nil, "")
	require.Error(t, err)
	require.Empty(t, vols)
}
