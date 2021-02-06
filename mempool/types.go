package mempool

import (
	"context"
	"time"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Mempool struct {
	Time                 time.Time `json:"time"`
	FirstSeenTime        time.Time `json:"first_seen_time"`
	NumberOfTransactions int       `json:"number_of_transactions"`
	Voters               int       `json:"voters"`
	Tickets              int       `json:"tickets"`
	Revocations          int       `json:"revocations"`
	Size                 int32     `json:"size"`
	TotalFee             float64   `json:"total_fee"`
	Total                float64   `json:"total"`
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

type DataStore interface {
	CreateTables(ctx context.Context) error
	DropTables() error
	MempoolTableName() string
	StoreMempool(context.Context, Mempool) error
	UpdateMempoolAggregateData(ctx context.Context) error
	LastMempoolTime() (entryTime time.Time, err error)
	LastMempoolBlockHeight() (height int64, err error)
	MempoolCount(ctx context.Context) (int64, error)
	Mempools(ctx context.Context, offtset int, limit int) ([]Dto, error)
	FetchEncodeChart(ctx context.Context, dataType, binString string) ([]byte, error)
}

type Collector struct {
	ctx                context.Context
	collectionInterval float64
	client             *dcrd.Dcrd
	dataStore          DataStore

	webServer *web.Server

	Version          string
	NetName          string
	MeanVotingBlocks int64
}
