package stakingreward

import (
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

// New creates a new Calculator module
func New(client *dcrd.Dcrd, webServer *web.Server, xcBot *exchanges.ExchangeBot) (*Calculator, error) {
	calc := &Calculator{
		webServer: webServer,
		xcBot:     xcBot,
		client:    client,
	}

	calc.MeanVotingBlocks = CalcMeanVotingBlocks(client.Params)

	hash, err := client.Rpc.GetBestBlockHash()
	if err != nil {
		return nil, err
	}

	blockHeader, err := client.Rpc.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	if err = calc.ConnectBlock(blockHeader); err != nil {
		return nil, err
	}

	calc.client.Notif.RegisterBlockHandlerGroup(calc.ConnectBlock)

	calc.webServer.AddMenuItem(web.MenuItem{
		Href:      "/stakingcalc",
		HyperText: "Staking Calc",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Staking Reward Calculator",
		},
	})

	err = webServer.Templates.AddTemplate("stakingreward")
	if err != nil {
		return nil, err
	}

	webServer.AddRoute("/stakingcalc", web.GET, calc.stakingReward)
	webServer.AddRoute("/stakingcalc/get-future-reward", web.GET, calc.targetTicketReward)

	return calc, nil
}

// ConnectBlock satisfies block notifier interface
func (calc *Calculator) ConnectBlock(w *wire.BlockHeader) error {
	calc.reorgLock.Lock()
	defer calc.reorgLock.Unlock()

	calc.Height = w.Height

	// Stake difficulty (ticket price)
	stakeDiff, err := calc.client.Rpc.GetStakeDifficulty()
	if err != nil {
		return err
	}
	calc.TicketPrice = stakeDiff.CurrentStakeDifficulty

	nbSubsidy, err := calc.client.Rpc.GetBlockSubsidy(int64(w.Height)+1, 5)
	if err != nil {
		log.Errorf("GetBlockSubsidy for %d failed: %v", w.Height, err)
	}

	posSubsPerVote := dcrutil.Amount(nbSubsidy.PoS).ToCoin() /
		float64(calc.client.Params.TicketsPerBlock)
	calc.TicketReward = 100 * posSubsPerVote / stakeDiff.CurrentStakeDifficulty

	// The actual reward of a ticket needs to also take into consideration the
	// ticket maturity (time from ticket purchase until its eligible to vote)
	// and coinbase maturity (time after vote until funds distributed to ticket
	// holder are available to use).
	avgSSTxToSSGenMaturity := calc.MeanVotingBlocks +
		int64(calc.client.Params.TicketMaturity) +
		int64(calc.client.Params.CoinbaseMaturity)
	calc.RewardPeriod = float64(avgSSTxToSSGenMaturity) *
		calc.client.Params.TargetTimePerBlock.Hours() / 24

	// Coin supply
	coinSupply, err := calc.client.Rpc.GetCoinSupply()
	if err != nil {
		return err
	}
	calc.coinSupply = coinSupply.ToCoin()

	// Ticket pool info
	poolValue, err := calc.client.Rpc.GetTicketPoolValue()
	if err != nil {
		return err
	}
	calc.stakePerc = poolValue.ToCoin() / coinSupply.ToCoin()

	return nil
}

// CalcMeanVotingBlocks computes the average number of blocks a ticket will be
// live before voting. The expected block (aka mean) of the probability
// distribution is given by:
//      sum(B * P(B)), B=1 to 40960
// Where B is the block number and P(B) is the probability of voting at
// block B.  For more information see:
// https://github.com/decred/dcrdata/issues/471#issuecomment-390063025
func CalcMeanVotingBlocks(params *chaincfg.Params) int64 {
	logPoolSizeM1 := math.Log(float64(params.TicketPoolSize) - 1)
	logPoolSize := math.Log(float64(params.TicketPoolSize))
	var v float64
	for i := float64(1); i <= float64(params.TicketExpiry); i++ {
		v += math.Exp(math.Log(i) + (i-1)*logPoolSizeM1 - i*logPoolSize)
	}
	return int64(v)
}

// Simulate ticket purchase and re-investment over a full year for a given
// starting amount of DCR and calculation parameters.  Generate a TEXT table of
// the simulation results that can optionally be used for future expansion of
// dcrdata functionality.
func (calc *Calculator) simulateStakingReward(numberOfDays float64, startingDCRBalance float64, integerTicketQty bool,
	currentStakePercent float64, actualCoinbase float64, startingBlockHeight float64,
	actualTicketPrice float64) (stakingReward, ticketPrice float64, simulationTable []simulationRow) {

	blocksPerDay := 86400 / calc.client.Params.TargetTimePerBlock.Seconds()
	numberOfBlocks := numberOfDays * blocksPerDay
	ticketsPurchased := float64(0)

	StakeRewardAtBlock := func(blocknum float64) float64 {
		// Option 1:  RPC Call

		Subsidy, _ := calc.client.Rpc.GetBlockSubsidy(int64(blocknum), 1)
		return dcrutil.Amount(Subsidy.PoS).ToCoin()

		// Option 2:  Calculation
		// epoch := math.Floor(blocknum / float64(exp.ChainParams.SubsidyReductionInterval))
		// RewardProportionPerVote := float64(exp.ChainParams.StakeRewardProportion) / (10 * float64(exp.ChainParams.TicketsPerBlock))
		// return RewardProportionPerVote * dcrutil.Amount(exp.ChainParams.BaseSubsidy).ToCoin() *
		// 	math.Pow(float64(exp.ChainParams.MulSubsidy)/float64(exp.ChainParams.DivSubsidy), epoch)
	}

	MaxCoinSupplyAtBlock := func(blocknum float64) float64 {
		// 4th order poly best fit curve to Decred mainnet emissions plot.
		// Curve fit was done with 0 Y intercept and Pre-Mine added after.

		return (-9e-19*math.Pow(blocknum, 4) +
			7e-12*math.Pow(blocknum, 3) -
			2e-05*math.Pow(blocknum, 2) +
			29.757*blocknum + 76963 +
			1680000) // Premine 1.68M
	}

	CoinAdjustmentFactor := actualCoinbase / MaxCoinSupplyAtBlock(startingBlockHeight)

	TheoreticalTicketPrice := func(blocknum float64) float64 {
		ProjectedCoinsCirculating := MaxCoinSupplyAtBlock(blocknum) * CoinAdjustmentFactor * currentStakePercent
		TicketPoolSize := (float64(calc.MeanVotingBlocks) + float64(calc.client.Params.TicketMaturity) +
			float64(calc.client.Params.CoinbaseMaturity)) * float64(calc.client.Params.TicketsPerBlock)

		return ProjectedCoinsCirculating / TicketPoolSize
	}
	ticketPrice = TheoreticalTicketPrice(startingBlockHeight)
	TicketAdjustmentFactor := actualTicketPrice / TheoreticalTicketPrice(float64(calc.Height))

	// Prepare for simulation
	simblock := startingBlockHeight
	var TicketPrice float64
	DCRBalance := startingDCRBalance

	simulationTable = append(simulationTable, simulationRow{
		SimBlock:    simblock,
		SimDay:      0,
		DCRBalance:  DCRBalance,
		TicketPrice: ticketPrice,
	})

	for simblock < (numberOfBlocks + startingBlockHeight) {
		// Simulate a Purchase on simblock
		TicketPrice = TheoreticalTicketPrice(simblock) * TicketAdjustmentFactor
		if integerTicketQty {
			// Use this to simulate integer qtys of tickets up to max funds
			ticketsPurchased = math.Floor(DCRBalance / TicketPrice)
		} else {
			// Use this to simulate ALL funds used to buy tickets - even fractional tickets
			// which is actually not possible
			ticketsPurchased = (DCRBalance / TicketPrice)
		}

		simulationTable[len(simulationTable)-1].TicketPrice = TicketPrice
		simulationTable[len(simulationTable)-1].TicketsPurchased = ticketsPurchased

		DCRBalance -= (TicketPrice * ticketsPurchased)

		// Move forward to average vote
		simblock += (float64(calc.client.Params.TicketMaturity) + float64(calc.MeanVotingBlocks))

		// Simulate return of funds
		DCRBalance += (TicketPrice * ticketsPurchased)

		// Simulate reward
		DCRBalance += (StakeRewardAtBlock(simblock) * ticketsPurchased)

		blocksPassed := simblock - simulationTable[len(simulationTable)-1].SimBlock
		daysPassed := blocksPassed / blocksPerDay
		day := simulationTable[len(simulationTable)-1].SimDay + int(daysPassed)
		simulationTable = append(simulationTable, simulationRow{
			SimBlock:     simblock,
			SimDay:       day,
			DCRBalance:   DCRBalance,
			Reward:       (StakeRewardAtBlock(simblock) * ticketsPurchased),
			ReturnedFund: (TicketPrice * ticketsPurchased),
			TicketPrice:  TheoreticalTicketPrice(simblock) * TicketAdjustmentFactor,
		})

		// Move forward to coinbase maturity
		simblock += float64(calc.client.Params.CoinbaseMaturity)

		// Need to receive funds before we can use them again so add 1 block
		simblock++
	}

	// Scale down to exactly numberOfDays days
	SimulationReward := ((DCRBalance - startingDCRBalance) / startingDCRBalance) * 100
	excessBlocks := (simblock - startingBlockHeight)
	stakingReward = (numberOfBlocks / excessBlocks) * SimulationReward
	overflow := startingDCRBalance * (SimulationReward - stakingReward) / 100
	simulationTable[len(simulationTable)-1].DCRBalance -= overflow
	simulationTable[len(simulationTable)-1].SimDay -= int(excessBlocks / blocksPerDay)
	simulationTable[len(simulationTable)-1].Reward -= overflow
	// remove nagative rewards from the table
	for i := len(simulationTable) - 1; i > 0; i-- {
		if simulationTable[i].Reward >= 0 {
			break
		}
		simulationTable[i-1].Reward += simulationTable[i].Reward
		simulationTable[i].Reward = 0
	}
	return
}

// stakingReward is the page handler for the "/ticket-reward" path.
func (calc *Calculator) stakingReward(w http.ResponseWriter, r *http.Request) {
	price := 24.42 // why this value?
	if calc.xcBot != nil {
		if rate := calc.xcBot.Conversion(1.0); rate != nil {
			price = rate.Value
		}
	}

	calc.reorgLock.Lock()

	str, err := calc.webServer.Templates.ExecTemplateToString("stakingreward", struct {
		*web.CommonPageData
		TicketPrice  float64
		RewardPeriod float64
		TicketReward float64
		DCRPrice     float64
	}{
		CommonPageData: calc.webServer.CommonData(r),
		TicketPrice:    calc.TicketPrice,
		RewardPeriod:   calc.RewardPeriod,
		TicketReward:   calc.TicketReward,
		DCRPrice:       price,
	})
	calc.reorgLock.Unlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		calc.webServer.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (calc *Calculator) targetTicketReward(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	startingBalance, err := strconv.ParseFloat(r.FormValue("startingBalance"), 64)
	if err != nil {
		log.Errorf("Read date failed: %v", err)
		web.RenderErrorfJSON(w, "Error in reading starting balance, %v", err)
		return
	}
	startDateUnix, err := strconv.ParseInt(r.FormValue("startDate"), 10, 64)
	if err != nil {
		log.Errorf("Read date failed: %v", err)
		web.RenderErrorfJSON(w, "Invalid timestamp suplied, %v", err)
		return
	}
	endDateUnix, err := strconv.ParseInt(r.FormValue("endDate"), 10, 64)
	if err != nil {
		log.Errorf("Read end date failed: %v", err)
		web.RenderErrorfJSON(w, "Invalid timestamp suplied, %v", err)
		return
	}

	startDate := time.Unix(startDateUnix/1000, 0).UTC().Truncate(24 * time.Hour)
	endDate := time.Unix(endDateUnix/1000, 0).UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// starting startingHeight
	var startingHeight = calc.Height

	if startDate != today {
		duration := startDate.Sub(today)
		minutes := duration.Minutes() + duration.Hours()*60
		minutesPerBlock := calc.client.Params.TargetTimePerBlock.Minutes() + calc.client.Params.TargetTimePerBlock.Hours()*60
		blockDiff := minutes / minutesPerBlock
		blockDiff = math.Abs(blockDiff)
		if startDate.Before(today) {
			startingHeight = calc.Height - uint32(blockDiff)
		} else {
			startingHeight = calc.Height + uint32(blockDiff)
		}
	}

	// accumulated staking reward
	asr, ticketPrice, simulationTable := calc.simulateStakingReward((endDate.Sub(startDate)).Hours()/24, startingBalance, true,
		calc.stakePerc, calc.coinSupply, float64(startingHeight), calc.TicketPrice)

	web.RenderJSON(w, struct {
		Height          uint32          `json:"height"`
		Reward          float64         `json:"reward"`
		TicketPrice     float64         `json:"ticketPrice"`
		SimulationTable []simulationRow `json:"simulation_table"`
	}{
		Height:          startingHeight,
		Reward:          asr,
		TicketPrice:     ticketPrice,
		SimulationTable: simulationTable,
	})
}
