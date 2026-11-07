package ledger

import "encoding/json"

// ValidatorState is the durable part of a governance validator record.
type ValidatorState struct {
	Stake    string          `json:"stake"`
	Active   bool            `json:"active"`
	Jailed   bool            `json:"jailed"`
	Reason   string          `json:"reason,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// BondState is the durable part of a bond escrow hold.
type BondState struct {
	HoldID          string `json:"hold_id"`
	Operator        string `json:"operator"`
	Amount          string `json:"amount"`
	RootRegistryID  string `json:"root_registry_id"`
	ExpiresAtUnix   uint64 `json:"expires_at_unix"`
	EscrowAddr      string `json:"escrow_addr"`
	Status          string `json:"status"`
	BootNodeChainID string `json:"boot_node_chain_id,omitempty"`
	BootNodeAddress string `json:"boot_node_address,omitempty"`
	ConsumedChainID string `json:"consumed_chain_id,omitempty"`
}

// State is the deterministic result of replaying the log. Every value is a
// canonical JSON scalar or a map keyed by a canonical string, so the state root
// is stable across processes and map iteration orders.
type State struct {
	ChainID       string                                `json:"chain_id"`
	Name          string                                `json:"name"`
	Symbol        string                                `json:"symbol"`
	Owner         string                                `json:"owner"`
	InitialSupply string                                `json:"initial_supply"`
	MaxSupply     string                                `json:"max_supply"`
	TotalSupply   string                                `json:"total_supply"`
	Balances      map[string]string                     `json:"balances"`
	Nonces        map[string]uint64                     `json:"nonces"`
	Registered    map[string]bool                       `json:"registered"`
	Validators    map[string]ValidatorState             `json:"validators"`
	Proposals     map[string]json.RawMessage            `json:"proposals"`
	Votes         map[string]map[string]json.RawMessage `json:"votes"`
	Params        json.RawMessage                       `json:"params,omitempty"`
	Governance    json.RawMessage                       `json:"governance,omitempty"`
	Bonds         map[string]BondState                  `json:"bonds,omitempty"`
}

// NewState returns an empty state with every map allocated.
func NewState() *State {
	return &State{
		Balances:   map[string]string{},
		Nonces:     map[string]uint64{},
		Registered: map[string]bool{},
		Validators: map[string]ValidatorState{},
		Proposals:  map[string]json.RawMessage{},
		Votes:      map[string]map[string]json.RawMessage{},
		Bonds:      map[string]BondState{},
	}
}

// Clone deep-copies a state so a caller can compare without aliasing.
func (s *State) Clone() *State {
	out := NewState()
	*out = *s
	out.Balances = make(map[string]string, len(s.Balances))
	for k, v := range s.Balances {
		out.Balances[k] = v
	}
	out.Nonces = make(map[string]uint64, len(s.Nonces))
	for k, v := range s.Nonces {
		out.Nonces[k] = v
	}
	out.Registered = make(map[string]bool, len(s.Registered))
	for k, v := range s.Registered {
		out.Registered[k] = v
	}
	out.Validators = make(map[string]ValidatorState, len(s.Validators))
	for k, v := range s.Validators {
		out.Validators[k] = v
	}
	out.Proposals = make(map[string]json.RawMessage, len(s.Proposals))
	for k, v := range s.Proposals {
		out.Proposals[k] = v
	}
	out.Votes = make(map[string]map[string]json.RawMessage, len(s.Votes))
	for k, v := range s.Votes {
		inner := make(map[string]json.RawMessage, len(v))
		for voter, vote := range v {
			inner[voter] = vote
		}
		out.Votes[k] = inner
	}
	out.Bonds = make(map[string]BondState, len(s.Bonds))
	for k, v := range s.Bonds {
		out.Bonds[k] = v
	}
	return out
}
