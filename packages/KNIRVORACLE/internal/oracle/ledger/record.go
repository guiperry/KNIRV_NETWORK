// Package ledger makes the Oracle's token, wallet, governance and validator
// state durable.
//
// Every mutation is appended to an fsync'd event log before the caller returns,
// and the same record set replays deterministically into the in-memory state a
// fresh process starts from. The log is the source of truth; the NRN token and
// the governance system are read models hydrated from it.
//
// Records are also committed to the Oracle's audit MMR (see Oracle.recordLocked),
// so checkpoints and blocks commit to the state that produced them.
package ledger

// Record kinds. A kind is part of the on-disk format: renaming one breaks
// replay of existing logs, so add a new kind instead.
const (
	KindGenesis         = "genesis"
	KindTokenMint       = "token_mint"
	KindTokenBurn       = "token_burn"
	KindTokenTransfer   = "token_transfer"
	KindTokenFee        = "token_fee"
	KindWalletRegister  = "wallet_register"
	KindValidatorUpsert = "validator_upsert"
	KindValidatorRemove = "validator_remove"
	KindProposalUpsert  = "proposal_upsert"
	KindVoteCast        = "vote_cast"
	KindParamsSet       = "params_set"
	KindGovernanceState = "governance_state"
)

// Genesis is written once, when the log is empty.
type Genesis struct {
	ChainID       string `json:"chain_id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Owner         string `json:"owner"`
	InitialSupply string `json:"initial_supply"`
	MaxSupply     string `json:"max_supply"`
}

// TokenMint credits an address and grows the supply.
type TokenMint struct {
	To     string `json:"to"`
	Amount string `json:"amount"`
}

// TokenBurn debits an address and shrinks the supply.
type TokenBurn struct {
	From   string `json:"from"`
	Amount string `json:"amount"`
	Reason string `json:"reason,omitempty"`
}

// TokenTransfer moves value between two addresses and advances the nonce.
type TokenTransfer struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
	Nonce  uint64 `json:"nonce"`
	// SetNonce is true for signature-authorized transfers (which advance the
	// replay nonce) and false for internal moves such as an escrow deposit.
	SetNonce bool `json:"set_nonce"`
}

// TokenFee is a fee collection: burn part of it and credit the rest to the
// reward pool. It is one record so a crash cannot split the pair.
type TokenFee struct {
	From       string `json:"from"`
	RewardPool string `json:"reward_pool"`
	Burn       string `json:"burn"`
	Reward     string `json:"reward"`
}

// WalletRegister marks an address as a registered wallet.
type WalletRegister struct {
	Address string `json:"address"`
}

// ValidatorUpsert stores the complete validator record; replay overwrites.
type ValidatorUpsert struct {
	Address  string `json:"address"`
	Stake    string `json:"stake"`
	Active   bool   `json:"active"`
	Jailed   bool   `json:"jailed"`
	Reason   string `json:"reason,omitempty"`
	Metadata any    `json:"metadata,omitempty"`
}

// ValidatorRemove drops a validator from the set.
type ValidatorRemove struct {
	Address string `json:"address"`
}

// ProposalUpsert stores the complete proposal record.
type ProposalUpsert struct {
	ID       string `json:"id"`
	Proposal any    `json:"proposal"`
}

// VoteCast stores one vote; replay overwrites a voter's previous vote.
type VoteCast struct {
	ProposalID string `json:"proposal_id"`
	Voter      string `json:"voter"`
	Vote       any    `json:"vote"`
}

// ParamsSet replaces the network parameters.
type ParamsSet struct {
	Params any `json:"params"`
}

// GovernanceState is a full governance snapshot. Governance mutates several
// maps together (proposal tally, validator flags, parameters), so one record
// per successful mutation is both smaller than a per-entity protocol and
// impossible to replay partially.
type GovernanceState struct {
	Proposals  any `json:"proposals"`
	Votes      any `json:"votes"`
	Validators any `json:"validators"`
	Params     any `json:"params"`
}
