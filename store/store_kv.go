package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/portworx/kvdb"
	"github.com/sirupsen/logrus"
)

const (
	cloudDriveKey     = "clouddrive/"
	cloudDriveLockKey = "_lock"
	maxWatchErrors    = 5
	defaultLockTryDuration = 1 * time.Minute
	defaultLockHoldDuration = 3 * time.Minute
)

var (
	// ErrKvdbNotInitialized is returned when kvdb is not initialized
	ErrKvdbNotInitialized = errors.New("KVDB is not initialized")
)

type kvStore struct {
	k                kvdb.Kvdb
	keyName          string
	lockTryDuration  time.Duration
	lockHoldDuration time.Duration
}

// NewKVStore returns a Store implementation which is a wrapper over
// kvdb.
func NewKVStore(kv kvdb.Kvdb) (Store, error) {
	return NewKVStoreWithParams(kv, cloudDriveLockKey, 0, 0)
}

// NewKVStoreWithParams returns a Store implementation which is a wrapper over
// kvdb.
func NewKVStoreWithParams(
	kv kvdb.Kvdb,
	name string,
	lockTryDuration time.Duration,
	lockHoldDuration time.Duration,
) (Store, error) {
	kstore := kvStore{}
	if kv == nil {
		return nil, ErrKvdbNotInitialized
	}
	if lockTryDuration != 0 {
		kstore.lockTryDuration = lockTryDuration
	} else {
		kstore.lockTryDuration= kv.GetLockTryDuration()
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
	kstore.keyName = cloudDriveKey + name
	return &kstore, nil
}

func (kv *kvStore) Lock(owner string) (*StoreLock, error) {
	return kv.lockWithKeyHelper(owner, kv.keyName)
}

func (kv *kvStore) Unlock(storeLock *StoreLock) error {
	kvp, ok := storeLock.internalLock.(*kvdb.KVPair)
	if !ok {
		return fmt.Errorf("Invalid StoreLock provided")
	}
	return kv.k.Unlock(kvp)
}

func (kv *kvStore) LockWithKey(owner, key string) (*StoreLock, error) {
	fullPath := kv.keyName + "/" + key
	return kv.lockWithKeyHelper(owner, fullPath)
}

func (kv *kvStore) lockWithKeyHelper(owner, key string) (*StoreLock, error) {
	kvLock, err := kv.k.LockWithTimeout(key, owner, kv.lockTryDuration, kv.lockHoldDuration)
	if err != nil {
		return nil, err
	}
	return &StoreLock{Key: key, internalLock: kvLock}, nil
}

func (kv *kvStore) IsKeyLocked(key string) (bool, string, error) {
	return kv.k.IsKeyLocked(key)
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

func (kv *kvStore) EnumerateKey(key string) ([]string, error) {
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
