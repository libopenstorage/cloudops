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
	request *cloudops.StorageUpdateRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StorageUpdateResponse, error) {
	logUpdateRequest(request)

	resp := &cloudops.StorageUpdateResponse{
		InstanceStorage:     make([]*cloudops.StoragePoolSpec, 0),
		ResizeOperationType: request.ResizeOperationType,
	}

	var (
		currentCapacity   uint64
		currentNumDrives  uint64
		existingDriveSize uint64
		existingDriveType string
	)

	for _, spec := range request.CurrentInstanceStorage {
		currentCapacity += uint64(spec.DriveCount) * spec.DriveCapacityGiB
		currentNumDrives += uint64(spec.DriveCount)
	}

	newDeltaCapacity := request.NewCapacity - currentCapacity
	if newDeltaCapacity < 0 {
		return nil, &cloudops.ErrInvalidStorageUpdateRequest{
			Request: request,
			Reason: fmt.Sprintf("reducing instance storage capacity is not supported"+
				"current: %d GiB requested: %d GiB", currentCapacity, request.NewCapacity),
		}
	}

	if len(request.CurrentInstanceStorage) > 0 {
		existingDriveType = request.CurrentInstanceStorage[0].DriveType
		existingDriveSize = request.CurrentInstanceStorage[0].DriveCapacityGiB
		for _, currentSpec := range request.CurrentInstanceStorage {
			if currentSpec.DriveCapacityGiB != existingDriveSize {
				return nil, &cloudops.ErrInvalidStorageUpdateRequest{
					Request: request,
					Reason: fmt.Sprintf("for ADD operation type for resize, all current" +
						" drives on the instances need to be of the same size."),
				}
			}

			if currentSpec.DriveType != existingDriveType {
				return nil, &cloudops.ErrInvalidStorageUpdateRequest{
					Request: request,
					Reason: fmt.Sprintf("for ADD operation type for resize, all current" +
						" drives on the instances need to be of the same type."),
				}
			}
		}

		if len(existingDriveType) == 0 {
			return nil, &cloudops.ErrInvalidStorageUpdateRequest{
				Request: request,
				Reason: fmt.Sprintf("for storage update operation, current drive" +
					"type is required to be provided"),
			}
		}

		logrus.Debugf("existing drive type: %s and size: %d", existingDriveType, existingDriveSize)
	}

	filteredRows := filterDecisionMatrix(decisionMatrix, request.NewIOPS, existingDriveType)

ROW_LOOP:
	for _, row := range filteredRows {
		switch request.ResizeOperationType {
		case api.StoragePoolResizeOperationType_ADD_DISK:
			// Add drives equivalent to newDeltaCapacity
			logrus.Debugf("need to add drive(s) for atleast: %d GiB", newDeltaCapacity)
			instStorage, err := instanceStorageForRow(row, newDeltaCapacity, existingDriveSize)
			if err != nil {
				if err == cloudops.ErrStorageDistributionCandidateNotFound {
					continue ROW_LOOP
				}

				return nil, err
			}

			resp.InstanceStorage = append(resp.InstanceStorage, instStorage)
			return resp, nil
		case api.StoragePoolResizeOperationType_RESIZE_DISK:
			// Resize existing drives equivalent to newDeltaCapacity
			if len(request.CurrentInstanceStorage) == 0 {
				return nil, &cloudops.ErrInvalidStorageUpdateRequest{
					Request: request,
					Reason: fmt.Sprintf("requested resize operation type cannot be " +
						"accomplished as no existing drives were provided"),
				}
			}

			newMinDeltaCapacityPerDrive := newDeltaCapacity / currentNumDrives

			logrus.Debugf("min delta capacity per drive needed: %d GiB num drives: %d",
				newMinDeltaCapacityPerDrive, currentNumDrives)

			for _, currentSpec := range request.CurrentInstanceStorage {
				// check in matrix if this drive can be resized to newMinCapacityPerDrive
				newMinCapacityPerDrive := currentSpec.DriveCapacityGiB + newMinDeltaCapacityPerDrive
				logrus.Debugf("need to resize drive to atleast: %d GiB", newMinCapacityPerDrive)
				instStorage, err := instanceStorageForRow(row, newMinCapacityPerDrive, 0)
				if err != nil {
					if err == cloudops.ErrStorageDistributionCandidateNotFound {
						continue ROW_LOOP
					}

					return nil, err
				}
				instStorage.DriveCount = currentSpec.DriveCount
				resp.InstanceStorage = append(resp.InstanceStorage, instStorage)
			}

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

	filteredRows := filterDecisionMatrix(decisionMatrix, request.IOPS, request.DriveType)
	if zoneCount <= 0 {
		return nil, 0, cloudops.ErrNumOfZonesCannotBeZero
	}
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
				logrus.Debugf("[debug] need to trim down")
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
			// start from candidateRow.MinSize to distribute the newCapacity across the drives
			for driveCount := row.InstanceMinDrives; driveCount <= row.InstanceMaxDrives; driveCount++ {
				driveSize := row.MaxSize / driveCount
				if driveCount != 1 {
					driveSize++
				}

				capacityWithDrives := driveCount * driveSize
				if capacityWithDrives < requiredCapacity {
					continue
				}

				if requiredCapacity < driveSize {
					instStorage.DriveCapacityGiB = requiredCapacity
				} else {
					instStorage.DriveCapacityGiB = driveSize
				}
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

		return instStorage, nil
	} /*else {
		logrus.Debugf("required capacity: %d GiB is lower than row's min size: %d GiB",
			requiredCapacity, row.MinSize)
	}*/

	return nil, cloudops.ErrStorageDistributionCandidateNotFound
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
	request *cloudops.StorageUpdateRequest,
) {
	logrus.WithFields(logrus.Fields{
		"IOPS":          request.NewIOPS,
		"MinCapacity":   request.NewCapacity,
		"OperationType": request.ResizeOperationType,
	}).Debugf("-- Storage Distribution Pool Request --")
}

func filterDecisionMatrix(
	decisionMatrix *cloudops.StorageDecisionMatrix,
	requiredIOPS uint64, requiredDriveType string) []cloudops.StorageDecisionMatrixRow {
	dm := utils.CopyDecisionMatrix(decisionMatrix)

	var filteredRows []cloudops.StorageDecisionMatrixRow
	if requiredIOPS > 0 {
		dm.Rows = utils.SortByClosestIOPS(requiredIOPS, dm.Rows)
		// TODO if 2 rows have the same IOPS, sort by priority
		filteredRows = dm.Rows
	} else {
		dm.Rows = utils.SortByIOPS(dm.Rows)
		// Filter out rows which have lower IOPS than required
		var index int
		for index = 0; index < len(dm.Rows); index++ {
			if dm.Rows[index].IOPS >= requiredIOPS {
				break
			}
		}

		filteredRows = dm.Rows[index:]
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
