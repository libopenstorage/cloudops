package main

import (
	"fmt"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/ibm/storagemanager"
	"github.com/libopenstorage/cloudops/pkg/parser"
)

// This is an exact copy of the generator for AWS decision matrix
// but it uses the limits prescribed by IBM as per this doc -
// https://cloud.ibm.com/docs/vpc?topic=vpc-block-storage-profiles&interface=ui#tiers

const (
	ibmYamlPath = "ibm.yaml"
)

func main() {
	matrixRows := append(
		getIopsTierStorageDecisionMatrixRows(
			30,
			48000,
			storagemanager.DriveType3IOPSTierMultiplier,
			storagemanager.DriveType3IOPSTier,
		),
		getIopsTierStorageDecisionMatrixRows(
			50,
			48000,
			storagemanager.DriveType5IOPSTierMultiplier,
			storagemanager.DriveType5IOPSTier,
		)...)

	matrixRows = append(
		matrixRows,
		getIopsTierStorageDecisionMatrixRows(
			100,
			48000,
			storagemanager.DriveType10IOPSTierMultiplier,
			storagemanager.DriveType10IOPSTier,
		)...)

	// General Purpose drive type is just another name for the 3 IOPS tier
	matrixRows = append(
		matrixRows,
		getIopsTierStorageDecisionMatrixRows(
			30,
			48000,
			storagemanager.DriveTypeGeneralPurposeMultiplier,
			storagemanager.DriveTypeGeneralPurpose,
		)...)

	matrix := cloudops.StorageDecisionMatrix{Rows: matrixRows}
	if err := parser.NewStorageDecisionMatrixParser().MarshalToYaml(&matrix, ibmYamlPath); err != nil {
		fmt.Println("Failed to generate ibm storage decision matrix yaml: ", err)
		return
	}
	fmt.Println("Generated ibm storage decision matrix yaml at ", ibmYamlPath)
}

// getIopsTierStorageDecisionMatrixRows will programmatically generate rows for IOPS tier drive type
func getIopsTierStorageDecisionMatrixRows(
	minIops uint64,
	maxIops uint64,
	iopsMultiplier uint64,
	driveType string,
) []cloudops.StorageDecisionMatrixRow {
	rows := []cloudops.StorageDecisionMatrixRow{}
	i := 0
	for iops := minIops; iops < maxIops; {
		row := getCommonRow(0)
		row.DriveType = driveType
		row.MinIOPS = iops
		if i == 0 {
			// First row starts at either 30/50/100
			row.MaxIOPS = 1000
			iops = 1000
			i = i + 1
		} else {
			row.MaxIOPS = iops + 1000
			iops = iops + 1000
		}
		row.MinSize = uint64(row.MinIOPS / iopsMultiplier)
		row.MaxSize = uint64(row.MaxIOPS / iopsMultiplier)
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
