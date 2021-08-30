// Copyright (c) 2019-2020, The Decred developers
// See LICENSE for details.

package types

import (
	recordsv1 "github.com/decred/politeia/politeiawww/api/records/v1"
	ticketvotev1 "github.com/decred/politeia/politeiawww/api/ticketvote/v1"
)

// ProposalRecord is the struct that holds all politeia data that dcrdata needs
// for each proposal. This is the object that is saved to stormdb. It uses data
// from three politeia API's: records, comments and ticketvote.
type ProposalRecord struct {
	ID int `json:"id" storm:"id,increment"`

	// Record API data
	State     recordsv1.RecordStateT  `json:"state"`
	Status    recordsv1.RecordStatusT `json:"status"`
	Token     string                  `json:"token"`
	Version   uint32                  `json:"version"`
	Timestamp uint64                  `json:"timestamp" storm:"index"`
	Username  string                  `json:"username"`

	// Pi metadata
	Name string `json:"name"`

	// User metadata
	UserID string `json:"userid"`

	// Comments API data
	CommentsCount int32 `json:"commentscount"`

	// Ticketvote API data
	VoteStatus       ticketvotev1.VoteStatusT  `json:"votestatus"`
	VoteResults      []ticketvotev1.VoteResult `json:"voteresults"`
	StatusChangeMsg  string                    `json:"statuschangemsg"`
	EligibleTickets  uint32                    `json:"eligibletickets"`
	StartBlockHeight uint32                    `json:"startblockheight"`
	EndBlockHeight   uint32                    `json:"endblockheight"`
	QuorumPercentage uint32                    `json:"quorumpercentage"`
	PassPercentage   uint32                    `json:"passpercentage"`
	TotalVotes       uint64                    `json:"totalvotes"`
	ChartData        *ProposalChartData        `json:"chartdata"`

	// Synced is used to indicate that this proposal is already fully
	// synced with politeia server, and does not need to make any more
	// http requests for this proposal
	Synced bool `json:"synced"`

	// Timestamps
	PublishedAt uint64 `json:"publishedat" storm:"index"`
	CensoredAt  uint64 `json:"censoredat"`
	AbandonedAt uint64 `json:"abandonedat"`
}

// ProposalChartData defines the data used to plot proposal ticket votes
// charts.
type ProposalChartData struct {
	Yes  []uint64 `json:"yes"`
	No   []uint64 `json:"no"`
	Time []int64  `json:"time"`
}

// IsEqual compares data between the two ProposalRecord structs passed.
func (pi *ProposalRecord) IsEqual(b ProposalRecord) bool {
	if pi.Token != b.Token || pi.Name != b.Name || pi.State != b.State ||
		pi.Status != b.Status || pi.StatusChangeMsg != b.StatusChangeMsg ||
		pi.CommentsCount != b.CommentsCount || pi.Timestamp != b.Timestamp ||
		pi.VoteStatus != b.VoteStatus || pi.TotalVotes != b.TotalVotes ||
		pi.PublishedAt != b.PublishedAt || pi.CensoredAt != b.CensoredAt ||
		pi.AbandonedAt != b.AbandonedAt || pi.ChartData != b.ChartData {
		return false
	}
	return true
}

// ProposalMetadata contains some status-dependent data representations for
// display purposes.
type ProposalMetadata struct {
	// Time until start for "Authorized" proposals, Time until done for
	// "Started" proposals.
	SecondsTil         int64
	IsPassing          bool
	Approval           float32
	Rejection          float32
	Yes                int64
	No                 int64
	VoteCount          int64
	QuorumCount        int64
	QuorumAchieved     bool
	PassPercent        float32
	VoteStatusDesc     string
	ProposalStateDesc  string
	ProposalStatusDesc string
}

// Metadata performs some common manipulations of the ProposalRecord data to
// prepare figures for display. Many of these manipulations require a tip
// height and a target block time for the network, so those must be provided
// as arguments.
func (pi *ProposalRecord) Metadata(tip, targetBlockTime int64) *ProposalMetadata {
	meta := new(ProposalMetadata)
	switch pi.VoteStatus {
	case ticketvotev1.VoteStatusStarted, ticketvotev1.VoteStatusFinished,
		ticketvotev1.VoteStatusApproved, ticketvotev1.VoteStatusRejected:
		for _, count := range pi.VoteResults {
			switch count.ID {
			case "yes":
				meta.Yes = int64(count.Votes)
			case "no":
				meta.No = int64(count.Votes)
			}
		}
		meta.VoteCount = meta.Yes + meta.No
		quorumPct := float32(pi.QuorumPercentage) / 100
		meta.QuorumCount = int64(quorumPct * float32(pi.EligibleTickets))
		meta.PassPercent = float32(pi.PassPercentage) / 100
		pctVoted := float32(meta.VoteCount) / float32(pi.EligibleTickets)
		meta.QuorumAchieved = pctVoted > quorumPct
		if meta.VoteCount > 0 {
			meta.Approval = float32(meta.Yes) / float32(meta.VoteCount)
			meta.Rejection = 1 - meta.Approval
		}
		meta.IsPassing = meta.Approval > meta.PassPercent
		if pi.VoteStatus == ticketvotev1.VoteStatusStarted {
			blocksLeft := int64(pi.EndBlockHeight) - tip
			meta.SecondsTil = blocksLeft * targetBlockTime
		}
	}
	meta.VoteStatusDesc = ticketvotev1.VoteStatuses[pi.VoteStatus]
	meta.ProposalStateDesc = recordsv1.RecordStates[pi.State]
	meta.ProposalStatusDesc = recordsv1.RecordStatuses[pi.Status]
	return meta
}

// TimeDef is time.Time wrapper that formats time by default as a string without
// a timezone. The time Stringer interface formats the time into a string
// with a timezone.
type TimeDef struct {
	T time.Time
}

const (
	timeDefFmtHuman        = "2006-01-02 15:04:05 (MST)"
	timeDefFmtDateTimeNoTZ = "2006-01-02 15:04:05"
	timeDefFmtJS           = time.RFC3339
)

// String formats the time in a human-friendly layout. This may be used when
// TimeDef values end up on the explorer pages.
func (t TimeDef) String() string {
	return t.T.Format(timeDefFmtHuman)
}

// RFC3339 formats the time in a machine-friendly layout.
func (t TimeDef) RFC3339() string {
	return t.T.Format(timeDefFmtJS)
}

// UNIX returns the UNIX epoch time stamp.
func (t TimeDef) UNIX() int64 {
	return t.T.Unix()
}

func (t TimeDef) Format(layout string) string {
	return t.T.Format(layout)
}

// DatetimeWithoutTZ formats the time in a human-friendly layout, without
// time zone.
func (t *TimeDef) DatetimeWithoutTZ() string {
	return t.T.Format(timeDefFmtDateTimeNoTZ)
}

// MarshalJSON is set as the default marshalling function for TimeDef struct.
func (t *TimeDef) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.RFC3339())
}

// NewTimeDef constructs a TimeDef from the given time.Time. It presets the
// timezone for formatting to UTC.
func NewTimeDef(t time.Time) TimeDef {
	return TimeDef{
		T: t.UTC(),
	}
}

// NewTimeDefFromUNIX constructs a TimeDef from the given UNIX epoch time stamp
// in seconds. It presets the timezone for formatting to UTC.
func NewTimeDefFromUNIX(t int64) TimeDef {
	return NewTimeDef(time.Unix(t, 0))
}

// Scan implements the sql.Scanner interface for TimeDef. This will not
// reinterpret the stored time string for a particular time zone. That is, if
// the stored time stamp shows no time zone (as with TIMESTAMP), the default
// time.Time scanner will load it as a local time, and this Scan converts to
// UTC. If the timestamp has a timezone (as with TIMESTAMPTZ), including UTC
// explicitly set, it will be accounted for when converting to UTC. All this
// Scan implementation does beyond the default time.Time scanner is to set the
// time.Time's location to UTC, which keeps the instant in time the same,
// adjusting the numbers in the time string to the equivalent time in UTC. For
// example, if the time read from the DB is "2016-02-08 12:00:00" (with no time
// zone) and the server time zone is CST (UTC-6), this ensures the default
// displayed time string is in UTC: "2016-02-08 18:00:00Z". On the other hand,
// if the time read from the DB is "2016-02-08 12:00:00+6", it does not matter
// what the server time zone is set to, and the time will still be converted to
// UTC as "2016-02-08 18:00:00Z".
func (t *TimeDef) Scan(src interface{}) error {
	srcTime, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("scanned value not a time.Time")
	}
	// Debug:
	// fmt.Printf("srcTime: %v, location: %p\n", srcTime, srcTime.Location()) // valid location not set!

	// Set location to UTC. This does not shift the UNIX epoch time.
	t.T = srcTime.UTC()

	// Debug:
	// fmt.Printf("t: %v, t.T: %v, location: %s\n", t, t.T, t.T.Location().String())
	return nil
}

// Value implements the sql.Valuer interface. It ensures that the Time Values
// are for the UTC time zone. Times will only survive a round trip to and from
// the DB tables if they are stored from a time.Time with Location set to UTC.
func (t TimeDef) Value() (driver.Value, error) {
	return t.T.UTC(), nil
}

// Ensure TimeDef satisfies sql.Valuer.
var _ driver.Valuer = (*TimeDef)(nil)

// ProposalChartsData defines the data used to plot proposal votes charts.
type ProposalChartsData struct {
	Yes  []uint64  `json:"yes,omitempty"`
	No   []uint64  `json:"no,omitempty"`
	Time []TimeDef `json:"time,omitempty"`
}
