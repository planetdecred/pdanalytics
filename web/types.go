package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/dcrutil/v2"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v2"
	"github.com/go-chi/chi"
)

type Server struct {
	webMux      *chi.Mux
	cfg         Config
	Templates   *Templates
	routes      map[string]route
	routeGroups []routeGroup
	common      CommonPageData
}

// Links to be passed with common page data.
type Links struct {
	CoinbaseComment string
	POSExplanation  string
	APIDocs         string
	InsightAPIDocs  string
	Github          string
	License         string
	NetParams       string
	DownloadLink    string
	// Testnet and below are set via pdanalytics config.
	Testnet       string
	Mainnet       string
	TestnetSearch string
	MainnetSearch string
	OnionURL      string
}

type MenuItem struct {
	Href       string
	HyperText  string
	Info       string
	Attributes map[string]string
}

type BreadcrumbItem struct {
	Href      string
	HyperText string
	Active    bool
}

// Cookies contains information from the request cookies.
type Cookies struct {
	DarkMode bool
}

// TimeDef is time.Time wrapper that formats time by default as a string without
// a timezone. The time Stringer interface formats the time into a string with a
// timezone.
type TimeDef struct {
	T time.Time
}

const (
	timeDefFmtHuman        = "2006-01-02 15:04:05 (MST)"
	timeDefFmtDateTimeNoTZ = "2006-01-02 15:04:05"
	timeDefFmtJS           = time.RFC3339
)

// String formats the time in a human-friendly layout. This ends up on the
// explorer web pages.
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

// MarshalJSON implements json.Marshaler.
func (t *TimeDef) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.RFC3339())
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *TimeDef) UnmarshalJSON(data []byte) error {
	if t == nil {
		return fmt.Errorf("TimeDef: UnmarshalJSON on nil pointer")
	}
	tStr := string(data)
	tStr = strings.Trim(tStr, `"`)
	T, err := time.Parse(timeDefFmtJS, tStr)
	if err != nil {
		return err
	}
	t.T = T
	return nil
}

// TxBasic models data for transactions on the block page
type TxBasic struct {
	TxID          string
	FormattedSize string
	Total         float64
	Fee           dcrutil.Amount
	FeeRate       dcrutil.Amount
	Coinbase      bool
	MixCount      uint32
	MixDenom      int64
}

// TrimmedTxInfo for use with /visualblocks
type TrimmedTxInfo struct {
	*TxBasic
	Fees      float64
	VinCount  int
	VoutCount int
	VoteValid bool
}

type BlockBasic struct {
	Height         int64   `json:"height"`
	Hash           string  `json:"hash"`
	Version        int32   `json:"version"`
	Size           int32   `json:"size"`
	Valid          bool    `json:"valid"`
	MainChain      bool    `json:"mainchain"`
	Voters         uint16  `json:"votes"`
	Transactions   int     `json:"tx"`
	IndexVal       int64   `json:"windowIndex"`
	FreshStake     uint8   `json:"tickets"`
	Revocations    uint32  `json:"revocations"`
	TxCount        uint32  `json:"tx_count"`
	BlockTime      TimeDef `json:"time"`
	FormattedBytes string  `json:"formatted_bytes"`
	Total          float64 `json:"total"`
}

// WebBasicBlock is used for quick DB data without rpc calls
type WebBasicBlock struct {
	Height      uint32   `json:"height"`
	Size        uint32   `json:"size"`
	Hash        string   `json:"hash"`
	Difficulty  float64  `json:"diff"`
	StakeDiff   float64  `json:"sdiff"`
	Time        int64    `json:"time"`
	NumTx       uint32   `json:"txlength"`
	PoolSize    uint32   `json:"poolsize"`
	PoolValue   float64  `json:"poolvalue"`
	PoolValAvg  float64  `json:"poolvalavg"`
	PoolWinners []string `json:"winners"`
}

// BlockInfo models data for display on the block page
type BlockInfo struct {
	*BlockBasic
	Confirmations         int64
	StakeRoot             string
	MerkleRoot            string
	TxAvailable           bool
	Tx                    []*TrimmedTxInfo
	Tickets               []*TrimmedTxInfo
	Revs                  []*TrimmedTxInfo
	Votes                 []*TrimmedTxInfo
	Misses                []string
	Nonce                 uint32
	VoteBits              uint16
	FinalState            string
	PoolSize              uint32
	Bits                  string
	SBits                 float64
	Difficulty            float64
	ExtraData             string
	StakeVersion          uint32
	PreviousHash          string
	NextHash              string
	TotalSent             float64
	MiningFee             float64
	TotalMixed            int64
	StakeValidationHeight int64
	Subsidy               *chainjson.GetBlockSubsidyResult
}

// TicketPoolInfo describes the live ticket pool
type TicketPoolInfo struct {
	Size          uint32  `json:"size"`
	Value         float64 `json:"value"`
	ValAvg        float64 `json:"valavg"`
	Percentage    float64 `json:"percent"`
	Target        uint32  `json:"target"`
	PercentTarget float64 `json:"percent_target"`
}

// ChainParams models simple data about the chain server's parameters used for
// some info on the front page.
type ChainParams struct {
	WindowSize       int64 `json:"window_size"`
	RewardWindowSize int64 `json:"reward_window_size"`
	TargetPoolSize   int64 `json:"target_pool_size"`
	BlockTime        int64 `json:"target_block_time"`
	MeanVotingBlocks int64
}

// BlockSubsidy is an implementation of chainjson.GetBlockSubsidyResult
type BlockSubsidy struct {
	Total int64 `json:"total"`
	PoW   int64 `json:"pow"`
	PoS   int64 `json:"pos"`
	Dev   int64 `json:"dev"`
}

// ExchanageConversion is a representation of some amount of DCR in another index.
type ExchanageConversion struct {
	Value float64 `json:"value"`
	Index string  `json:"index"`
}

// HomeInfo represents data used for the home page
type HomeInfo struct {
	CoinSupply            int64                `json:"coin_supply"`
	StakeDiff             float64              `json:"sdiff"`
	NextExpectedStakeDiff float64              `json:"next_expected_sdiff"`
	NextExpectedBoundsMin float64              `json:"next_expected_min"`
	NextExpectedBoundsMax float64              `json:"next_expected_max"`
	IdxBlockInWindow      int                  `json:"window_idx"`
	IdxInRewardWindow     int                  `json:"reward_idx"`
	Difficulty            float64              `json:"difficulty"`
	DevFund               int64                `json:"dev_fund"`
	DevAddress            string               `json:"dev_address"`
	TicketReward          float64              `json:"reward"`
	RewardPeriod          string               `json:"reward_period"`
	ASR                   float64              `json:"ASR"`
	NBlockSubsidy         BlockSubsidy         `json:"subsidy"`
	Params                ChainParams          `json:"params"`
	PoolInfo              TicketPoolInfo       `json:"pool_info"`
	TotalLockedDCR        float64              `json:"total_locked_dcr"`
	HashRate              float64              `json:"hash_rate"`
	HashRateChangeDay     float64              `json:"hash_rate_change_day"`
	HashRateChangeMonth   float64              `json:"hash_rate_change_month"`
	ExchangeRate          *ExchanageConversion `json:"exchange_rate,omitempty"`
}

type PageData struct {
	sync.RWMutex
	BlockInfo      *BlockInfo
	BlockchainInfo *chainjson.GetBlockChainInfoResult
	HomeInfo       *HomeInfo
}

// CommonPageData is the basis for data structs used for HTML templates.
// explorerUI.commonData returns an initialized instance or CommonPageData,
// which itself should be used to initialize page data template structs.
type CommonPageData struct {
	Tip           *WebBasicBlock
	Version       string
	ChainParams   *chaincfg.Params
	BlockTimeUnix int64
	DevAddress    string
	Links         *Links
	MenuItems     []MenuItem
	NetName       string
	Cookies       Cookies
	RequestURI    string
}
