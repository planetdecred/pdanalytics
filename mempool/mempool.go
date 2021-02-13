package mempool

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	dcrjson "github.com/decred/dcrd/rpc/jsonrpc/types/v2"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

func NewCollector(ctx context.Context, client *dcrd.Dcrd, interval float64,
	dataStore DataStore, webServer *web.Server) (*Collector, error) {

	c := &Collector{
		ctx:                ctx,
		webServer:          webServer,
		client:             client,
		collectionInterval: interval,
		dataStore:          dataStore,
	}

	if err := c.webServer.Templates.AddTemplate("mempool"); err != nil {
		log.Errorf("Unable to create new html template: %v", err)
		return nil, err
	}

	c.webServer.AddMenuItem(web.MenuItem{
		Href:      "/mempool",
		HyperText: "Mempool",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Historic mempool data",
		},
	})

	webServer.AddRoute("/mempool", web.GET, c.mempoolPage)
	webServer.AddRoute("/getmempool", web.GET, c.getMempool)
	webServer.AddRoute("/api/charts/mempool/{chartDataType}", web.GET, c.chart, web.ChartDataTypeCtx)

	return c, nil
}

func (c *Collector) StartMonitoring(ctx context.Context) {
	var mu sync.Mutex

	collectMempool := func() {

		mu.Lock()
		defer mu.Unlock()

		mempoolTransactionMap, err := c.client.Rpc.GetRawMempoolVerbose(dcrjson.GRMAll)
		if err != nil {
			log.Error(err)
			return
		}

		if len(mempoolTransactionMap) == 0 {
			return
		}

		mempoolDto := Mempool{
			NumberOfTransactions: len(mempoolTransactionMap),
			Time:                 web.NowUTC(),
			FirstSeenTime:        web.NowUTC(), //todo: use the time of the first tx in the mempool
		}

		for hashString, tx := range mempoolTransactionMap {
			hash, err := chainhash.NewHashFromStr(hashString)
			if err != nil {
				log.Error(err)
				continue
			}
			rawTx, err := c.client.Rpc.GetRawTransactionVerbose(hash)
			if err != nil {
				log.Error(err)
				continue
			}

			totalOut := 0.0
			for _, v := range rawTx.Vout {
				totalOut += v.Value
			}

			mempoolDto.Total += totalOut
			mempoolDto.TotalFee += tx.Fee
			mempoolDto.Size += tx.Size
			if mempoolDto.FirstSeenTime.Unix() > tx.Time {
				mempoolDto.FirstSeenTime = web.UnixTime(tx.Time)
			}

		}

		votes, err := c.client.Rpc.GetRawMempool(dcrjson.GRMVotes)
		if err != nil {
			log.Error(err)
			return
		}
		mempoolDto.Voters = len(votes)

		tickets, err := c.client.Rpc.GetRawMempool(dcrjson.GRMTickets)
		if err != nil {
			log.Error(err)
			return
		}
		mempoolDto.Tickets = len(tickets)

		revocations, err := c.client.Rpc.GetRawMempool(dcrjson.GRMRevocations)
		if err != nil {
			log.Error(err)
			return
		}
		mempoolDto.Revocations = len(revocations)

		if err = c.dataStore.StoreMempool(ctx, mempoolDto); err != nil {
			log.Error(err)
		}
	}

	lastMempoolTime, err := c.dataStore.LastMempoolTime()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("Unable to get last mempool entry time: %s", err.Error())
		}
	} else {
		sencodsPassed := math.Abs(time.Since(lastMempoolTime).Seconds())
		if sencodsPassed < c.collectionInterval {
			timeLeft := c.collectionInterval - sencodsPassed
			log.Infof("Fetching mempool every %dm, collected %0.2f ago, will fetch in %0.2f.", 1, sencodsPassed,
				timeLeft)
			time.Sleep(time.Duration(timeLeft) * time.Second)
		}
	}
	collectMempool()
	ticker := time.NewTicker(time.Duration(c.collectionInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collectMempool()
			break
		case <-ctx.Done():
			return
		}
	}
}
