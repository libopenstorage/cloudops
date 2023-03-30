package store

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/portworx/sched-ops/k8s/core/configmap"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	confgMapPrefix  = "px-cloud-drive-"
	cloudDriveEntry = "cloud-drive"
	waitDuration    = 2 * time.Second
	waitFactor      = 1.5
	waitSteps       = 5
)

// GetSanitizedK8sName will sanitize the name conforming to RFC 1123 standards so that it's a "qualified name" per k8s
func GetSanitizedK8sName(k8sName string) string {
	sanitizedString := ""
	if len(k8sName) > 0 {
		k8sName = strings.ToLower(k8sName)
		k8sName = strings.ReplaceAll(k8sName, " ", "-")
		if msgs := validation.IsDNS1123Subdomain(k8sName); len(msgs) > 0 {
			for _, z := range k8sName {
				// Names are expected to start and end with alphanumeric characters. Hence adding "a"
				if msgs := validation.IsDNS1123Subdomain("a" + string(z) + "a"); len(msgs) == 0 {
					sanitizedString += string(z)
				} else {
					sanitizedString += "."
				}
			}
			//After sanitizing, we need the first and last letter to be alphanumeric
			isAlpha := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
			first := float64(len(sanitizedString))
			last := float64(0)
			for i, s := range sanitizedString {
				if isAlpha(string(s)) {
					first = float64(math.Min(first, float64(i)))
					last = float64(math.Max(last, float64(i)))
				}
			}
			sanitizedString = sanitizedString[int(first) : int(last)+1]
			logrus.Infof("k8s name is not per RFC 1123 standard sanitized %s to %s", k8sName, sanitizedString)
		} else {
			sanitizedString = k8sName
		}
	}
	return sanitizedString
}

var (
	// total wait time: 16.25 seconds
	waitBackoff = wait.Backoff{
		Duration: waitDuration, // the base duration
		Factor:   waitFactor,   // Duration is multiplied by factor each iteration
		Steps:    waitSteps,    // Exit with error after this many steps
	}
	errorsToRetryOn = []error{rpctypes.ErrLeaderChanged}
)

type k8sStore struct {
	cm configmap.ConfigMap
}

// NewK8sStore returns a Store implementation which uses
// k8s configmaps to store data.
func NewK8sStore(clusterID string) (Store, configmap.ConfigMap, error) {
	ns := os.Getenv("PX_NAMESPACE")
	k8sStore, cm, err := newK8sStoreWithParams(
		configmap.GetName(confgMapPrefix, clusterID),
		configmap.DefaultK8sLockTimeout,
		configmap.DefaultK8sLockAttempts*time.Second,
		ns,
	)
	if err != nil {
		return nil, nil, err
	}
	return k8sStore, cm, nil
}

// newK8sStoreWithParams returns a Store implementation which uses
// k8s configmaps to store data. ConfigMap properties can be customized.
func newK8sStoreWithParams(
	name string,
	lockTryDuration time.Duration,
	lockTimeout time.Duration,
	nameSpace string,
) (Store, configmap.ConfigMap, error) {
	lockAttempts := uint((lockTryDuration / time.Second))
	cm, err := configmap.New(
		name,
		nil,
		lockTimeout,
		lockAttempts,
		0,
		0,
		nameSpace,
	)
	if err != nil {
		return nil, nil, err
	}
	return &k8sStore{cm}, cm, nil
}

func (k8s *k8sStore) Lock(owner string) (*Lock, error) {
	if err := k8s.cm.Lock(owner); err != nil {
		return nil, err
	}
	return &Lock{owner: owner}, nil
}

func (k8s *k8sStore) LockWithKey(owner, key string) (*Lock, error) {
	if err := k8s.cm.LockWithKey(owner, key); err != nil {
		return nil, err
	}
	return &Lock{Key: key, owner: owner, lockedWithKey: true}, nil
}

func (k8s *k8sStore) Unlock(storeLock *Lock) error {
	if storeLock.lockedWithKey {
		return k8s.cm.UnlockWithKey(storeLock.Key)
	}
	return k8s.cm.Unlock()
}

func (k8s *k8sStore) IsKeyLocked(key string) (bool, string, error) {
	return k8s.cm.IsKeyLocked(key)
}

func (k8s *k8sStore) CreateKey(key string, value []byte) error {
	sanitizedKey := GetSanitizedK8sName(key)
	err := k8s.cm.LockWithKey(string(value), sanitizedKey)
	if err != nil {
		logrus.Errorf("unable to lock with key %v", key)
		return err
	}
	defer func() {
		err := k8s.cm.UnlockWithKey(sanitizedKey)
		if err != nil {
			logrus.Warnf("unable to unlock with key %v", key)
		}
	}()

	data, err := k8s.cm.Get()
	if err != nil {
		return err
	}

	if _, ok := data[key]; ok {
		return &KeyExists{
			Key:     key,
			Message: "Use PutKey API",
		}
	}

	if data == nil {
		data = make(map[string]string)
	}
	data[key] = string(value)
	return k8s.patchWithRetries(data)
}

func (k8s *k8sStore) PutKey(key string, value []byte) error {
	data, err := k8s.cm.Get()
	if err != nil {
		return err
	}

	data[key] = string(value)
	return k8s.patchWithRetries(data)
}

func (k8s *k8sStore) GetKey(key string) ([]byte, error) {
	data, err := k8s.cm.Get()
	if err != nil {
		return nil, err
	}

	value, ok := data[key]
	if !ok {
		return nil, &KeyDoesNotExist{
			Key: key,
		}
	}

	return []byte(value), nil
}

func (k8s *k8sStore) DeleteKey(key string) error {
	sanitizedKey := GetSanitizedK8sName(key)
	// Let's use the sanitized key itself as the owner.
	err := k8s.cm.LockWithKey(sanitizedKey, sanitizedKey)
	if err != nil {
		logrus.Errorf("unable to lock with key %v", key)
		return err
	}
	defer func() {
		err := k8s.cm.UnlockWithKey(sanitizedKey)
		if err != nil {
			logrus.Infof("unable to unlock with key %v", key)
		}
	}()
	data, err := k8s.cm.Get()
	if err != nil {
		return err
	}

	if _, ok := data[key]; !ok {
		return nil
	}

	delete(data, key)
	return k8s.cm.Update(data)
}

func (k8s *k8sStore) EnumerateWithKeyPrefix(key string) ([]string, error) {
	data, err := k8s.cm.Get()
	if err != nil {
		return nil, err
	}

	returnKeys := make([]string, 0)
	for k := range data {
		if strings.HasPrefix(k, key) {
			returnKeys = append(returnKeys, k)
		}
	}

	return returnKeys, nil
}

func (k8s *k8sStore) patchWithRetries(data map[string]string) error {
	f := func() (bool, error) {
		err := k8s.cm.Patch(data)

		for _, retryErr := range errorsToRetryOn {
			if err == retryErr {
				logrus.Warnf("patch operation on config map failed with an error: %v, retrying", err)
				return false, nil // retry
			}
		}

		if err != nil {
			return false, err
		}

		return true, nil
	}
	if err := wait.ExponentialBackoff(waitBackoff, f); err != nil {
		return fmt.Errorf("failed to patch configmap data: %s, %w", data, err)
	}
	return nil
}
