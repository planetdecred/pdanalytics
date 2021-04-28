package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/commstats"
	"github.com/planetdecred/pdanalytics/dcrd"
	exchangesModule "github.com/planetdecred/pdanalytics/exchanges"
	"github.com/planetdecred/pdanalytics/gov/politeia"
	"github.com/planetdecred/pdanalytics/homepage"
	"github.com/planetdecred/pdanalytics/mempool"
	"github.com/planetdecred/pdanalytics/netsnapshot"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/postgres"
	"github.com/planetdecred/pdanalytics/pow"
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

	var pgDb *postgres.PgDb
	var dbInstance = func() (*postgres.PgDb, error) {
		if pgDb == nil {
			pgDb, err = postgres.NewPgDb(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DebugLevel == "debug")
			if err != nil {
				return nil, err
			}
			if err = pgDb.CreateTables(ctx); err != nil {
				log.Error("Error creating tables: ", err)
				return nil, err
			}
		}
		return pgDb, nil
	}

	var mp *mempool.Collector
	if cfg.EnableMempool {
		mdb, err := dbInstance()
		if err != nil {
			return err
		}

		mp, err = mempool.NewCollector(ctx, client, cfg.MempoolInterval, mdb, server)
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
			syncDb, err := postgres.NewPgDb(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass,
				databaseName, cfg.DebugLevel == "debug")
			if err != nil {
				return err
			}

			if !syncDb.TableExists("block") {
				msg := fmt.Sprintf("the database, %s is missing the block table", databaseName)
				log.Error(msg)
				return errors.New(msg)
			}

			if !syncDb.TableExists("vote") {
				msg := fmt.Sprintf("the database, %s is missing the vote table", databaseName)
				log.Error(msg)
				return errors.New(msg)
			}
			syncDbs[databaseName] = syncDb
		}

		propDb, err := dbInstance()
		if err != nil {
			return err
		}
		_, err = propagation.New(ctx, client, propDb, syncDbs, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new propagation component, %s", err.Error())
		}
	}

	if cfg.EnableProposals || cfg.EnableProposalsHttp {
		db, err := dbInstance()
		if err != nil {
			return err
		}

		go func() {
			err = politeia.Activate(ctx, client, db, cfg.PoliteiaAPIURL,
				cfg.ProposalsFileName, cfg.PiPropRepoOwner, cfg.PiPropRepoName, cfg.DataDir, server,
				cfg.EnableProposals, cfg.EnableProposalsHttp)
			if err != nil {
				log.Error(err)
				requestShutdown()
			}
			log.Info("Proposals module Enabled")
		}()
	}

	if cfg.EnableNetworkSnapshot || cfg.EnableNetworkSnapshotHTTP {
		db, err := dbInstance()
		if err != nil {
			return err
		}

		err = netsnapshot.Activate(ctx, db, cfg.NetworkSnapshotOptions, server)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to activate network snapshot component, %s", err.Error())
		}
		log.Info("Network nodes module Enabled")
	}

	if cfg.EnableExchange || cfg.EnableExchangeHttp {
		db, err := dbInstance()
		if err != nil {
			return err
		}
		if err := exchangesModule.Activate(ctx, strings.Split(cfg.DisabledExchanges, ","), db, server,
			cfg.EnableExchange, cfg.EnableExchangeHttp); err != nil {
			return fmt.Errorf("Failed to ectivate the exchanges modules, %s", err.Error())
		}
		log.Info("Exchange module enabled")
	}

	if cfg.CommunityStat || cfg.CommunityStatHttp {
		db, err := dbInstance()
		if err != nil {
			return err
		}
		if err := commstats.Activate(ctx, db, server, &cfg.CommunityStatOptions); err != nil {
			return fmt.Errorf("Failed to ectivate the exchanges modules, %s", err.Error())
		}
		log.Info("Community stat module enabled")
	}

	if cfg.EnablePow || cfg.EnablePowHttp {
		db, err := dbInstance()
		if err != nil {
			return err
		}
		if err := pow.Activate(ctx, cfg.DisabledPows, cfg.PowInterval, db, server, cfg.EnablePow, cfg.EnablePowHttp); err != nil {
			return fmt.Errorf("Failed to ectivate the pow modules, %s", err.Error())
		}
		log.Info("PoW module enabled")
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
