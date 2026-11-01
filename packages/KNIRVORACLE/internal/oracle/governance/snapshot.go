package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// snapshotLocked exports the whole governance state in a deterministic order.
// Governance mutates proposals, votes, validators and parameters together, so
// one snapshot per successful mutation is both smaller than a per-entity
// protocol and impossible to replay partially.
type snapshot struct {
	Proposals  []*Proposal                 `json:"proposals"`
	Votes      map[string]map[string]*Vote `json:"votes"`
	Validators []*Validator                `json:"validators"`
	Params     *NetworkParameters          `json:"params"`
}

// SetRecorder installs the durable-ledger hook. Call before serving requests.
func (gs *GovernanceSystem) SetRecorder(recorder func(kind string, payload any) error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.recorder = recorder
	gs.lastFingerprint = ""
}

// record writes a snapshot if the state changed since the last one. The caller
// must hold gs.mu (it is called from deferred functions inside the mutators, so
// it runs after the mutation and before the lock is released).
func (gs *GovernanceSystem) snapshotLocked() {
	snap := gs.exportLocked()
	body, err := json.Marshal(snap)
	if err != nil {
		gs.logger.Warn("governance snapshot encode failed; not recorded", zap.Error(err))
		return
	}
	sum := sha256.Sum256(body)
	fingerprint := hex.EncodeToString(sum[:])
	if fingerprint == gs.lastFingerprint {
		return
	}
	if gs.recorder != nil {
		if err := gs.recorder(ledger.KindGovernanceState, ledger.GovernanceState{
			Proposals: snap.Proposals, Votes: snap.Votes, Validators: snap.Validators, Params: snap.Params,
		}); err != nil {
			gs.logger.Warn("governance record failed; state is in memory only", zap.Error(err))
			return
		}
	}
	gs.lastFingerprint = fingerprint
}

func (gs *GovernanceSystem) exportLocked() snapshot {
	proposals := make([]*Proposal, 0, len(gs.proposals))
	for _, p := range gs.proposals {
		proposals = append(proposals, p)
	}
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].ID < proposals[j].ID })

	validators := make([]*Validator, 0, len(gs.validators))
	for _, v := range gs.validators {
		validators = append(validators, v)
	}
	sort.Slice(validators, func(i, j int) bool { return validators[i].Address.String() < validators[j].Address.String() })

	votes := make(map[string]map[string]*Vote, len(gs.votes))
	for proposalID, byVoter := range gs.votes {
		inner := make(map[string]*Vote, len(byVoter))
		for voter, vote := range byVoter {
			inner[voter.String()] = vote
		}
		votes[proposalID] = inner
	}
	return snapshot{Proposals: proposals, Votes: votes, Validators: validators, Params: gs.params}
}

// Hydrate replaces the in-memory governance read model with state replayed from
// the ledger.
func (gs *GovernanceSystem) Hydrate(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return fmt.Errorf("governance hydrate: %w", err)
	}

	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.proposals = make(map[string]*Proposal, len(snap.Proposals))
	for _, p := range snap.Proposals {
		if p != nil {
			gs.proposals[p.ID] = p
		}
	}
	gs.validators = make(map[types.Address]*Validator, len(snap.Validators))
	for _, v := range snap.Validators {
		if v != nil {
			gs.validators[v.Address] = v
		}
	}
	gs.votes = make(map[string]map[types.Address]*Vote, len(snap.Votes))
	for proposalID, byVoter := range snap.Votes {
		inner := make(map[types.Address]*Vote, len(byVoter))
		for rawVoter, vote := range byVoter {
			address, err := types.AddressFromString(rawVoter)
			if err != nil {
				return fmt.Errorf("governance hydrate voter %q: %w", rawVoter, err)
			}
			inner[address] = vote
		}
		gs.votes[proposalID] = inner
	}
	if snap.Params != nil {
		gs.params = snap.Params
	}
	gs.lastFingerprint = ""
	return nil
}
