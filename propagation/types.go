package propagation

import (
	"context"
	"sync"
	"time"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type propagation struct {
	ctx             context.Context
	client          *dcrd.Dcrd
	dataStore       Store
	externalDBs     map[string]Store
	externalDBNames []string
	ticketInds      dcrd.BlockValidatorIndex
	syncIsDone      bool
	ticketIndsMutex sync.Mutex

	Version          string
	NetName          string
	MeanVotingBlocks int64

	server *web.Server
}

type Store interface {
	BlockTableName() string
	VoteTableName() string
	SaveBlock(context.Context, Block) error
	UpdateBlockBinData(context.Context) error
	SaveVote(ctx context.Context, vote Vote) error
	UpdateVoteTimeDeviationData(context.Context) error

	BlockCount(ctx context.Context) (int64, error)
	Blocks(ctx context.Context, offset int, limit int) ([]BlockDto, error)
	BlocksWithoutVotes(ctx context.Context, offset int, limit int) ([]BlockDto, error)

	Votes(ctx context.Context, offset int, limit int) ([]VoteDto, error)
	VotesByBlock(ctx context.Context, blockHash string) ([]VoteDto, error)
	VotesCount(ctx context.Context) (int64, error)

	BlockDelays(ctx context.Context, height int) ([]PropagationChartData, error)
	SourceDeviations(ctx context.Context, source, bin string) ([]SourceDeviation, error)
	BlockBinData(ctx context.Context, bin string) ([]BlockBinDto, error)
	VotesBlockReceiveTimeDiffs(ctx context.Context) ([]PropagationChartData, error)
	VoteReceiveTimeDeviations(ctx context.Context, bin string) ([]VoteReceiveTimeDeviation, error)
}

type Dto struct {
	Time                 string  `json:"time"`
	FirstSeenTime        string  `json:"first_seen_time"`
	NumberOfTransactions int     `json:"number_of_transactions"`
	Voters               int     `json:"voters"`
	Tickets              int     `json:"tickets"`
	Revocations          int     `json:"revocations"`
	Size                 int32   `json:"size"`
	TotalFee             float64 `json:"total_fee"`
	Total                float64 `json:"total"`
}

type Block struct {
	BlockReceiveTime  time.Time
	BlockInternalTime time.Time
	BlockHeight       uint32
	BlockHash         string
}

type BlockDto struct {
	BlockReceiveTime  string    `json:"block_receive_time"`
	BlockInternalTime string    `json:"block_internal_time"`
	Delay             string    `json:"delay"`
	BlockHeight       uint32    `json:"block_height"`
	BlockHash         string    `json:"block_hash"`
	Votes             []VoteDto `json:"votes"`
}

// BlockBin is an object representing the database table.
type BlockBinDto struct {
	Height            int64   `json:"height" toml:"height" yaml:"height"`
	ReceiveTimeDiff   float64 `json:"receive_time_diff" toml:"receive_time_diff" yaml:"receive_time_diff"`
	InternalTimestamp int64   `json:"internal_timestamp" toml:"internal_timestamp" yaml:"internal_timestamp"`
}

type PropagationChartData struct {
	BlockHeight    int64     `json:"block_height"`
	TimeDifference float64   `json:"time_difference"`
	BlockTime      time.Time `json:"block_time"`
}

// SourceDeviation give the difference in block receive time of this instance and an external source
type SourceDeviation struct {
	Height    int64   `json:"height" toml:"height" yaml:"height"`
	Time      int64   `json:"time" toml:"time" yaml:"time"`
	Deviation float64 `json:"deviation" toml:"deviation" yaml:"deviation"`
}

type BlockReceiveTime struct {
	BlockHeight int64 `json:"block_height"`
	ReceiveTime time.Time
}

type Vote struct {
	Hash              string
	ReceiveTime       time.Time
	TargetedBlockTime time.Time
	BlockReceiveTime  time.Time
	VotingOn          int64
	BlockHash         string
	ValidatorId       int
	Validity          string
}

// VoteReceiveTimeDeviation is used to keep track of the block/vote receive time
type VoteReceiveTimeDeviation struct {
	BlockHeight           int64   `json:"block_height" toml:"block_height" yaml:"block_height"`
	BlockTime             int64   `json:"block_time" toml:"block_time" yaml:"block_time"`
	ReceiveTimeDifference float64 `json:"receive_time_difference" toml:"receive_time_difference" yaml:"receive_time_difference"`
}

type VoteDto struct {
	Hash                  string `json:"hash"`
	ReceiveTime           string `json:"receive_time"`
	TargetedBlockTimeDiff string `json:"block_time_diff"`
	BlockReceiveTimeDiff  string `json:"block_receive_time_diff"`
	VotingOn              int64  `json:"voting_on"`
	BlockHash             string `json:"block_hash"`
	ShortBlockHash        string `json:"short_block_hash"`
	ValidatorId           int    `json:"validator_id"`
	Validity              string `json:"validity"`
}
