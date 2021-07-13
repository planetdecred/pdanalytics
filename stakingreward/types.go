package stakingreward

import (
	"sync"

	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Calculator struct {
	webServer *web.Server
	xcBot     *exchanges.ExchangeBot
	client    *dcrd.Dcrd

	Height       uint32
	stakePerc    float64
	coinSupply   float64
	TicketPrice  float64
	TicketReward float64
	RewardPeriod float64

	MeanVotingBlocks int64

	reorgLock sync.Mutex
}

type simulationRow struct {
	SimBlock         float64 `json:"height"`
	SimDay           int     `json:"day"`
	TicketPrice      float64 `json:"ticket_price"`
	MatrueTickets    float64 `json:"matured_tickets"`
	DCRBalance       float64 `json:"dcr_balance"`
	TicketsPurchased float64 `json:"tickets_purchased"`
	Reward           float64 `json:"reward"`
	LockedFund     float64 `json:"locked_fund"`
}
