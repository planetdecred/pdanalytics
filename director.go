package main

import (
	"fmt"

	"github.com/planetdecred/pdanalytics/base"
	"github.com/planetdecred/pdanalytics/parameters"
)

func setupModules(cfg *config, b *base.Base) error {
	if cfg.EnableStakingRewardCalculator {
		// rewardCalculator, err := stakingreward.New(dcrdClient, webServer, "", xcBot, activeChain)
		// if err != nil {
		// 	log.Error(err)
		// 	return fmt.Errorf("Failed to create staking reward component, %s", err.Error())
		// }

		// notif.RegisterBlockHandlerGroup(rewardCalculator.ConnectBlock)
	}

	if cfg.EnableChainParameters {
		_, err := parameters.New(b)
		if err != nil {
			log.Error(err)
			return fmt.Errorf("Failed to create parameters component, %s", err.Error())
		}

		log.Info("Chain Parameters Enabled")
	}

	if cfg.EnableAttackCost {
		// attCost, err := attackcost.New(dcrdClient, webServer, "", xcBot, activeChain)
		// if err != nil {
		// 	log.Error(err)
		// 	return fmt.Errorf("Failed to create attackcost component, %s", err.Error())
		// }

		// notif.RegisterBlockHandlerGroup(attCost.ConnectBlock)
	}

	return nil
}
