// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package vsp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/planetdecred/pdanalytics/app"
	"github.com/planetdecred/pdanalytics/app/helpers"
	"github.com/planetdecred/pdanalytics/web"
)

const (
	requestURL = "https://api.decred.org/?c=gsd"
	retryLimit = 3
)

func Activate(ctx context.Context, period int64, store DataStore, server *web.Server, dataMode, httpMode bool) error {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}

	c := &Collector{
		client:    http.Client{Timeout: time.Minute},
		period:    time.Duration(period),
		request:   request,
		dataStore: store,
		server:    server,
	}

	if dataMode {
		go func() {
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

func (vsp *Collector) fetch(ctx context.Context, response interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// log.Tracef("GET %v", requestURL)
	resp, err := vsp.client.Do(vsp.request.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Failed to decode json: %v", err))
	}

	return nil
}

func (vsp *Collector) Run(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	lastCollectionDate := vsp.dataStore.LastVspTickEntryTime()
	secondsPassed := time.Since(lastCollectionDate)
	period := vsp.period * time.Second

	log.Info("Starting VSP collection cycle.")

	if secondsPassed < period {
		timeLeft := period - secondsPassed
		log.Infof("Fetching VSPs every %dm, collected %s ago, will fetch in %s.", vsp.period/60, helpers.DurationToString(secondsPassed),
			helpers.DurationToString(timeLeft))

		time.Sleep(timeLeft)
	}

	// continually check the state of the app until its free to run this module
	app.MarkBusyIfFree()

	err := vsp.collectAndStore(ctx)
	app.ReleaseForNewModule()
	if err != nil {
		log.Errorf("Could not start collection: %s", err.Error())
		return
	}

	go func() {
		ticker := time.NewTicker(vsp.period * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Infof("Shutting down VSP collector")
				return
			case <-ticker.C:
				// continually check the state of the app until its free to run this module
				app.MarkBusyIfFree()
				err := vsp.collectAndStore(ctx)
				app.ReleaseForNewModule()
				if err != nil {
					return
				}
			}
		}
	}()
}

func (vsp *Collector) collectAndStore(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	log.Info("Fetching VSP from source")

	resp := new(Response)
	err := vsp.fetch(ctx, resp)
	for retry := 0; err != nil; retry++ {
		if retry == retryLimit {
			return err
		}
		log.Warn(err)
		err = vsp.fetch(ctx, resp)
	}

	numberStored, errs := vsp.dataStore.StoreVSPs(ctx, *resp)
	for _, err = range errs {
		if err != nil {
			if e, ok := err.(PoolTickTimeExistsError); ok {
				log.Trace(e)
			} else {
				log.Error(err)
				return err
			}
		}
	}

	log.Infof("Saved ticks for %d VSPs from %s", numberStored, requestURL)
	if err = vsp.dataStore.UpdateVspChart(ctx); err != nil {
		return fmt.Errorf("Error in initial VSP bin update, %s", err.Error())
	}
	return nil
}

func (vsp *Collector) setupServer() error {
	if err := vsp.server.Templates.AddTemplate("vsp"); err != nil {
		log.Errorf("Unable to create new html template: %v", err)
		return err
	}

	vsp.server.AddMenuItem(web.MenuItem{
		Href:      "/vsp",
		HyperText: "VSP",
		Info:      "Voting Service Provider data",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Voting Service Provider data",
		},
	})

	vsp.server.AddRoute("/vsp", web.GET, vsp.vspPage)
	vsp.server.AddRoute("/vsps", web.GET, vsp.getFilteredVspTicks)
	vsp.server.AddRoute("/api/charts/vsp/{chartDataType}", web.GET, vsp.chart, web.ChartDataTypeCtx)

	return nil
}
