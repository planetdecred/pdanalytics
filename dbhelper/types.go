package dbhelper

// AgendaSummary describes a short summary of a given agenda that includes
// vote choices tally and deployment rule change intervals.
type AgendaSummary struct {
	Yes           uint32
	No            uint32
	Abstain       uint32
	VotingStarted int64
	LockedIn      int64
}
