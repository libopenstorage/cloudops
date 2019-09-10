package storagedistribution

import (
	"fmt"

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
  - An storage decision matrix.

  Following is the algorithm:
  - Sort the decision matrix by IOPS
  - Filter out the rows which do not meet the requested IOPS
  - Sort the filtered rows by Priority
  - Calculate minCapacityPerZone
  - (loop) For each of the filtered row:
    - Find capacityPerNode = minCapacityPerZone / instancesPerZone
    - (loop) Until capacityPerNode is less than row.MinSize*row.MinDrives
        - Reduce instancesPerZone by 1
        - if instancesPerZone reaches 0 try the next filtered row
        - else recalculate capacityPerNode using the new instancesPerZone
    - capacityPerNode is at optimum level. Found the right candidate
  - Out of both the loops, could not find a candidate.

  TODO:
   - Take into account instance types and their supported drives
   - Take into account max no. of drives that need to be attached on the instance.
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
		instStorage, instancePerZone, err :=
			getStorageDistributionCandidate(
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

// GetStorageUpdateConfig returns the storage configuration for updating an instances
// storage
func GetStorageUpdateConfig(
	request *cloudops.StoragePoolUpdateRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StoragePoolUpdateResponse, error) {
	logUpdateRequest(request)

	resp := &cloudops.StoragePoolUpdateResponse{
		InstanceStorage:     make([]*cloudops.StoragePoolSpec, 0),
		ResizeOperationType: request.ResizeOperationType,
	}

	currentCapacity := request.CurrentDriveCount * request.CurrentDriveSize
	newDeltaCapacity := request.DesiredCapacity - currentCapacity
	if newDeltaCapacity < 0 {
		return nil, &cloudops.ErrInvalidStoragePoolUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("reducing instance storage capacity is not supported"+
				"current: %d GiB requested: %d GiB", currentCapacity, request.DesiredCapacity),
		}
	}

	if newDeltaCapacity == 0 {
		return nil, cloudops.ErrCurrentCapacitySameAsDesired
	}

	if request.CurrentDriveCount > 0 && len(request.CurrentDriveType) == 0 {
		return nil, &cloudops.ErrInvalidStoragePoolUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("for storage update operation, current drive" +
				"type is required to be provided if drives already exist"),
		}
	}

	logrus.Debugf("instance currently has %d X %d GiB %s drives",
		request.CurrentDriveCount, request.CurrentDriveSize, request.CurrentDriveType)
	filteredRows := filterDecisionMatrix(decisionMatrix, request.CurrentIOPS, request.CurrentDriveType)

ROW_LOOP:
	for _, row := range filteredRows {
		switch request.ResizeOperationType {
		case api.SdkStoragePool_RESIZE_TYPE_ADD_DISK:
			// Add drives equivalent to newDeltaCapacity
			logrus.Debugf("check if we can add drive(s) for atleast: %d GiB", newDeltaCapacity)
			instStorage, err := instanceStorageForRow(row, newDeltaCapacity, request.CurrentDriveSize)
			if err != nil {
				if err == cloudops.ErrStorageDistributionCandidateNotFound {
					continue ROW_LOOP
				}

				return nil, err
			}

			resp.InstanceStorage = append(resp.InstanceStorage, instStorage)
			return resp, nil
		case api.SdkStoragePool_RESIZE_TYPE_RESIZE_DISK:
			// Resize existing drives equivalent to newDeltaCapacity
			if request.CurrentDriveCount == 0 {
				return nil, &cloudops.ErrInvalidStoragePoolUpdateRequest{
					Request: request,
					Reason: fmt.Sprintf("requested resize operation type cannot be " +
						"accomplished as no existing drives were provided"),
				}
			}

			newMinDeltaCapacityPerDrive := newDeltaCapacity / request.CurrentDriveCount

			logrus.Debugf("min delta capacity per drive needed: %d GiB num drives: %d",
				newMinDeltaCapacityPerDrive, request.CurrentDriveCount)

			// check in matrix if this drive can be resized to newMinCapacityPerDrive
			newMinCapacityPerDrive := request.CurrentDriveSize + newMinDeltaCapacityPerDrive
			logrus.Debugf("need to resize drive to atleast: %d GiB", newMinCapacityPerDrive)
			if newMinCapacityPerDrive > row.MaxSize {
				continue ROW_LOOP
			}

			resp.InstanceStorage = append(resp.InstanceStorage, &cloudops.StoragePoolSpec{
				DriveType:        row.DriveType,
				IOPS:             row.IOPS,
				DriveCount:       request.CurrentDriveCount,
				DriveCapacityGiB: newMinCapacityPerDrive,
			})

			return resp, nil
		default:
			// TODO try resize first. If doesn't satisfy, try add
			return nil, &cloudops.ErrNotSupported{
				Operation: fmt.Sprintf("GetStorageUpdateConfig with operation type: %d",
					request.ResizeOperationType),
			}
		}
	}

	return nil, cloudops.ErrStorageDistributionCandidateNotFound
}

func getStorageDistributionCandidate(
	decisionMatrix *cloudops.StorageDecisionMatrix,
	request *cloudops.StorageSpec,
	requestedInstancesPerZone uint64,
	zoneCount uint64,
) (*cloudops.StoragePoolSpec, uint64, error) {
	logDistributionRequest(request, requestedInstancesPerZone, zoneCount)

	if zoneCount <= 0 {
		return nil, 0, cloudops.ErrNumOfZonesCannotBeZero
	}

	filteredRows := filterDecisionMatrix(decisionMatrix, request.IOPS, request.DriveType)
	// Calculate min capacity per zone
	minCapacityPerZone := request.MinCapacity / uint64(zoneCount)

	for _, row := range filteredRows {
		var (
			capacityPerNode  uint64
			instancesPerZone uint64
		)
		for instancesPerZone = requestedInstancesPerZone; instancesPerZone > 0; instancesPerZone-- {
			capacityPerNode = minCapacityPerZone / uint64(instancesPerZone)
			printCandidates("Candidate", []cloudops.StorageDecisionMatrixRow{row}, instancesPerZone, capacityPerNode)

			instStorage, err := instanceStorageForRow(row, capacityPerNode, 0)
			if err != nil {
				if err == cloudops.ErrStorageDistributionCandidateNotFound {
					printCandidates("Candidate failed to satisfy requirements.",
						[]cloudops.StorageDecisionMatrixRow{row}, instancesPerZone, capacityPerNode)
					continue
				}

				logrus.Errorf("failed to get storage spec for instance: %v", err)
				return nil, 0, cloudops.ErrStorageDistributionCandidateNotFound
			}

			// additional check so that we are not overprovisioning
			for instancesPerZone > 1 && instStorage.DriveCapacityGiB*instancesPerZone > minCapacityPerZone {
				if instStorage.DriveCapacityGiB*(instancesPerZone-1) >= minCapacityPerZone {
					instancesPerZone--
				} else {
					break // no more trim down possible
				}
			}

			return instStorage, instancesPerZone, nil
		}

		printCandidates("Candidate failed as instances per zone exhausted",
			[]cloudops.StorageDecisionMatrixRow{row}, 0, capacityPerNode)

	}

	return nil, 0, cloudops.ErrStorageDistributionCandidateNotFound
}

func instanceStorageForRow(
	row cloudops.StorageDecisionMatrixRow,
	requiredCapacity uint64,
	requiredDriveSize uint64,
) (*cloudops.StoragePoolSpec, error) {
	instStorage := &cloudops.StoragePoolSpec{
		DriveType: row.DriveType,
		IOPS:      row.IOPS,
	}

	if requiredDriveSize > 0 {
		// user has asked for a drive size. We can only create/update drives
		// in increments of this size
		if requiredDriveSize >= row.MinSize && requiredDriveSize <= row.MaxSize {
			instStorage.DriveCapacityGiB = requiredDriveSize
			remainingCapacity := int(requiredCapacity)
			for remainingCapacity > 0 {
				remainingCapacity -= int(requiredDriveSize)
				instStorage.DriveCount++
			}
			return instStorage, nil
		}

		logrus.Debugf("skipping row as requiredDriveSize: %d not in [%d,%d]",
			requiredDriveSize, row.MinSize, row.MaxSize)
	} else {
		if requiredCapacity >= row.MinSize {
			// start from row's min drives to distribute the requiredCapacity across the drives
			for driveCount := row.InstanceMinDrives; driveCount <= row.InstanceMaxDrives; driveCount++ {
				driveSize := requiredCapacity / driveCount

				if driveSize > row.MaxSize || driveSize < row.MinSize {
					// drive is outside row's [min, max] bounds
					continue
				}

				instStorage.DriveCapacityGiB = driveSize
				instStorage.DriveCount = driveCount
				instStorage.IOPS = row.IOPS
				break
			}

			if instStorage.DriveCount == 0 {
				return nil, cloudops.ErrStorageDistributionCandidateNotFound
			}

		} else {
			logrus.Debugf("required capacity: %d GiB is lower than row's min size: %d GiB",
				requiredCapacity, row.MinSize)
			instStorage.DriveCapacityGiB = row.MinSize
			instStorage.DriveCount = 1
			instStorage.IOPS = row.IOPS
		}

		prettyPrintStoragePoolSpec(instStorage, "instanceStorageForRow returning")
		return instStorage, nil
	} /*else {
		logrus.Debugf("required capacity: %d GiB is lower than row's min size: %d GiB",
			requiredCapacity, row.MinSize)
	}*/

	return nil, cloudops.ErrStorageDistributionCandidateNotFound
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
			"IOPS":       candidate.IOPS,
			"MinSize":    candidate.MinSize,
			"DriveType":  candidate.DriveType,
			"Priority":   candidate.Priority,
			"DriveCount": candidate.InstanceMinDrives,
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
	}).Debugf("-- Storage Distribution Pool Request --")
}

func filterDecisionMatrix(
	decisionMatrix *cloudops.StorageDecisionMatrix,
	requiredIOPS uint64, requiredDriveType string,
) []cloudops.StorageDecisionMatrixRow {
	dm := utils.CopyDecisionMatrix(decisionMatrix)

	var filteredRows []cloudops.StorageDecisionMatrixRow
	if len(requiredDriveType) > 0 {
		for _, row := range dm.Rows {
			if row.DriveType == requiredDriveType {
				filteredRows = append(filteredRows, row)
			}
		}
	} else {
		filteredRows = dm.Rows
	}

	if requiredIOPS > 0 {
		filteredRows = utils.SortByClosestIOPS(requiredIOPS, filteredRows)
		// TODO if 2 rows have the same IOPS, sort by priority
	} else {
		filteredRows = utils.SortByIOPS(filteredRows)
		// Filter out rows which have lower IOPS than required
		var index int
		for index = 0; index < len(filteredRows); index++ {
			if filteredRows[index].IOPS >= requiredIOPS {
				break
			}
		}

		filteredRows = filteredRows[index:]
		// Sort the filtered rows by priority
		filteredRows = utils.SortByPriority(filteredRows)
	}

	if len(requiredDriveType) > 0 {
		response := make([]cloudops.StorageDecisionMatrixRow, 0)
		for _, filteredRow := range filteredRows {
			if filteredRow.DriveType == requiredDriveType {
				response = append(response, filteredRow)
			}
		}

		return response
	}

	return filteredRows
}
