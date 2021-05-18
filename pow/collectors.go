// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pow

import (
	"context"
	"net/http"
	"time"

	"github.com/planetdecred/pdanalytics/app"
	"github.com/planetdecred/pdanalytics/app/helpers"
	"github.com/planetdecred/pdanalytics/web"
)

var (
	availablePows = []string{
		Coinmine,
		Luxor,
		F2pool,
		Uupool,
	}
)

type PowDataStore interface {
	PowTableName() string
	AddPowData(context.Context, []PowData) error
	LastPowEntryTime(source string) (time int64)
	FetchPowDataForSync(ctx context.Context, date int64, skip, take int) ([]PowData, int64, error)
	UpdatePowChart(ctx context.Context) error

	FetchEncodePowChart(ctx context.Context, dataType,
		binString string, pools ...string) ([]byte, error)
	PowCount(ctx context.Context) (int64, error)
	FetchPowData(ctx context.Context, offset, limit int) ([]PowDataDto, int64, error)
	FetchPowDataBySource(ctx context.Context, source string, offset, limit int) ([]PowDataDto, int64, error)
	FetchPowSourceData(ctx context.Context) ([]PowDataSource, error)
	FetchPowChartData(ctx context.Context, pool string, dataType string) ([]PowChartData, error)
	GetPowDistinctDates(ctx context.Context, vsps []string) ([]time.Time, error)
}

type Collector struct {
	pows   []Pow
	period int64
	store  PowDataStore
	server *web.Server
}

func Activate(ctx context.Context, disabledPows []string, period int64,
	store PowDataStore, server *web.Server, dataMode, httpMode bool) error {

	pows := make([]Pow, 0, len(availablePows)-len(disabledPows))
	disabledMap := make(map[string]struct{})
	for _, pow := range disabledPows {
		disabledMap[pow] = struct{}{}
	}

	for _, pow := range availablePows {
		if _, disabled := disabledMap[pow]; disabled {
			continue
		}

		if contructor, ok := PowConstructors[pow]; ok {
			lastEntryTime := store.LastPowEntryTime(pow)
			in, err := contructor(&http.Client{Timeout: 10 * time.Second}, lastEntryTime) // Consider if sharing a single client is better
			if err != nil {
				return err
			}
			pows = append(pows, in)
		}
	}

	c := &Collector{
		pows:   pows,
		period: period,
		store:  store,
		server: server,
	}

	if dataMode {
		go func() {
			if err := c.store.UpdatePowChart(ctx); err != nil {
				log.Error(err)
			}
			c.Run(ctx)
		}()
	}

	if httpMode {
		if err := c.setupServer(); err != nil {
			return err
		}
	}

	return nil
}

func (pc *Collector) Run(ctx context.Context) {
	app.MarkBusyIfFree()
	log.Info("Triggering PoW collectors.")

	lastCollectionDateUnix := pc.store.LastPowEntryTime("")
	lastCollectionDate := helpers.UnixTime(lastCollectionDateUnix)
	secondsPassed := time.Since(lastCollectionDate)
	period := time.Duration(pc.period) * time.Second

	if lastCollectionDateUnix > 0 && secondsPassed < period {
		timeLeft := period - secondsPassed
		log.Infof("Fetching PoW data every %dm, collected %s ago, will fetch in %s.", pc.period/60,
			helpers.DurationToString(secondsPassed), helpers.DurationToString(timeLeft))

		app.ReleaseForNewModule()
		time.Sleep(timeLeft)
	}

	if lastCollectionDateUnix > 0 && secondsPassed < period {
		// continually check the state of the app until its free to run this module
		app.MarkBusyIfFree()
	}
	pc.Collect(ctx)
	app.ReleaseForNewModule()
	go pc.CollectAsync(ctx)
}

func (pc *Collector) CollectAsync(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(time.Duration(pc.period) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Stopping PoW collectors")
			return
		case <-ticker.C:
			// continually check the state of the app until its free to run this module
			app.MarkBusyIfFree()
			completeCollectionCycle := pc.store.LastPowEntryTime("")
			collectionCycleDate := helpers.UnixTime(completeCollectionCycle)
			timeInterval := time.Since(collectionCycleDate)
			log.Info("The next collection cycle begins in", timeInterval)

			log.Info("Starting a new PoW collection cycle")
			pc.Collect(ctx)
			app.ReleaseForNewModule()
		}
	}
}

func (pc *Collector) Collect(ctx context.Context) {
	log.Info("Fetching PoW data.")
	for _, powInfo := range pc.pows {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := powInfo.Collect(ctx)
			if err != nil {
				log.Error(err, powInfo.Name())
			}
			err = pc.store.AddPowData(ctx, data)
			if err != nil {
				log.Error(err)
			}
		}
	}
	if err := pc.store.UpdatePowChart(ctx); err != nil {
		log.Error(err)
	}
}

func (c *Collector) setupServer() error {
	if err := c.server.Templates.AddTemplate("pow"); err != nil {
		log.Errorf("Unable to create new html template: %v", err)
		return err
	}

	c.server.AddMenuItem(web.MenuItem{
		Href:      "/pow",
		HyperText: "PoW",
		Info:      "Proof of Work mining pool data",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Proof of Work mining pool data",
		},
	}, web.HistoricNavGroup)
	c.server.AddRoute("/pow", web.GET, c.powPage)
	c.server.AddRoute("/filteredpow", web.GET, c.getFilteredPowData)
	c.server.AddRoute("/api/charts/pow/{chartDataType}", web.GET, c.chart, web.ChartDataTypeCtx)

	return nil
}
