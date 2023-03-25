package store

import (
	"fmt"
	"github.com/portworx/kvdb"
	"time"
)

// PX specific scheduler constants
const (
	// Kubernetes identifies kubernetes as the scheduler
	Kubernetes = "kubernetes"
)

type StoreParams struct {
	Kv            kvdb.Kvdb
	InternalKvdb  bool
	SchedulerType string
}

// StoreLock identifies a lock taken over CloudDrive store
type StoreLock struct {
	// Key is the name on which the lock is acquired.
	// This is used by the callers for logging purpose. Hence public
	Key string
	// Name of the owner who acquired the lock
	owner string
	// true if this lock was acquired using LockWithKey() interface
	lockedWithKey bool
	// lock structure as returned from the KVDB interface
	internalLock interface{}
}

// StoreKeyDoesNotExist is error type when the key does not exist
type StoreKeyDoesNotExist struct {
	Key string
}

func (e *StoreKeyDoesNotExist) Error() string {
	return fmt.Sprintf("key %s does not exist", e.Key)
}

// StoreKeyExists is error type when the key already exist in store
type StoreKeyExists struct {
	// Key that exists
	Key string
	// Message is an optional message to the user
	Message string
}

func (e *StoreKeyExists) Error() string {
	errMsg := fmt.Sprintf("key %s already exists in store", e.Key)
	if len(e.Message) > 0 {
		errMsg += " " + e.Message
	}
	return errMsg
}

// Store provides a set of APIs to CloudDrive to store its metadata
// in a persistent store
type Store interface {
	// Lock locks the cloud drive store for a node to perform operations
	Lock(owner string) (*StoreLock, error)
	// Unlock unlocks the cloud drive store
	Unlock(storeLock *StoreLock) error
	// LockWithKey locks the cloud drive store with an arbitrary key
	LockWithKey(owner, key string) (*StoreLock, error)
	// IsKeyLocked checks if the specified key is currently locked
	IsKeyLocked(key string) (bool, string, error)
	// CreateKey creates the given key with the value
	CreateKey(key string, value []byte) error
	// PutKey updates the given key with the value
	PutKey(key string, value []byte) error
	// GetKey returns the value for the given key
	GetKey(key string) ([]byte, error)
	// DeleteKey deletes the given key
	DeleteKey(key string) error
	// EnumerateWithKeyPrefix enumerates all keys in the store that begin with the given key
	EnumerateWithKeyPrefix(key string) ([]string, error)
}

// GetStoreWithParams returns instance for Store
// kv: bootstrap kvdb
// schedulerType: node scheduler type e.g Kubernetes
// internalKvdb: If the cluster is configured to have internal kvdb
// name: Name for the store
// lockTryDuration: Total time to try acquiring the lock for
// lockHoldTimeout: Once a lock is acquired, if it's held beyond this time, there will be panic
func GetStoreWithParams(
	kv kvdb.Kvdb,
	schedulerType string,
	internalKvdb bool,
	name string,
	lockTryDuration time.Duration,
	lockHoldTimeout time.Duration,
) (Store, error) {
	var (
		s          Store
		err        error
		withParams bool
	)

	withParams = (lockHoldTimeout > 0) || (lockTryDuration > 0)
	if internalKvdb && schedulerType == Kubernetes {
		if len(name) == 0 {
			return nil, fmt.Errorf("name required to create k8s store")
		}
		if withParams {
			s, _, err = NewK8sStoreWithParams(name, lockTryDuration, lockHoldTimeout)
		} else {
			s, _, err = NewK8sStore(name)
		}
	} else if internalKvdb && kv == nil {
		return nil, fmt.Errorf("bootstrap kvdb cannot be empty")
	} else {
		// Two cases:
		// internal kvdb && kv is not nil
		// external kvdb
		if !internalKvdb {
			if kvdb.Instance() == nil {
				return nil, fmt.Errorf("kvdb is not initialized")
			} else {
				kv = kvdb.Instance()
			}
		}
		if withParams {
			s, err = NewKVStoreWithParams(kv, name, lockTryDuration, lockHoldTimeout)
		} else {
			s, err = NewKVStore(kv)
		}
	}
	return s, err
}
