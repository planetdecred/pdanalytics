// Code generated by SQLBoiler 4.5.0 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import "testing"

func TestUpsert(t *testing.T) {
	t.Run("Blocks", testBlocksUpsert)

	t.Run("BlockBins", testBlockBinsUpsert)

	t.Run("Heartbeats", testHeartbeatsUpsert)

	t.Run("Mempools", testMempoolsUpsert)

	t.Run("MempoolBins", testMempoolBinsUpsert)

	t.Run("NetworkSnapshots", testNetworkSnapshotsUpsert)

	t.Run("NetworkSnapshotBins", testNetworkSnapshotBinsUpsert)

	t.Run("Nodes", testNodesUpsert)

	t.Run("NodeLocations", testNodeLocationsUpsert)

	t.Run("NodeVersions", testNodeVersionsUpsert)

	t.Run("Propagations", testPropagationsUpsert)

	t.Run("Votes", testVotesUpsert)

	t.Run("VoteReceiveTimeDeviations", testVoteReceiveTimeDeviationsUpsert)
}
