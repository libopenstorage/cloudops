package oracle

import (
	"fmt"
	"testing"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/test"
)

func TestAll(t *testing.T) {
	drivers := make(map[string]cloudops.Ops)
	diskTemplates := make(map[string]map[string]interface{})

	d, err := NewClient()
	if err != nil {
		fmt.Printf("err : %+v", err)
		t.Skipf("skipping Oracle tests as environment is not set...\n")
	}
	drivers[d.Name()] = d
	test.RunTest(drivers, diskTemplates, sizeCheck, t)
}

func sizeCheck(template interface{}, targetSize uint64) bool {
	// TODO: implement it right way
	return true
}
