// Copyright (c) 2019-2020, The Decred developers
// See LICENSE for details.

package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	piapi "github.com/decred/politeia/politeiawww/api/www/v1"
)

// Politeia votes occur in 2016 block windows.
const windowSize = 2016

// ProposalInfo holds the proposal details as document here
// https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/api.md#user-proposals.
// It also holds the votes status details. The ID field is auto incremented by
// the db. A proposal can now be uniquely identified by the RefID value and the
// the contents on the CensorShipRecord struct.
type ProposalInfo struct {
	ID              int                `json:"id" storm:"id,increment"`
	Name            string             `json:"name"`
	State           ProposalStateType  `json:"state"`
	Status          ProposalStatusType `json:"status"`
	Timestamp       uint64             `json:"timestamp"`
	UserID          string             `json:"userid"`
	Username        string             `json:"username"`
	PublicKey       string             `json:"publickey"`
	Signature       string             `json:"signature"`
	Version         string             `json:"version"`
	NumComments     int32              `json:"numcomments"`
	StatusChangeMsg string             `json:"statuschangemessage"`
	PublishedDate   uint64             `json:"publishedat" storm:"index"`
	CensoredDate    uint64             `json:"censoredat"`
	AbandonedDate   uint64             `json:"abandonedat"`
	// RefID was added to create an easily readable part of the URL that helps
	// to reference the proposals details page. Storm db ignores entries with
	// duplicate pk but returns "ErrAlreadyExists" error if the field other than
	// the pk has the tag "unique".
	RefID string `storm:"unique"`
	// "unique" tag helps to detect when a single proposal instance wants to be
	// pushed to the db as two different instances instead of one. This bug
	// happened due to edits made to a proposal title thus new RefID was created.
	CensorshipRecord `json:"censorshiprecord" storm:"unique"`
	ProposalVotes    `json:"votes"`
	// Files           []AttachmentFile   `json:"files"`
}

// Proposals defines an array of proposals payload as returned by RouteAllVetted route.
type Proposals struct {
	Data []*ProposalInfo `json:"proposals"`
}

// Proposal defines a proposal payload as returned by RouteProposalDetails route.
type Proposal struct {
	Data *ProposalInfo `json:"proposal"`
}

// CensorshipRecord is an entry that was created when the proposal was submitted.
// https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/api.md#censorship-record
type CensorshipRecord struct {
	TokenVal   string `json:"token" storm:"unique"`
	MerkleRoot string `json:"merkle"`
	Signature  string `json:"signature"`
}

// AttachmentFile are files and documents submitted as proposal details.
// https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/api.md#file
type AttachmentFile struct {
	Name      string `json:"name"`
	MimeType  string `json:"mime"`
	DigestKey string `json:"digest"`
	Payload   string `json:"payload"`
}

// ProposalVotes defines the proposal status (Vote info for the public proposals).
// https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/api.md#proposal-vote-status
type ProposalVotes struct {
	Token              string         `json:"token"`
	VoteStatus         VoteStatusType `json:"status"`
	VoteResults        []Results      `json:"optionsresult"`
	TotalVotes         int64          `json:"totalvotes"`
	Endheight          string         `json:"endheight"`
	NumOfEligibleVotes int64          `json:"numofeligiblevotes"`
	QuorumPercentage   uint32         `json:"quorumpercentage"`
	PassPercentage     uint32         `json:"passpercentage"`
}

// Votes defines a slice of VotesStatuses as returned by RouteAllVoteStatus.
type Votes struct {
	Data []ProposalVotes `json:"votesstatus"`
}

// Results defines the actual vote count info per the votes choices available.
type Results struct {
	Option        VoteOption `json:"option"`
	VotesReceived int64      `json:"votesreceived"`
}

// VoteOption defines the actual high level vote results for the specific agenda.
type VoteOption struct {
	OptionID    string `json:"id"`
	Description string `json:"description"`
	Bits        int32  `json:"bits"`
}

// ProposalStatusType defines the various proposal statuses available as referenced
// in https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/v1.go
type ProposalStatusType piapi.PropStatusT

func (p ProposalStatusType) String() string {
	return piapi.PropStatus[piapi.PropStatusT(p)]
}

// VoteStatusType defines the various vote statuses available as referenced in
// https://github.com/decred/politeia/blob/master/politeiawww/api/www/v1/v1.go
type VoteStatusType piapi.PropVoteStatusT

// ShorterDesc maps the short description to there respective vote status type.
var ShorterDesc = map[piapi.PropVoteStatusT]string{
	piapi.PropVoteStatusInvalid:       "Invalid",
	piapi.PropVoteStatusNotAuthorized: "Not Authorized",
	piapi.PropVoteStatusAuthorized:    "Authorized",
	piapi.PropVoteStatusStarted:       "Started",
	piapi.PropVoteStatusFinished:      "Finished",
	piapi.PropVoteStatusDoesntExist:   "Doesn't Exist",
}

// ShortDesc returns the shorter vote status description.
func (s VoteStatusType) ShortDesc() string {
	return ShorterDesc[piapi.PropVoteStatusT(s)]
}

// LongDesc returns the long vote status description.
func (s VoteStatusType) LongDesc() string {
	return piapi.PropVoteStatus[piapi.PropVoteStatusT(s)]
}

// ProposalStateType defines the proposal state entry.
type ProposalStateType int8

const (
	// InvalidState defines the invalid state proposals.
	InvalidState ProposalStateType = iota

	// UnvettedState defines the unvetted state proposals and includes proposals
	// with a status of:
	//   * PropStatusNotReviewed
	//   * PropStatusUnreviewedChanges
	//   * PropStatusCensored
	UnvettedState

	// VettedState defines the vetted state proposals and includes proposals
	// with a status of:
	//   * PropStatusPublic
	//   * PropStatusAbandoned
	VettedState

	// UnknownState indicates a proposal state that is unrecognized.
	UnknownState
)

func (f ProposalStateType) String() string {
	switch f {
	case InvalidState:
		return "invalid"
	case UnvettedState:
		return "unvetted"
	case VettedState:
		return "vetted"
	default:
		return "unknown"
	}
}

// ProposalStateFromStr converts the string into ProposalStateType value.
func ProposalStateFromStr(val string) ProposalStateType {
	switch val {
	case "invalid":
		return InvalidState
	case "unvetted":
		return UnvettedState
	case "vetted":
		return VettedState
	default:
		return UnknownState
	}
}

// VotesStatuses returns the ShorterDesc map contents exclusive of Invalid and
// Doesn't exist statuses.
func VotesStatuses() map[VoteStatusType]string {
	m := make(map[VoteStatusType]string)
	for k, val := range ShorterDesc {
		if k == piapi.PropVoteStatusInvalid ||
			k == piapi.PropVoteStatusDoesntExist {
			continue
		}
		m[VoteStatusType(k)] = val
	}
	return m
}

// IsEqual compares CensorshipRecord, Name, State, NumComments, StatusChangeMsg,
// Timestamp, CensoredDate, AbandonedDate, PublishedDate, Token, VoteStatus,
// TotalVotes and count of VoteResults between the two ProposalsInfo structs passed.
func (pi *ProposalInfo) IsEqual(b *ProposalInfo) bool {
	if pi.CensorshipRecord != b.CensorshipRecord || pi.Name != b.Name || pi.State != b.State ||
		pi.NumComments != b.NumComments || pi.StatusChangeMsg != b.StatusChangeMsg ||
		pi.Status != b.Status || pi.Timestamp != b.Timestamp || pi.Token != b.Token ||
		pi.CensoredDate != b.CensoredDate || pi.AbandonedDate != b.AbandonedDate ||
		pi.VoteStatus != b.VoteStatus || pi.TotalVotes != b.TotalVotes ||
		pi.PublishedDate != b.PublishedDate || len(pi.VoteResults) != len(b.VoteResults) {
		return false
	}
	return true
}

// ProposalMetadata contains some status-dependent data representations for
// display purposes.
type ProposalMetadata struct {
	// The start height of the vote. The end height is already part of the
	// ProposalInfo struct.
	StartHeight int64
	// Time until start for "Authorized" proposals, Time until done for "Started"
	// proposals.
	SecondsTil     int64
	IsPassing      bool
	Approval       float32
	Rejection      float32
	Yes            int64
	No             int64
	VoteCount      int64
	QuorumCount    int64
	QuorumAchieved bool
	PassPercent    float32
}

// Metadata performs some common manipulations of the ProposalInfo data to
// prepare figures for display. Many of these manipulations require a tip height
// and a target block time for the network, so those must be provided as
// arguments.
func (pi *ProposalInfo) Metadata(tip, targetBlockTime int64) *ProposalMetadata {
	meta := new(ProposalMetadata)
	desc := pi.VoteStatus.ShortDesc()
	switch desc {
	case "Started", "Finished":
		endHeight, _ := strconv.ParseInt(pi.ProposalVotes.Endheight, 10, 64)
		meta.StartHeight = endHeight - windowSize
		for _, count := range pi.VoteResults {
			switch count.Option.OptionID {
			case "yes":
				meta.Yes = count.VotesReceived
			case "no":
				meta.No = count.VotesReceived
			}
		}
		meta.VoteCount = meta.Yes + meta.No
		quorumPct := float32(pi.QuorumPercentage) / 100
		meta.QuorumCount = int64(quorumPct * float32(pi.NumOfEligibleVotes))
		meta.PassPercent = float32(pi.PassPercentage) / 100
		pctVoted := float32(meta.VoteCount) / float32(pi.NumOfEligibleVotes)
		meta.QuorumAchieved = pctVoted > quorumPct
		if meta.VoteCount > 0 {
			meta.Approval = float32(meta.Yes) / float32(meta.VoteCount)
			meta.Rejection = 1 - meta.Approval
		}
		meta.IsPassing = meta.Approval > meta.PassPercent
		if desc == "Started" {
			blocksLeft := endHeight - tip
			meta.SecondsTil = blocksLeft * targetBlockTime
		}
	}
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
