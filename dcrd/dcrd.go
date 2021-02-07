package dcrd

import (
	"fmt"
	"math"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrd/txscript/v2"
)

type Dcrd struct {
	Rpc    *rpcclient.Client
	Params *chaincfg.Params
	Notif  *Notifier
}

// IsSynced returns the sync status of dcrd node by checking the age of
// the best block against the TargetTimePerBlock and a 5 mins tolerance
func (d Dcrd) IsSynced() (bool, error) {
	hash, err := d.Rpc.GetBestBlockHash()
	if err != nil {
		return false, err
	}
	blockHeader, err := d.Rpc.GetBlockHeader(hash)
	if err != nil {
		return false, err
	}
	maxAge := d.Params.TargetTimePerBlock + 5*time.Minute
	return time.Since(blockHeader.Timestamp) <= maxAge, nil
}

// DevSubsidyAddress returns the development subsidy address for the specified
// network.
func DevSubsidyAddress(params *chaincfg.Params) (string, error) {
	var devSubsidyAddress string
	var err error
	switch params.Name {
	case "testnet2":
		// TestNet2 uses an invalid organization PkScript
		devSubsidyAddress = "TccTkqj8wFqrUemmHMRSx8SYEueQYLmuuFk"
		err = fmt.Errorf("testnet2 has invalid project fund script")
	default:
		_, devSubsidyAddresses, _, err0 := txscript.ExtractPkScriptAddrs(
			params.OrganizationPkScriptVersion, params.OrganizationPkScript, params)
		if err0 != nil || len(devSubsidyAddresses) != 1 {
			err = fmt.Errorf("failed to decode dev subsidy address: %v", err0)
		} else {
			devSubsidyAddress = devSubsidyAddresses[0].String()
		}
	}
	return devSubsidyAddress, err
}

// CalculateHashRate calculates the hashrate from the difficulty value and
// the targetTimePerBlock in seconds. The hashrate returned is in form PetaHash
// per second (PH/s).
func CalculateHashRate(difficulty, targetTimePerBlock float64) float64 {
	return ((difficulty * math.Pow(2, 32)) / targetTimePerBlock) / 1000000
}
