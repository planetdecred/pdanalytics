// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package exchanges

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/planetdecred/pdanalytics/app"
	"github.com/planetdecred/pdanalytics/app/helpers"
	"github.com/planetdecred/pdanalytics/exchanges/ticks"
	"github.com/planetdecred/pdanalytics/web"
)

const (
	clientTimeout = time.Minute
)

type TickHub struct {
	server     *web.Server
	collectors []ticks.Collector
	client     *http.Client
	store      ticks.Store
}

var (
	availableExchanges = []string{
		ticks.Bittrex,
		ticks.Bittrexusd,
		ticks.Binance,
		// ticks.Bleutrade,
		ticks.Poloniex,
	}
)

func Activate(ctx context.Context, disabledexchanges []string, store ticks.Store, server *web.Server,
	 dataMode, httpMode bool) error {
	collectors := make([]ticks.Collector, 0, len(availableExchanges)-len(disabledexchanges))
	disabledMap := make(map[string]struct{})
	for _, e := range disabledexchanges {
		disabledMap[e] = struct{}{}
	}
	enabledExchanges := make([]string, 0, cap(collectors))
	for _, exchange := range availableExchanges {
		if _, ok := disabledMap[exchange]; !ok {
			collector, err := ticks.CollectorConstructors[exchange](ctx, store)
			if err != nil {
				log.Error(err)
				continue
			}
			collectors = append(collectors, collector)
			enabledExchanges = append(enabledExchanges, exchange)
		}
	}

	if len(collectors) == 0 {
		return fmt.Errorf("No tick collectors")
	}

	log.Infof("Enabled exchange tick collection for %v", enabledExchanges)

	t := &TickHub{
		collectors: collectors,
		client:     &http.Client{Timeout: clientTimeout},
		store:      store,
		server: server,
	}

	if httpMode {
		if err := t.setupHttp(); err != nil {
			return err
		}
	}

	if dataMode {
		go func() {
			t.Run(ctx)
		}()
	}
	return nil
}

func (hub *TickHub) CollectShort(ctx context.Context) {
	wg := new(sync.WaitGroup)
	for _, collector := range hub.collectors {
		if ctx.Err() != nil {
			log.Error(ctx.Err())
			break
		}
		wg.Add(1)
		func(ctx context.Context, wg *sync.WaitGroup, collector ticks.Collector) {
			err := collector.GetShort(ctx)
			if err != nil {
				log.Error(err)
			}
			wg.Done()
		}(ctx, wg, collector)
	}
	wg.Wait()
	log.Info("Completed short collection")
}

func (hub *TickHub) CollectLong(ctx context.Context) {
	wg := new(sync.WaitGroup)
	for _, collector := range hub.collectors {
		if ctx.Err() != nil {
			log.Error(ctx.Err())
			break
		}
		wg.Add(1)
		func(ctx context.Context, wg *sync.WaitGroup, collector ticks.Collector) {
			err := collector.GetLong(ctx)
			if err != nil {
				log.Error(err)
			}
			wg.Done()
		}(ctx, wg, collector)
	}
	wg.Wait()
	log.Info("Completed long collection")
}

func (hub *TickHub) CollectHistoric(ctx context.Context) {
	wg := new(sync.WaitGroup)
	for _, collector := range hub.collectors {
		if ctx.Err() != nil {
			log.Error(ctx.Err())
			break
		}
		wg.Add(1)
		func(ctx context.Context, wg *sync.WaitGroup, collector ticks.Collector) {
			err := collector.GetHistoric(ctx)
			if err != nil {
				log.Error(err)
			}
			wg.Done()
		}(ctx, wg, collector)
	}
	wg.Wait()
	log.Info("Completed historic collection")
}

func (hub *TickHub) CollectAll(ctx context.Context) {
	for _, collector := range hub.collectors {
		if ctx.Err() != nil {
			log.Error(ctx.Err())
			break
		}

		err := collector.GetShort(ctx)
		if err != nil {
			log.Error(err)
		}

		err = collector.GetLong(ctx)
		if err != nil {
			log.Error(err)
		}

		err = collector.GetHistoric(ctx)
		if err != nil {
			log.Error(err)
		}
	}
}

func (hub *TickHub) Run(ctx context.Context) {
	shortTicker := time.NewTicker(5 * time.Minute)
	longTicker := time.NewTicker(time.Hour)
	dayTicker := time.NewTicker(24 * time.Hour)
	defer shortTicker.Stop()
	defer longTicker.Stop()
	defer dayTicker.Stop()

	if ctx.Err() != nil {
		log.Error(ctx.Err())
		return
	}

	lastCollectionDate := hub.store.LastExchangeTickEntryTime()
	secondsPassed := time.Since(lastCollectionDate)
	period := 5 * time.Minute

	if secondsPassed < period {
		timeLeft := period - secondsPassed
		log.Infof("Fetching exchange ticks every %dm, collected %s ago, will fetch in %s.", period/time.Minute, helpers.DurationToString(secondsPassed),
			helpers.DurationToString(timeLeft))

		time.Sleep(timeLeft)
	}

	registerStarter := func() {
		// continually check the state of the app until its free to run this module
		app.MarkBusyIfFree()

		log.Info("Starting exchange tick collection cycle")
	}

	registerStarter()
	hub.CollectAll(ctx)
	app.ReleaseForNewModule()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-shortTicker.C:
				registerStarter()
				hub.CollectShort(ctx)
				app.ReleaseForNewModule()
			case <-longTicker.C:
				registerStarter()
				hub.CollectLong(ctx)
				app.ReleaseForNewModule()
			case <-dayTicker.C:
				registerStarter()
				hub.CollectHistoric(ctx)
				app.ReleaseForNewModule()
			}
		}
	}()
}

func (hub *TickHub) setupHttp() error {
	hub.server.AddMenuItem(web.MenuItem{
		Href:      "/exchanges",
		HyperText: "Exchanges",
		Info:      "Historical exchange rate information",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Historical exchange rate information",
		},
	})

	if err := hub.server.Templates.AddTemplate("exchange"); err != nil {
		return err
	}

	hub.server.AddRoute("/exchanges", web.GET, hub.exchangesPage)
	hub.server.AddRoute("/exchangedata", web.GET, hub.getFilteredExchangeTicks)
	hub.server.AddRoute("/exchangechart", web.GET, hub.getExchangeChartData)
	hub.server.AddRoute("/api/exchanges/intervals", web.GET, hub.tickIntervalsByExchangeAndPair)
	hub.server.AddRoute("/api/exchanges/currency-pairs", web.GET, hub.currencyPairByExchange)
	hub.server.AddRoute("/api/charts/exchange/{chartDataType}", web.GET, hub.chart, web.ChartDataTypeCtx)

	return nil
}
