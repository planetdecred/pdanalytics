package treasury

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	Txns []*TreasuryTx `json:"txns"`
}

type TreasuryTx struct {
	TxID        string  `json:"TxID"`
	Type        int     `json:"Type"`
	Amount      int64   `json:"Amount"`
	BlockHash   string  `json:"BlockHash"`
	BlockHeight int64   `json:"BlockHeight"`
	BlockTime   TimeDef `json:"BlockTime"`
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
	Transactions    []*TreasuryTx
	NumTransactions int64 // len(Transactions) but int64 for dumb template

	Balance          *dbtypes.TreasuryBalance
	ConvertedBalance *exchanges.Conversion
	TypeCount        int64
	APIURL           string
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

// Format formats the time in the given layout.
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

// PrettyMDY formats the time down to day only, using 3 day month, unpadded day,
// comma, and 4 digit year.
func (t *TimeDef) PrettyMDY() string {
	return t.T.Format("Jan 2, 2006")
}

// HMSTZ is the hour:minute:second with 3-digit timezone code.
func (t *TimeDef) HMSTZ() string {
	return t.T.Format("15:04:05 MST")
}

// DatetimeWithoutTZ formats the time in a human-friendly layout, without
// time zone.
func (t *TimeDef) DatetimeWithoutTZ() string {
	return t.T.Format(timeDefFmtDateTimeNoTZ)
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
