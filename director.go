package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/homepage"
	"github.com/planetdecred/pdanalytics/mempool"
	"github.com/planetdecred/pdanalytics/mempool/postgres"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/propagation"
	"github.com/planetdecred/pdanalytics/stakingreward"
	"github.com/planetdecred/pdanalytics/web"
)

func setupModules(ctx context.Context, cfg *config, client *dcrd.Dcrd, server *web.Server, xcBot *exchanges.ExchangeBot) error {
	var err error

	var stk *stakingreward.Calculator
	if cfg.EnableStakingRewardCalculator {
		stk, err = stakingreward.New(client, server, xcBot)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create staking reward component, %s", err.Error())
		}

		log.Info("Staking Reward Calculator Enabled")
	}

	var prms *parameters.Parameters
	if cfg.EnableChainParameters {
		prms, err = parameters.New(client, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create parameters component, %s", err.Error())
		}

		log.Info("Chain Parameters Enabled")
	}

	var ac *attackcost.Attackcost
	if cfg.EnableAttackCost {
		ac, err = attackcost.New(client, server, xcBot)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create attackcost component, %s", err.Error())
		}

		log.Info("Attack Cost Calculator Enabled")
	}

	var db *sql.DB
	dbInstance := func() (*sql.DB, error) {
		if db == nil {
			db, err = dbhelper.Connect(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)
			if err != nil {
				return nil, fmt.Errorf("error in establishing database connection: %s", err.Error())
			}
			db.SetMaxOpenConns(5)
		}
		return db, nil
	}

	var mp *mempool.Collector
	if cfg.EnableMempool {
		db, err := dbInstance()
		if err != nil {
			return err
		}

		mpdb := postgres.NewPgDb(db, cfg.DebugLevel == "debug")
		if err = mpdb.CreateTables(ctx); err != nil {
			log.Error("Error creating mempool tables: ", err)
			return err
		}

		mp, err = mempool.NewCollector(ctx, client, cfg.MempoolInterval, mpdb, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new mempool component, %s", err.Error())
		}
		go mp.StartMonitoring(ctx)

		log.Info("Attack Cost Calculator Enabled")
	}

	if cfg.EnablePropagation {
		var syncDbs = map[string]propagation.Store{}
		//register instances
		for i := 0; i < len(cfg.SyncDatabases); i++ {
			databaseName := cfg.SyncDatabases[i]
			dbConn, err := dbhelper.Connect(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, databaseName)
			if err != nil {
				return fmt.Errorf("error in establishing database connection: %s", err.Error())
			}
			dbConn.SetMaxOpenConns(5)

			// the external db must be accessible and contain block and vote tables
			syncDb := propagation.NewPgDb(dbConn, cfg.DebugLevel == "Debug")

			if !syncDb.BlockTableExits() {
				msg := fmt.Sprintf("the database, %s is missing the block table", databaseName)
				log.Error(msg)
				return errors.New(msg)
			}

			if !syncDb.VoteTableExits() {
				msg := fmt.Sprintf("the database, %s is missing the vote table", databaseName)
				log.Error(msg)
				return errors.New(msg)
			}
			syncDbs[databaseName] = syncDb
		}

		dbConn, err := dbInstance()
		if err != nil {
			return err
		}

		propDb := propagation.NewPgDb(dbConn, cfg.DebugLevel == "Debug")
		if err := propDb.CreateTables(ctx); err != nil {
			return fmt.Errorf("Failed to create new propagation component, error in creating tables, %s",
				err.Error())
		}
		_, err = propagation.New(ctx, client, propDb, syncDbs, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new propagation component, %s", err.Error())
		}
	}

	_, err = homepage.New(server, homepage.Mods{
		Stk: stk,
		Prm: prms,
		Ac:  ac,
	})
	if err != nil {
		log.Error(err)
	}
	return nil
}
