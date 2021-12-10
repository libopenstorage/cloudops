package backoff

import (
	"reflect"
	"testing"
)

func TestVolumeIdsToString(t *testing.T) {
	expectedVolumeIdsStr := []string{"123", "456"}
	var volumeIds []*string
	for i := range expectedVolumeIdsStr {
		volumeIds = append(volumeIds, &expectedVolumeIdsStr[i])
	}
	result, _ := volumeIdsStringDereference(volumeIds)

	if isEqual := reflect.DeepEqual(expectedVolumeIdsStr, result); !isEqual {
		t.Error("volumeIds doesn't match dereferenced value. got: ", result, "; expected: ", expectedVolumeIdsStr)
	}

}

func TestVolumeIdsToStringWithNil(t *testing.T) {
	expectedVolumeIdsStr := []string{"123", "456"}
	var volumeIds []*string
	for i := range expectedVolumeIdsStr {
		volumeIds = append(volumeIds, &expectedVolumeIdsStr[i])
	}
	volumeIds = append(volumeIds, nil)
	result, err := volumeIdsStringDereference(volumeIds)

	if result != nil || err == nil {
		t.Error("result expected to be nil got: ", result, ", error value: ", err)
	}

}
