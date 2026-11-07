package ledger

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// Apply is the one deterministic state-transition function. The live path and
// replay both call it, so a bug can never make a restart produce different
// state from the live process.
//
// It has no wall clock, no randomness and no map-iteration dependence: adding a
// record can only touch the keys named in its payload.
func Apply(s *State, r Record) error {
	switch r.Kind {
	case KindGenesis:
		var g Genesis
		if err := json.Unmarshal(r.Payload, &g); err != nil {
			return fmt.Errorf("genesis: %w", err)
		}
		s.ChainID, s.Name, s.Symbol, s.Owner = g.ChainID, g.Name, g.Symbol, g.Owner
		s.InitialSupply, s.MaxSupply = g.InitialSupply, g.MaxSupply
		s.TotalSupply = g.InitialSupply
		s.Balances[g.Owner] = g.InitialSupply

	case KindTokenMint:
		var m TokenMint
		if err := json.Unmarshal(r.Payload, &m); err != nil {
			return fmt.Errorf("token_mint: %w", err)
		}
		if err := addToBalance(s, m.To, m.Amount); err != nil {
			return err
		}
		s.TotalSupply = addDecimal(s.TotalSupply, m.Amount)

	case KindTokenBurn:
		var b TokenBurn
		if err := json.Unmarshal(r.Payload, &b); err != nil {
			return fmt.Errorf("token_burn: %w", err)
		}
		if err := subFromBalance(s, b.From, b.Amount); err != nil {
			return err
		}
		s.TotalSupply = subDecimal(s.TotalSupply, b.Amount)

	case KindTokenTransfer:
		var t TokenTransfer
		if err := json.Unmarshal(r.Payload, &t); err != nil {
			return fmt.Errorf("token_transfer: %w", err)
		}
		if err := subFromBalance(s, t.From, t.Amount); err != nil {
			return err
		}
		if err := addToBalance(s, t.To, t.Amount); err != nil {
			return err
		}
		if t.SetNonce {
			s.Nonces[t.From]++
		}

	case KindTokenFee:
		var f TokenFee
		if err := json.Unmarshal(r.Payload, &f); err != nil {
			return fmt.Errorf("token_fee: %w", err)
		}
		total := addDecimal(f.Burn, f.Reward)
		if err := subFromBalance(s, f.From, total); err != nil {
			return err
		}
		if err := addToBalance(s, f.RewardPool, f.Reward); err != nil {
			return err
		}
		s.TotalSupply = subDecimal(s.TotalSupply, f.Burn)

	case KindWalletRegister:
		var w WalletRegister
		if err := json.Unmarshal(r.Payload, &w); err != nil {
			return fmt.Errorf("wallet_register: %w", err)
		}
		s.Registered[w.Address] = true

	case KindValidatorUpsert:
		var v ValidatorUpsert
		if err := json.Unmarshal(r.Payload, &v); err != nil {
			return fmt.Errorf("validator_upsert: %w", err)
		}
		meta, err := json.Marshal(v.Metadata)
		if err != nil {
			return fmt.Errorf("validator_upsert metadata: %w", err)
		}
		s.Validators[v.Address] = ValidatorState{
			Stake: v.Stake, Active: v.Active, Jailed: v.Jailed, Reason: v.Reason, Metadata: meta,
		}

	case KindValidatorRemove:
		var v ValidatorRemove
		if err := json.Unmarshal(r.Payload, &v); err != nil {
			return fmt.Errorf("validator_remove: %w", err)
		}
		delete(s.Validators, v.Address)

	case KindProposalUpsert:
		var p ProposalUpsert
		if err := json.Unmarshal(r.Payload, &p); err != nil {
			return fmt.Errorf("proposal_upsert: %w", err)
		}
		raw, err := json.Marshal(p.Proposal)
		if err != nil {
			return fmt.Errorf("proposal_upsert payload: %w", err)
		}
		s.Proposals[p.ID] = raw

	case KindVoteCast:
		var v VoteCast
		if err := json.Unmarshal(r.Payload, &v); err != nil {
			return fmt.Errorf("vote_cast: %w", err)
		}
		raw, err := json.Marshal(v.Vote)
		if err != nil {
			return fmt.Errorf("vote_cast payload: %w", err)
		}
		if s.Votes[v.ProposalID] == nil {
			s.Votes[v.ProposalID] = map[string]json.RawMessage{}
		}
		s.Votes[v.ProposalID][v.Voter] = raw

	case KindParamsSet:
		var p ParamsSet
		if err := json.Unmarshal(r.Payload, &p); err != nil {
			return fmt.Errorf("params_set: %w", err)
		}
		raw, err := json.Marshal(p.Params)
		if err != nil {
			return fmt.Errorf("params_set payload: %w", err)
		}
		s.Params = raw

	case KindGovernanceState:
		var g GovernanceState
		if err := json.Unmarshal(r.Payload, &g); err != nil {
			return fmt.Errorf("governance_state: %w", err)
		}
		raw, err := json.Marshal(g)
		if err != nil {
			return fmt.Errorf("governance_state payload: %w", err)
		}
		s.Governance = raw

	case KindBondHold:
		var h BondHold
		if err := json.Unmarshal(r.Payload, &h); err != nil {
			return fmt.Errorf("bond_hold: %w", err)
		}
		// A hold only ever creates or refreshes a pending escrow; if a later
		// release/refund record already advanced the status, keep that status.
		if existing, ok := s.Bonds[h.HoldID]; ok && existing.Status != BondStatusPending {
			break
		}
		s.Bonds[h.HoldID] = BondState{
			HoldID: h.HoldID, Operator: h.Operator, Amount: h.Amount,
			RootRegistryID: h.RootRegistryID, ExpiresAtUnix: h.ExpiresAtUnix,
			EscrowAddr: h.EscrowAddr, Status: h.Status,
		}

	case KindBondRelease:
		var rel BondRelease
		if err := json.Unmarshal(r.Payload, &rel); err != nil {
			return fmt.Errorf("bond_release: %w", err)
		}
		bs, ok := s.Bonds[rel.HoldID]
		if !ok {
			return fmt.Errorf("bond_release: unknown hold %q", rel.HoldID)
		}
		bs.Status = rel.Status
		bs.BootNodeChainID = rel.BootNodeChainID
		bs.BootNodeAddress = rel.BootNodeAddress
		bs.ConsumedChainID = rel.ConsumedChainID
		s.Bonds[rel.HoldID] = bs

	default:
		return fmt.Errorf("ledger: unknown record kind %q (refusing to guess)", r.Kind)
	}
	return nil
}

// Replay builds state from records from scratch.
func Replay(records []Record) (*State, error) {
	s := NewState()
	for _, r := range records {
		if err := Apply(s, r); err != nil {
			return nil, fmt.Errorf("replay seq %d: %w", r.Seq, err)
		}
	}
	return s, nil
}

func addToBalance(s *State, address, amount string) error {
	s.Balances[address] = addDecimal(s.Balances[address], amount)
	return nil
}

func subFromBalance(s *State, address, amount string) error {
	current := s.Balances[address]
	if current == "" {
		current = "0"
	}
	next := subDecimal(current, amount)
	if next[0] == '-' {
		return fmt.Errorf("insufficient balance for %s: have %s, need %s", address, current, amount)
	}
	s.Balances[address] = next
	return nil
}

func addDecimal(a, b string) string {
	return decimal(a, b, func(x, y *big.Int) *big.Int { return new(big.Int).Add(x, y) })
}

func subDecimal(a, b string) string {
	return decimal(a, b, func(x, y *big.Int) *big.Int { return new(big.Int).Sub(x, y) })
}

func decimal(a, b string, op func(*big.Int, *big.Int) *big.Int) string {
	x, ok := new(big.Int).SetString(zeroIfEmpty(a), 10)
	if !ok {
		return "0"
	}
	y, ok := new(big.Int).SetString(zeroIfEmpty(b), 10)
	if !ok {
		return x.String()
	}
	return op(x, y).String()
}

func zeroIfEmpty(v string) string {
	if v == "" {
		return "0"
	}
	return v
}
