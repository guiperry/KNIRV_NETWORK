package main

import (
	"KNIRVGRAPH/internal/blockchain"
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"fmt"
	"time"
)

func main() {
	fmt.Println("Creating storage...")
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		fmt.Printf("Failed to create storage: %v\n", err)
		return
	}

	fmt.Println("Creating chain...")
	chain := blockchain.NewChain(storage)

	fmt.Println("Creating test block...")
	block := &types.Block{
		Header: types.BlockHeader{
			Height:    1,
			Timestamp: time.Now(),
			PrevHash:  "",
		},
		Transactions: []*types.Transaction{},
	}
	block.Hash = block.CalculateHash()

	fmt.Println("Adding block...")
	err = chain.AddBlock(block)
	if err != nil {
		fmt.Printf("Failed to add block: %v\n", err)
		return
	}

	fmt.Println("Block added successfully!")
}
