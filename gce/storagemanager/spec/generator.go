package main

import (
	"fmt"
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/gce/storagemanager"
	"github.com/libopenstorage/cloudops/pkg/parser"
	"math"
)

const (
	gceYamlPath = "gce.yaml"
)

func main() {
	// Max/Min IOPS/Size data for all disks can be found below
	// https://cloud.google.com/compute/docs/disks#:~:text=Standard%20persistent%20disks%20(%20pd%2Dstandard,that%20balance%20performance%20and%20cost.
	matrixRows := getStandardDecisionMatrixRows()
	matrixRows = append(matrixRows, getSSDDecisionMatrixRows()...)
	matrixRows = append(matrixRows, getBalancedDecisionMatrixRows()...)
	matrix := cloudops.StorageDecisionMatrix{Rows: matrixRows}
	if err := parser.NewStorageDecisionMatrixParser().MarshalToYaml(&matrix, gceYamlPath); err != nil {
		fmt.Println("Failed to generate aws storage decision matrix yaml: ", err)
		return
	}
	fmt.Println("Generated gce storage decision matrix yaml at ", gceYamlPath)

}

func getBalancedDecisionMatrixRows() []cloudops.StorageDecisionMatrixRow {
	rows := []cloudops.StorageDecisionMatrixRow{}
	// 15000 IOPS is max read IOPS for Balanced persistent disks
	// 10GB is the minimum disk size. Hence, 60 iops is the minimum iops that we need to start with
	row := getCommonRow(1)
	row.DriveType = storagemanager.GCEDriveTypeBalanced
	// 6 multiplier * 10GB (min size) = 60 iops.
	row.MinIOPS = 60
	row.MaxIOPS = 100
	row.MinSize = 10
	row.MaxSize = uint64(math.Ceil(float64(100) / storagemanager.GCEBalancedIopsMultiplier))
	rows = append(rows, row)
	for iops := 100; iops < int(storagemanager.GCEBalancedMaxIopsLeast); iops = iops + 50 {
		row := getCommonRow(1)
		row.DriveType = storagemanager.GCEDriveTypeBalanced
		row.MinIOPS = uint64(iops)
		row.MaxIOPS = uint64(iops + 50)
		row.MinSize = uint64(math.Ceil(float64(iops) / storagemanager.GCEBalancedIopsMultiplier))
		row.MaxSize = uint64(math.Ceil(float64(iops+50) / storagemanager.GCEBalancedIopsMultiplier))
		rows = append(rows, row)
	}
	row = getCommonRow(1)
	row.DriveType = storagemanager.GCEDriveTypeBalanced
	row.MinIOPS = storagemanager.GCEBalancedMaxIopsLeast
	row.MaxIOPS = storagemanager.GCEBalancedMaxIopsMost
	row.MinSize = uint64(math.Ceil(float64(storagemanager.GCEBalancedMaxIopsLeast) / storagemanager.GCEBalancedIopsMultiplier))
	// 64TB is the maximum size supported by GCE
	row.MaxSize = 64000
	rows = append(rows, row)
	return rows
}

func getSSDDecisionMatrixRows() []cloudops.StorageDecisionMatrixRow {
	rows := []cloudops.StorageDecisionMatrixRow{}
	// 10GB is the minimum disk size. Hence, 300 iops is the minimum iops that we need to start with
	for iops := 300; iops < int(storagemanager.GCESSDMaxIopsLeast); iops = iops + 50 {
		row := getCommonRow(1)
		row.DriveType = storagemanager.GCEDriveTypeSSD
		row.MinIOPS = uint64(iops)
		row.MaxIOPS = uint64(iops + 50)
		row.MinSize = uint64(math.Ceil(float64(iops) / storagemanager.GCESSDIopsMultiplier))
		row.MaxSize = uint64(math.Ceil(float64(iops+50) / storagemanager.GCESSDIopsMultiplier))
		rows = append(rows, row)
	}
	// Last row accounts for ranged maxIOPs
	row := getCommonRow(1)
	row.DriveType = storagemanager.GCEDriveTypeSSD
	row.MinIOPS = storagemanager.GCESSDMaxIopsLeast
	row.MaxIOPS = storagemanager.GCESSDMaxIopsMost
	row.MinSize = uint64(math.Ceil(float64(storagemanager.GCESSDMaxIopsLeast) / storagemanager.GCESSDIopsMultiplier))
	// 64TB is the maximum size supported by GCE
	row.MaxSize = 64000
	rows = append(rows, row)
	return rows
}

func getStandardDecisionMatrixRows() []cloudops.StorageDecisionMatrixRow {
	rows := []cloudops.StorageDecisionMatrixRow{}
	// First row has min and max 100 IOPS for 0 - 134Gi
	row := getCommonRow(0)
	row.DriveType = storagemanager.GCEDriveTypeStandard
	// .75 multiplier * 10GB = ciel(7.5) iops.
	row.MinIOPS = 8
	row.MaxIOPS = 50
	row.MinSize = 10
	row.MaxSize = uint64(math.Ceil(float64(50) / storagemanager.GCEStandardIopsMultiplier))
	rows = append(rows, row)
	// 7500 IOPS is max read IOPS for Zonal standard persistent disks
	for iops := 50; iops < int(storagemanager.GCEStandardMaxIops); iops = iops + 50 {
		row := getCommonRow(0)
		row.DriveType = storagemanager.GCEDriveTypeStandard
		row.MinIOPS = uint64(iops)
		row.MaxIOPS = uint64(iops + 50)
		row.MinSize = uint64(math.Ceil(float64(iops) / storagemanager.GCEStandardIopsMultiplier))
		row.MaxSize = uint64(math.Ceil(float64(iops+50) / storagemanager.GCEStandardIopsMultiplier))
		rows = append(rows, row)
	}
	// Last row has min and max 7500 IOPS and max size of 64TB
	row = getCommonRow(0)
	row.DriveType = storagemanager.GCEDriveTypeStandard
	row.MinIOPS = storagemanager.GCEStandardMaxIops
	row.MaxIOPS = storagemanager.GCEStandardMaxIops
	row.MinSize = uint64(storagemanager.GCEStandardIopsMultiplier * float64(storagemanager.GCEStandardMaxIops))
	// 64 TB is the max size of GCE disk
	row.MaxSize = 64000
	rows = append(rows, row)
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
