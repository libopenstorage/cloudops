package storagedistribution

import (
	"fmt"
	"math"

	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/utils"
	"github.com/libopenstorage/openstorage/api"
	"github.com/sirupsen/logrus"
)

/*
   ====================================
   Storage Distribution Algorithm
   ====================================

  The storage distribution algorithm provides an optimum
  storage distribution strategy for a given set of following inputs:
  - Requested IOPS from cloud storage.
  - Minimum capacity for the whole cluster.
  - Number of zones in the cluster.
  - Number of instances in the cluster.
  - A storage decision matrix.

  TODO:
   - Take into account instance types and their supported drives
   - Take into account the effect on the overall throughput when multiple drives are attached
     on the same instance.
*/

// GetStorageDistribution returns the storage distribution
// for the provided request and decision matrix
func GetStorageDistribution(
	request *cloudops.StorageDistributionRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StorageDistributionResponse, error) {
	response := &cloudops.StorageDistributionResponse{}
	for _, userRequest := range request.UserStorageSpec {
		// for for request, find how many instances per zone needs to have storage
		// and the storage spec for each of them
		instStorage, instancePerZone, err :=
			getStorageDistributionCandidateForPool(
				decisionMatrix,
				userRequest,
				request.InstancesPerZone,
				request.ZoneCount,
			)
		if err != nil {
			return nil, err
		}
		response.InstanceStorage = append(
			response.InstanceStorage,
			&cloudops.StoragePoolSpec{
				DriveCapacityGiB: instStorage.DriveCapacityGiB,
				DriveType:        instStorage.DriveType,
				InstancesPerZone: instancePerZone,
				DriveCount:       instStorage.DriveCount,
				IOPS:             instStorage.IOPS,
			},
		)

	}
	return response, nil
}

// GetStorageUpdateConfig returns the storage configuration for updating
// an instance's storage based on the requested new capacity.
// To meet the new capacity requirements this function with either:
// - Resize existing disks
// - Add more disks
// This is based of the ResizeOperationType input argument. If no such input is
// provided then this function tries Resize first and then an Add.
// The algorithms for Resize and Add are explained with their respective function
// definitions.
func GetStorageUpdateConfig(
	request *cloudops.StoragePoolUpdateRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StoragePoolUpdateResponse, error) {
	logUpdateRequest(request)

	switch request.ResizeOperationType {
	case api.SdkStoragePool_RESIZE_TYPE_ADD_DISK:
		// Add drives equivalent to newDeltaCapacity
		return AddDisk(request, decisionMatrix)
	case api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK:
		// Resize existing drives equivalent to newDeltaCapacity
		return ResizeDisk(request, decisionMatrix)
	default:
		// Auto-mode. Try resize first then add
		resp, err := ResizeDisk(request, decisionMatrix)
		if err != nil {
			return AddDisk(request, decisionMatrix)
		}
		return resp, err
	}
}

// AddDisk tries to satisfy the StoragePoolUpdateRequest by adding more disks
// to the existing storage pool. Following is a high level algorithm/steps used
// to achieve this:
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// - Calculate deltaCapacity = input.RequestedCapacity - input.CurrentCapacity					 //
// - Calculate currentDriveSize from the request.								 //
// - Calculate the requiredDriveCount for achieving the deltaCapacity.						 //
// - Find out if any rows from the decision matrix fit in our new configuration					 //
//      - Filter out the rows which do not have the same input.DriveType					 //
//      - Filter out rows which do not fit input.CurrentDriveSize in row.MinSize and row.MaxSize		 //
//      - Filter out rows which do not fit requiredDriveCount in row.InstanceMinDrives and row.InstanceMaxDrives //
// - Pick the 1st row from the decision matrix as your candidate.						 //
// - If no row found:												 //
//     - failed to AddDisk											 //
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func AddDisk(
	request *cloudops.StoragePoolUpdateRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StoragePoolUpdateResponse, error) {
	if err := validateUpdateRequest(request); err != nil {
		return nil, err
	}

	currentCapacity := request.CurrentDriveCount * request.CurrentDriveSize
	deltaCapacity := request.DesiredCapacity - currentCapacity

	logrus.Debugf("check if we can add drive(s) for atleast: %d GiB", deltaCapacity)
	dm := utils.CopyDecisionMatrix(decisionMatrix)

	currentDriveSize := request.CurrentDriveSize
	if currentDriveSize == 0 {
		// No drives have been provisioned yet.
		// Lets select a row witch matches the deltaCapacity
		// TODO: Need to start with different increments here. (Vsphere: TestCase: 8)
		currentDriveSize = deltaCapacity
	}

	// Calculate the driveCount required to fit the deltaCapacity
	requiredDriveCount := uint64(math.Ceil(float64(deltaCapacity) / float64(currentDriveSize)))

	updatedTotalDrivesOnNodes := requiredDriveCount + request.TotalDrivesOnNode

	// Filter the decision matrix and check if there any rows which satisfy our requirements.
	dm.FilterByDriveType(request.CurrentDriveType).
		FilterByDriveSize(currentDriveSize).
		FilterByDriveCount(updatedTotalDrivesOnNodes)

	if len(dm.Rows) == 0 {
		return nil, cloudops.ErrStorageDistributionCandidateNotFound
	}
	row := dm.Rows[0]
	printCandidates("AddDisk Candidate", []cloudops.StorageDecisionMatrixRow{row}, 0, 0)

	instStorage := &cloudops.StoragePoolSpec{
		DriveType:        row.DriveType,
		IOPS:             row.IOPS,
		DriveCapacityGiB: currentDriveSize,
		DriveCount:       uint64(requiredDriveCount),
	}
	prettyPrintStoragePoolSpec(instStorage, "AddDisk")
	resp := &cloudops.StoragePoolUpdateResponse{
		InstanceStorage:     []*cloudops.StoragePoolSpec{instStorage},
		ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_ADD_DISK,
	}
	return resp, nil
}

// ResizeDisk tries to satisfy the StoragePoolUpdateRequest by expanding existing disks
// from the storage pool. Following is a high level algorithm/steps used
// to achieve this:
//////////////////////////////////////////////////////////////////////////////////////////////////
// - Calculate deltaCapacity = input.RequestedCapacity - input.CurrentCapacity		        //
// - Calculate deltaCapacityPerDrive = deltaCapacityPerNode / input.CurrentNumberOfDrivesInPool //
// - Filter out the rows which do not have the same input.DriveType			        //
// - Filter out the rows which do not have the same input.IOPS				        //
// - Filter out rows which do not fit input.CurrentDriveSize in row.MinSize and row.MaxSize     //
// - Sort the rows by IOPS								        //
// - First row in the filtered decision matrix is our best candidate.			        //
// - If input.CurrentDriveSize + deltaCapacityPerDrive > row.MaxSize:			        //
//       - failed to expand								        //
//   Else										        //
//       - success									        //
//////////////////////////////////////////////////////////////////////////////////////////////////
func ResizeDisk(
	request *cloudops.StoragePoolUpdateRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StoragePoolUpdateResponse, error) {
	if err := validateUpdateRequest(request); err != nil {
		return nil, err
	}

	if request.CurrentDriveCount == 0 {
		return nil, &cloudops.ErrInvalidStoragePoolUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("requested resize operation type cannot be " +
				"accomplished as no existing drives were provided"),
		}
	}

	currentCapacity := request.CurrentDriveCount * request.CurrentDriveSize
	deltaCapacity := request.DesiredCapacity - currentCapacity
	deltaCapacityPerDrive := deltaCapacity / request.CurrentDriveCount

	dm := utils.CopyDecisionMatrix(decisionMatrix)

	// Filter the decision matrix
	dm.FilterByDriveType(request.CurrentDriveType).
		FilterByIOPS(request.CurrentIOPS).
		FilterByDriveSize(request.CurrentDriveSize).
		SortByIOPS()

	// We select the first matching row of the matrix as it satisfies the following:
	// 1. same drive type
	// 2. drive size lies between row's min and max size
	// 3. row's IOPS is closest to the current IOPS

	if len(dm.Rows) == 0 {
		return nil, cloudops.ErrStorageDistributionCandidateNotFound
	}

	row := dm.Rows[0]
	printCandidates("ResizeDisk Candidate", []cloudops.StorageDecisionMatrixRow{row}, 0, 0)
	if request.CurrentDriveSize+deltaCapacityPerDrive > row.MaxSize {
		return nil, cloudops.ErrStorageDistributionCandidateNotFound
	}

	instStorage := &cloudops.StoragePoolSpec{
		DriveType:        row.DriveType,
		IOPS:             row.IOPS,
		DriveCapacityGiB: request.CurrentDriveSize + deltaCapacityPerDrive,
		DriveCount:       request.CurrentDriveCount,
	}
	prettyPrintStoragePoolSpec(instStorage, "ResizeDisk")
	resp := &cloudops.StoragePoolUpdateResponse{
		InstanceStorage:     []*cloudops.StoragePoolSpec{instStorage},
		ResizeOperationType: api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK,
	}
	return resp, nil

}

// getStorageDistributionCandidateForPool() tries to determine a drive configuration
// to satisfy the input storage pool requirements. Following is a high level algorithm/steps used
// to achieve this:
//
//////////////////////////////////////////////////////////////////////////////
// - Calculate minCapacityPerZone = input.MinCapacity / zoneCount	    //
// - Calculate maxCapacityPerZone = input.MaxCapacity / zoneCount	    //
// - Filter the decision matrix based of our requirements:		    //
//     - Filter out the rows which do not have the same input.DriveType	    //
//     - Filter out the rows which do not meet input.IOPS		    //
//     - Sort the decision matrix by IOPS				    //
//     - Sort the decision matrix by Priority				    //
// - instancesPerZone = input.RequestedInstancesPerZone			    //
// - (row_loop) For each of the filtered row:				    //
//     - (instances_per_zone_loop) For instancesPerZone > 0:		    //
//         - Find capacityPerNode = minCapacityPerZone / instancesPerZone   //
//             - (drive_count_loop) For driveCount > row.InstanceMinDrives: //
//                 - driveSize = capacityPerNode / driveCount		    //
//                 - If driveSize within row.MinSize and row.MaxSize:	    //
//                     break drive_count_loop (Found candidate)		    //
//             - If (drive_count_loop) fails/exhausts:			    //
//                   - reduce instancesPerZone by 1			    //
//                   - goto (instances_per_zone_loop)			    //
//               Else found candidate					    //
//                   - break instances_per_zone_loop (Found candidate)	    //
//     - If (instances_per_zone_loop) fails:				    //
//         - Try the next filtered row					    //
//         - goto (row_loop)						    //
// - If (row_loop) fails:						    //
//       - failed to get a candidate					    //
//////////////////////////////////////////////////////////////////////////////
func getStorageDistributionCandidateForPool(
	decisionMatrix *cloudops.StorageDecisionMatrix,
	request *cloudops.StorageSpec,
	requestedInstancesPerZone uint64,
	zoneCount uint64,
) (*cloudops.StoragePoolSpec, uint64, error) {
	logDistributionRequest(request, requestedInstancesPerZone, zoneCount)

	if zoneCount <= 0 {
		return nil, 0, cloudops.ErrNumOfZonesCannotBeZero
	}

	// Filter the decision matrix rows based on the input request
	dm := utils.CopyDecisionMatrix(decisionMatrix)
	dm.FilterByDriveType(request.DriveType).
		FilterByIOPS(request.IOPS).
		SortByIOPS().
		SortByPriority()

	// Calculate min and max capacity per zone
	minCapacityPerZone := request.MinCapacity / uint64(zoneCount)
	maxCapacityPerZone := request.MaxCapacity / uint64(zoneCount)
	var (
		capacityPerNode, instancesPerZone, driveCount, driveSize uint64
		row                                                      cloudops.StorageDecisionMatrixRow
		rowIndex                                                 int
	)

row_loop:
	for rowIndex := uint64(0); rowIndex < uint64(len(dm.Rows)); rowIndex++ {
		row = dm.Rows[rowIndex]
		// Favour maximum instances per zone
	instances_per_zone_loop:
		for instancesPerZone = requestedInstancesPerZone; instancesPerZone > 0; instancesPerZone-- {
			capacityPerNode = minCapacityPerZone / uint64(instancesPerZone)
			printCandidates("Candidate", []cloudops.StorageDecisionMatrixRow{row}, instancesPerZone, capacityPerNode)

			// Favour maximum drive count
			// drive_count_loop:
			foundCandidate := false
			for driveCount = row.InstanceMaxDrives; driveCount >= row.InstanceMinDrives; driveCount-- {
				driveSize = capacityPerNode / driveCount
				if driveSize >= row.MinSize && driveSize <= row.MaxSize {
					// Found a candidate
					foundCandidate = true
					break
				}
				if driveCount == row.InstanceMinDrives {
					// We have exhausted the drive_count_loop
					if driveSize < row.MinSize {
						// If the last calculated driveSize is less than row.MinSize
						// that indicates none of the driveSizes in the drive_count_loop
						// were greater than row.MinSize. Lets try with row.MinSize
						driveSize = row.MinSize
						driveCount = row.InstanceMinDrives
						if driveSize*instancesPerZone < maxCapacityPerZone {
							// Found a candidate
							foundCandidate = true
							break
						}
					}
				}
			}

			if !foundCandidate {
				// drive_count_loop failed
				continue instances_per_zone_loop
			}
			break instances_per_zone_loop
		}

		if instancesPerZone == 0 {
			// instances_per_zone_loop failed
			continue row_loop
		}
		// break row_loop
		break row_loop
	}

	if rowIndex == len(dm.Rows) {
		// row_loop failed
		return nil, 0, cloudops.ErrStorageDistributionCandidateNotFound
	}

	// optimize instances per zone
	var optimizedInstancesPerZone uint64
	for optimizedInstancesPerZone = uint64(1); optimizedInstancesPerZone < instancesPerZone; optimizedInstancesPerZone++ {
		// Check if we can satisfy the minCapacityPerZone for this optimizedInstancesPerZone, driveCount and driveSize
		if minCapacityPerZone > optimizedInstancesPerZone*driveCount*driveSize {
			// we are not satisfying the minCapacityPerZone
			continue
		}
		break
	}
	instStorage := &cloudops.StoragePoolSpec{
		DriveType:        row.DriveType,
		IOPS:             row.IOPS,
		DriveCapacityGiB: driveSize,
		DriveCount:       driveCount,
	}
	prettyPrintStoragePoolSpec(instStorage, "getStorageDistributionCandidate returning")
	return instStorage, optimizedInstancesPerZone, nil

}

// validateUpdateRequest validates the StoragePoolUpdateRequest
func validateUpdateRequest(
	request *cloudops.StoragePoolUpdateRequest,
) error {
	currentCapacity := request.CurrentDriveCount * request.CurrentDriveSize
	newDeltaCapacity := request.DesiredCapacity - currentCapacity

	if newDeltaCapacity < 0 {
		return &cloudops.ErrInvalidStoragePoolUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("reducing instance storage capacity is not supported"+
				"current: %d GiB requested: %d GiB", currentCapacity, request.DesiredCapacity),
		}
	}

	if newDeltaCapacity == 0 {
		return cloudops.ErrCurrentCapacitySameAsDesired
	}

	if request.CurrentDriveCount > 0 && len(request.CurrentDriveType) == 0 {
		return &cloudops.ErrInvalidStoragePoolUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("for storage update operation, current drive" +
				"type is required to be provided if drives already exist"),
		}
	}
	return nil
}

func prettyPrintStoragePoolSpec(spec *cloudops.StoragePoolSpec, prefix string) {
	logrus.Infof("%s instStorage: %d X %d GiB %s drives", prefix, spec.DriveCount,
		spec.DriveCapacityGiB, spec.DriveType)
}

func printCandidates(
	msg string,
	candidates []cloudops.StorageDecisionMatrixRow,
	instancePerZone uint64,
	capacityPerNode uint64,
) {
	for _, candidate := range candidates {
		logrus.WithFields(logrus.Fields{
			"IOPS":              candidate.IOPS,
			"MinSize":           candidate.MinSize,
			"MaxSize":           candidate.MaxSize,
			"DriveType":         candidate.DriveType,
			"Priority":          candidate.Priority,
			"InstanceMinDrives": candidate.InstanceMinDrives,
			"InstanceMaxDrives": candidate.InstanceMaxDrives,
		}).Debugf("%v for %v instances per zone with a total capacity per node %v",
			msg, instancePerZone, capacityPerNode)
	}
}

func logDistributionRequest(
	request *cloudops.StorageSpec,
	requestedInstancesPerZone uint64,
	zoneCount uint64,
) {
	logrus.WithFields(logrus.Fields{
		"IOPS":             request.IOPS,
		"MinCapacity":      request.MinCapacity,
		"InstancesPerZone": requestedInstancesPerZone,
		"ZoneCount":        zoneCount,
	}).Debugf("-- Storage Distribution Pool Request --")
}

func logUpdateRequest(
	request *cloudops.StoragePoolUpdateRequest,
) {
	logrus.WithFields(logrus.Fields{
		"MinCapacity":   request.DesiredCapacity,
		"OperationType": request.ResizeOperationType,
	}).Debugf("-- Storage Distribution Pool Update Request --")
}
