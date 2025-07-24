//go:build root
// +build root

package main

import (
	"KNIRVROOT/config"
	"KNIRVROOT/utils"
	"log"
)

func init() {
	// This file is only compiled when the 'root' build tag is specified
	log.Println("Initializing KNIRVROOT Root Node")

	// Set the node role to Root
	nodeRole = config.Root

	// Set root constants from main package
	rootConstants := config.RootConstants{
		BlockchainAddress:    utils.BLOCKCHAIN_ADDRESS,
		BlockchainPrivateKey: utils.BLOCKCHAIN_PRIVATE_KEY,
		RootchainURL:         utils.ROOTCHAIN_URL,
		BlockchainName:       utils.BLOCKCHAIN_NAME,
		CurrencyName:         utils.CURRENCY_NAME,
		Decimal:              utils.DECIMAL,
		MiningDifficulty:     utils.MINING_DIFFICULTY,
		MiningReward:         utils.MINING_REWARD,
	}
	config.SetRootConstants(rootConstants)
	log.Println("Root constants set from build configuration")
}
