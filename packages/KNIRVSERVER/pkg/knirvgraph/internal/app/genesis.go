package app

import (
	"KNIRVGRAPH/internal/types"
	"encoding/json"
	"os"
	"time"
)

type GenesisDoc struct {
	ChainID     string             `json:"chain_id"`
	GenesisTime time.Time          `json:"genesis_time"`
	Validators  []GenesisValidator `json:"validators"`
	AppState    GenesisAppState    `json:"app_state"`
}

type GenesisValidator struct {
	Address string `json:"address"`
	PubKey  string `json:"pub_key"`
	Power   int64  `json:"power"`
	Name    string `json:"name"`
}

type GenesisAppState struct {
	Accounts []types.Account `json:"accounts"`
}

func DefaultGenesis() *GenesisDoc {
	return &GenesisDoc{
		ChainID:     "blockchain-testnet",
		GenesisTime: time.Now(),
		Validators: []GenesisValidator{
			{
				Address: "0x1234567890123456789012345678901234567890",
				PubKey:  "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Power:   10,
				Name:    "validator1",
			},
		},
		AppState: GenesisAppState{
			Accounts: []types.Account{
				{
					Address: "0x1234567890123456789012345678901234567890",
					Balance: 1000000,
					Nonce:   0,
				},
				{
					Address: "0x0987654321098765432109876543210987654321",
					Balance: 250000,
					Nonce:   0,
				},
				{
					Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					Balance: 500000,
					Nonce:   0,
				},
			},
		},
	}
}

func (g *GenesisDoc) SaveAs(file string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0600) // More secure file permissions
}

func LoadGenesis(file string) (*GenesisDoc, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var genesis GenesisDoc
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, err
	}

	return &genesis, nil
}
