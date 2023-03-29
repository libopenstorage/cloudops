package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/portworx/kvdb"
)

const (
	cloudDriveKey           = "clouddrive/"
	cloudDriveLockKey       = "_lock"
	defaultLockTryDuration  = 1 * time.Minute
	defaultLockHoldDuration = 3 * time.Minute
)

var (
	// ErrKvdbNotInitialized is returned when kvdb is not initialized
	ErrKvdbNotInitialized = errors.New("KVDB is not initialized")
)

type kvStore struct {
	k                kvdb.Kvdb
	lockPrefix       string
	lockTryDuration  time.Duration
	lockHoldDuration time.Duration
}

// NewKVStore returns a Store implementation which is a wrapper over
// kvdb.
func NewKVStore(kv kvdb.Kvdb) (Store, error) {
	return newKVStoreWithParams(kv, cloudDriveKey, 0, 0)
}

// NewKVStoreWithParams returns a Store implementation which is a wrapper over
// kvdb.
func newKVStoreWithParams(
	kv kvdb.Kvdb,
	name string,
	lockTryDuration time.Duration,
	lockHoldDuration time.Duration,
) (Store, error) {
	kstore := kvStore{}
	if kv == nil {
		return nil, ErrKvdbNotInitialized
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if lockTryDuration != 0 {
		kstore.lockTryDuration = lockTryDuration
	} else {
		kstore.lockTryDuration = kv.GetLockTryDuration()
	}
	if kstore.lockTryDuration == 0 {
		kstore.lockTryDuration = defaultLockTryDuration
	}

	if lockHoldDuration != 0 {
		kstore.lockHoldDuration = lockHoldDuration
	} else {
		kstore.lockHoldDuration = kv.GetLockHoldDuration()
	}
	if kstore.lockHoldDuration == 0 {
		kstore.lockHoldDuration = defaultLockHoldDuration
	}

	kstore.k = kv
	kstore.lockPrefix = name
	return &kstore, nil
}

func (kv *kvStore) Lock(owner string) (*Lock, error) {
	return kv.lockWithKeyHelper(owner, kv.getFullLockPath(cloudDriveLockKey))
}

func (kv *kvStore) Unlock(storeLock *Lock) error {
	kvp, ok := storeLock.internalLock.(*kvdb.KVPair)
	if !ok {
		return fmt.Errorf("invalid store lock provided")
	}
	return kv.k.Unlock(kvp)
}

func (kv *kvStore) getFullLockPath(key string) string {
	return kv.lockPrefix + "/" + key
}

func (kv *kvStore) LockWithKey(owner, key string) (*Lock, error) {
	fullPath := kv.getFullLockPath(key)
	kvPair, err := kv.lockWithKeyHelper(owner, fullPath)
	if err != nil {
		return nil, err
	}
	kvPair.lockedWithKey = true
	return kvPair, err
}

func (kv *kvStore) lockWithKeyHelper(owner, key string) (*Lock, error) {
	kvLock, err := kv.k.LockWithTimeout(key, owner, kv.lockTryDuration, kv.lockHoldDuration)
	if err != nil {
		return nil, err
	}
	return &Lock{Key: key, internalLock: kvLock}, nil
}

func (kv *kvStore) IsKeyLocked(key string) (bool, string, error) {
	fullPath := kv.getFullLockPath(key)
	return kv.k.IsKeyLocked(fullPath)
}

func (kv *kvStore) CreateKey(key string, value []byte) error {
	_, err := kv.k.Create(key, string(value), 0)
	return err
}

func (kv *kvStore) PutKey(key string, value []byte) error {
	_, err := kv.k.Put(key, string(value), 0)
	return err
}

func (kv *kvStore) GetKey(key string) ([]byte, error) {
	keyData, err := kv.k.Get(key)
	if err != nil {
		return nil, err
	}

	return keyData.Value, nil
}

func (kv *kvStore) DeleteKey(key string) error {
	_, err := kv.k.Delete(key)
	return err
}

func (kv *kvStore) EnumerateWithKeyPrefix(key string) ([]string, error) {
	output, err := kv.k.Enumerate(key)
	if err != nil {
		return nil, err
	}

	returnKeys := make([]string, 0)
	for _, entry := range output {
		returnKeys = append(returnKeys, entry.Key)
	}

	return returnKeys, nil
}
