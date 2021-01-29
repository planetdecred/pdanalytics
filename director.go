package main

import (
	"fmt"

	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/stakingreward"
	"github.com/planetdecred/pdanalytics/web"
)

func setupModules(cfg *config, dcrdClient *rpcclient.Client, webServer *web.Server, xcBot *exchanges.ExchangeBot, notif *notifier) error {
	if cfg.EnableStakingRewardCalculator == 1 {
		rewardCalculator, err := stakingreward.New(dcrdClient, webServer, "", xcBot, activeChain)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new staking reward component, %s", err.Error())
		}

		notif.RegisterBlockHandlerGroup(rewardCalculator.ConnectBlock)
	}

	if cfg.EnableChainParameters == 1 {
		_, err := parameters.New(dcrdClient, webServer, "", activeChain)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new parameters component, %s", err.Error())
		}
	}

	if cfg.EnableAttackCost == 1 {
		attCost, err := attackcost.New(dcrdClient, webServer, "", xcBot, activeChain)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create new attackcost component, %s", err.Error())
		}

		notif.RegisterBlockHandlerGroup(attCost.ConnectBlock)
	}

	return nil
}
