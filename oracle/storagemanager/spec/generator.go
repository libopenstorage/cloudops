package main

import (
	"fmt"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/parser"
)

const (
	oracleYamlPath = "oracle.yaml"
	vpusSuffix     = "_vpus"
)

func main() {
	// Max/Min IOPS/Size data for all disks can be found below
	// https://docs.oracle.com/en-us/iaas/Content/Block/Concepts/blockvolumeperformance.htm
	matrixRows := []cloudops.StorageDecisionMatrixRow{}
	for vpu := 0; vpu <= 120; vpu = vpu + 10 {
		matrixRows = append(matrixRows, getMatrixRows(vpu)...)
	}
	matrix := cloudops.StorageDecisionMatrix{Rows: matrixRows}
	if err := parser.NewStorageDecisionMatrixParser().MarshalToYaml(&matrix, oracleYamlPath); err != nil {
		fmt.Println("Failed to generate oracle storage decision matrix yaml: ", err)
		return
	}
	fmt.Println("Generated oracle storage decision matrix yaml at ", oracleYamlPath)

}

func getMatrixRows(vpu int) []cloudops.StorageDecisionMatrixRow {
	var iopsPerGB, maxIopsPerVol int64
	rows := []cloudops.StorageDecisionMatrixRow{}
	switch vpu {
	case 0:
		iopsPerGB = 2
		maxIopsPerVol = 3000
	case 10:
		iopsPerGB = 60
		maxIopsPerVol = 25000
	case 20:
		iopsPerGB = 75
		maxIopsPerVol = 50000
	case 30:
		iopsPerGB = 90
		maxIopsPerVol = 75000
	case 40:
		iopsPerGB = 105
		maxIopsPerVol = 100000
	case 50:
		iopsPerGB = 120
		maxIopsPerVol = 125000
	case 60:
		iopsPerGB = 135
		maxIopsPerVol = 150000
	case 70:
		iopsPerGB = 150
		maxIopsPerVol = 175000
	case 80:
		iopsPerGB = 165
		maxIopsPerVol = 200000
	case 90:
		iopsPerGB = 180
		maxIopsPerVol = 225000
	case 100:
		iopsPerGB = 195
		maxIopsPerVol = 250000
	case 110:
		iopsPerGB = 210
		maxIopsPerVol = 275000
	case 120:
		iopsPerGB = 225
		maxIopsPerVol = 300000
	}
	row := getCommonRow(0)

	for iops := 0; iops < int(maxIopsPerVol); iops = iops + 50 {
		row.DriveType = fmt.Sprintf("%d%s", vpu, vpusSuffix)
		row.MinIOPS = uint64(iops)
		row.MaxIOPS = uint64(iops + 50)
		row.MinSize = row.MinIOPS / uint64(iopsPerGB)
		row.MaxSize = row.MaxIOPS / uint64(iopsPerGB)
		rows = append(rows, row)
	}
	return rows
}

func getCommonRow(priority int) cloudops.StorageDecisionMatrixRow {
	return cloudops.StorageDecisionMatrixRow{
		InstanceType:      "*",
		InstanceMaxDrives: 8,
		InstanceMinDrives: 1,
		Region:            "*",
		Priority:          priority,
	}
}
