package common

import (
	"github.com/libopenstorage/cloudops"
	"github.com/libopenstorage/cloudops/pkg/utils"
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

  It uses the following algorithm:
  1. For the requested instances per zone and total zone count
    2. Calculate the number of total storage instances
    3. Divide the minimum capacity with total instances to get "PerNodeCapacity"
    4. Check with the matrix if PerNodeCapacity satisfies the requested IOPS for any drive
    5. Generate a list of candidates (drive types and sizes) that satisfy the requirements
    6. Sort the candidates by closest to requested IOPS such that the first element would
       be closest to the requested IOPS
    7. Remove the candidates whose IOPS are greater than requested by a factor of 1000 (constant could be changed?)
    8. If there are still multiple candidates use priority.
    9. If no candidates were found in this process reduce the requested instances per zone by 1 and go to Step 1.

  TODO:
   - Take into account DriveCount while determing the best choice
   - Take into account instance types and their supported drives
   - Take into account max no. of drives that need to be attached on the instance.
*/

// GetStorageDistribution returns the storage distribution
// for the provided request and decision matrix
func GetStorageDistribution(
	request *cloudops.StorageDistributionRequest,
	decisionMatrix *cloudops.StorageDecisionMatrix,
) (*cloudops.StorageDistributionResponse, error) {
	response := &cloudops.StorageDistributionResponse{}
	for _, userRequest := range request.UserStorageSpec {
		candidate, instancePerZone, size, err := getStorageDistributionCandidate(
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
				DriveCapacityGiB: int64(size),
				DriveType:        candidate.DriveType,
				InstancesPerZone: instancePerZone,
				// TODO: Add distribution logic that takes into
				// account adding multiple drives of the same
				// type
				DriveCount: 1,
			},
		)

	}
	return response, nil
}

func getStorageDistributionCandidate(
	decisionMatrix *cloudops.StorageDecisionMatrix,
	request *cloudops.StorageSpec,
	requestedInstancesPerZone int,
	zoneCount int,
) (*cloudops.StorageDecisionMatrixRow, int, int, error) {

	logRequest(request, requestedInstancesPerZone, zoneCount)

	dm := utils.CopyDecisionMatrix(decisionMatrix)

	// Sort by min capacity
	dm.Rows = utils.SortByMinCapacity(dm.Rows)

	// Start with the requested instances per zone and check if there is any
	// configuration that meets the IOPS and minimum requirement. If no
	// configuration found decrease the instancePerZone by 1 until 1
	for instancesPerZone := requestedInstancesPerZone; instancesPerZone >= 1; instancesPerZone-- {
		// Get the requested total storage nodes
		totalRequestedStorageNodes := zoneCount * instancesPerZone

		// Get storage capacity per node
		storagePerNode := int(request.MinCapacity) / totalRequestedStorageNodes

		// 1st. round of filtering - based of min capacity and IOPS

		// Check if both min drive size and IOPS requirement are met by
		// any configuration in the decision matrix
		candidates := []cloudops.StorageDecisionMatrixRow{}
		for _, row := range dm.Rows {
			if row.IOPS < request.IOPS {
				continue
			}
			if int(row.MinSize) > storagePerNode {
				continue
			}
			// Found a candidate
			candidates = append(candidates, row)
		}
		printCandidates("Candidates", candidates, instancesPerZone, zoneCount)
		if len(candidates) == 0 {
			// No candidates found
			continue
		}
		if len(candidates) == 1 {
			return &candidates[0], instancesPerZone, storagePerNode, nil
		}

		// 2nd round of filtering - based of the candidates which have
		// the closest value to the requested IOPS

		// We have more than one candidate. Choose the candidate which
		// is closest to the requested IOPS
		candidates = utils.SortByClosestIOPS(
			request.IOPS,
			candidates,
		)
		// Remove the candidates which are greater than a factor of 1000
		closestIOPSCandidates := []cloudops.StorageDecisionMatrixRow{}
		for _, candidate := range candidates {
			if candidate.IOPS-request.IOPS < 1000 {
				closestIOPSCandidates = append(
					closestIOPSCandidates,
					candidate,
				)
			} else {
				// All the next candidates are going to have
				// higher IOPS. No need of looping
				break
			}
		}
		printCandidates("Candidates after filtering IOPS cutoff ",
			closestIOPSCandidates, instancesPerZone, zoneCount)
		if len(closestIOPSCandidates) == 0 {
			// Return the candidate which we think is the closest
			return &candidates[0], instancesPerZone, storagePerNode, nil
		}
		if len(closestIOPSCandidates) == 1 {
			return &closestIOPSCandidates[0], instancesPerZone, storagePerNode, nil
		}

		// Choose the candidate which has the highest priority
		if closestIOPSCandidates[0].Priority < closestIOPSCandidates[1].Priority {
			return &closestIOPSCandidates[0], instancesPerZone, storagePerNode, nil
		}
		return &closestIOPSCandidates[1], instancesPerZone, storagePerNode, nil
	}

	return nil, -1, -1, cloudops.ErrStorageDistributionCandidateNotFound
}

func printCandidates(
	msg string,
	candidates []cloudops.StorageDecisionMatrixRow,
	instancePerZone, zoneCount int,
) {
	for _, candidate := range candidates {
		logrus.WithFields(logrus.Fields{
			"IOPS":      candidate.IOPS,
			"MinSize":   candidate.MinSize,
			"DriveType": candidate.DriveType,
			"Priority":  candidate.Priority,
		}).Debugf("%v for %v instances per zone with a total of "+
			"%v zones", msg, instancePerZone, zoneCount)
	}
}

func logRequest(
	request *cloudops.StorageSpec,
	requestedInstancesPerZone int,
	zoneCount int,
) {
	logrus.WithFields(logrus.Fields{
		"IOPS":             request.IOPS,
		"MinCapacity":      request.MinCapacity,
		"InstancesPerZone": requestedInstancesPerZone,
		"ZoneCount":        zoneCount,
	}).Debugf("-- Storage Distribution Pool Request --")
}
