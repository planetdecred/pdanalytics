// TODO: move to shared package
package web

import (
	"fmt"
	"math"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/txscript/v2"
)

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

// AddrPrefix represent the address name it's prefix and description
type AddrPrefix struct {
	Name        string
	Prefix      string
	Description string
}

// AddressPrefixes generates an array AddrPrefix by using chaincfg.Params
func AddressPrefixes(params *chaincfg.Params) []AddrPrefix {
	Descriptions := []string{"P2PK address",
		"P2PKH address prefix. Standard wallet address. 1 public key -> 1 private key",
		"Ed25519 P2PKH address prefix",
		"secp256k1 Schnorr P2PKH address prefix",
		"P2SH address prefix",
		"WIF private key prefix",
		"HD extended private key prefix",
		"HD extended public key prefix",
	}
	Name := []string{"PubKeyAddrID",
		"PubKeyHashAddrID",
		"PKHEdwardsAddrID",
		"PKHSchnorrAddrID",
		"ScriptHashAddrID",
		"PrivateKeyID",
		"HDPrivateKeyID",
		"HDPublicKeyID",
	}

	MainnetPrefixes := []string{"Dk", "Ds", "De", "DS", "Dc", "Pm", "dprv", "dpub"}
	TestnetPrefixes := []string{"Tk", "Ts", "Te", "TS", "Tc", "Pt", "tprv", "tpub"}
	SimnetPrefixes := []string{"Sk", "Ss", "Se", "SS", "Sc", "Ps", "sprv", "spub"}

	name := params.Name
	var netPrefixes []string
	if name == "mainnet" {
		netPrefixes = MainnetPrefixes
	} else if strings.HasPrefix(name, "testnet") {
		netPrefixes = TestnetPrefixes
	} else if name == "simnet" {
		netPrefixes = SimnetPrefixes
	} else {
		return nil
	}

	addrPrefix := make([]AddrPrefix, 0, len(Descriptions))
	for i, desc := range Descriptions {
		addrPrefix = append(addrPrefix, AddrPrefix{
			Name:        Name[i],
			Description: desc,
			Prefix:      netPrefixes[i],
		})
	}
	return addrPrefix
}

// CalculateHashRate calculates the hashrate from the difficulty value and
// the targetTimePerBlock in seconds. The hashrate returned is in form PetaHash
// per second (PH/s).
func CalculateHashRate(difficulty, targetTimePerBlock float64) float64 {
	return ((difficulty * math.Pow(2, 32)) / targetTimePerBlock) / 1000000
}
