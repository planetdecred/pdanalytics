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
	dataStore       store
	externalDBs []string
	ticketInds      dcrd.BlockValidatorIndex
	syncIsDone      bool
	ticketIndsMutex sync.Mutex

	Version          string
	NetName          string
	MeanVotingBlocks int64

	server *web.Server
}

type store interface {
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

	FetchEncodePropagationChart(ctx context.Context, dataType, axis string,
		binString string, extras ...string) ([]byte, error)
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

type PropagationChartData struct {
	BlockHeight    int64     `json:"block_height"`
	TimeDifference float64   `json:"time_difference"`
	BlockTime      time.Time `json:"block_time"`
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
