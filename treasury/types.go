package treasury

import (
	"github.com/decred/dcrd/blockchain/stake/v4"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/decred/dcrdata/v7/db/dbtypes"
)

// TxParams models the treasury transactions post data structure.
type TxParams struct {
	Limit  int64        `json:"limit"`
	Offset int64        `json:"offset"`
	TxType stake.TxType `json:"txtype"`
}

//  Txns models treasury transactions
type Txns struct {
	Txns []*dbtypes.TreasuryTx `json:"txns"`
}

// Balance models treasury balance.
type Balance struct {
	Balance *dbtypes.TreasuryBalance `json:"balance"`
}

// A page number has the information necessary to create numbered pagination
// links.
type PageNumber struct {
	Active bool   `json:"active"`
	Link   string `json:"link"`
	Str    string `json:"str"`
}

type PageNumbers []PageNumber

type TreasuryInfo struct {
	Net string

	// Page parameters
	MaxTxLimit    int64
	Path          string
	Limit, Offset int64  // ?n=Limit&start=Offset
	TxnType       string // ?txntype=TxnType

	// TODO: tadd and tspend can be unconfirmed. tspend for a very long time.
	// NumUnconfirmed is the number of unconfirmed txns
	// NumUnconfirmed  int64
	// UnconfirmedTxns []*dbtypes.TreasuryTx

	// Transactions on the current page
	Transactions    []*dbtypes.TreasuryTx
	NumTransactions int64 // len(Transactions) but int64 for dumb template

	Balance          *dbtypes.TreasuryBalance
	ConvertedBalance *exchanges.Conversion
	TypeCount        int64
}
