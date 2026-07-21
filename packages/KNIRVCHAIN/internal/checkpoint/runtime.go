package checkpoint

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"KNIRVCHAIN/internal/blockchain"
)

type RuntimeSource interface {
	CheckpointSource
	BlocksRange(start, end uint64) ([]*blockchain.Block, error)
}

type RuntimeStatus struct {
	Enabled          bool         `json:"enabled"`
	ChainID          string       `json:"chain_id"`
	TipHeight        uint64       `json:"tip_height"`
	Interval         uint64       `json:"interval"`
	FinalityDepth    uint64       `json:"finality_depth"`
	LastEndHeight    uint64       `json:"last_end_height"`
	LastSubmitStatus SubmitStatus `json:"last_submit_status,omitempty"`
	LastError        string       `json:"last_error,omitempty"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type Runtime struct {
	mu             sync.RWMutex
	source         RuntimeSource
	builder        *Builder
	poster         *Poster
	signers        map[string]*ecdsa.PrivateKey
	status         RuntimeStatus
	work           chan uint64
	registrationMu sync.Mutex
	registered     bool
}

func NewRuntime(source RuntimeSource, builder *Builder, poster *Poster, signers map[string]*ecdsa.PrivateKey) (*Runtime, error) {
	if source == nil || builder == nil || poster == nil {
		return nil, fmt.Errorf("checkpoint source, builder, and poster are required")
	}
	interval, finality := builder.Config()
	r := &Runtime{source: source, builder: builder, poster: poster, signers: signers, work: make(chan uint64, 1)}
	r.status = RuntimeStatus{Enabled: true, ChainID: source.GetChainID(), Interval: interval, FinalityDepth: finality, UpdatedAt: time.Now().UTC()}
	stored, err := builder.LoadStored(source.GetChainID())
	if err != nil {
		return nil, fmt.Errorf("restore checkpoints: %w", err)
	}
	if len(stored) > 0 {
		latest := stored[len(stored)-1]
		builder.SetProgress(latest.EndHeight, latest.Digest())
		r.status.LastEndHeight = latest.EndHeight
		if state, stateErr := poster.SubmitStatusOf(latest); stateErr == nil {
			r.status.LastSubmitStatus = state
		}
	}
	go r.loop()
	go func() { _ = r.ensureRegistration() }()
	return r, nil
}

func (r *Runtime) ensureRegistration() error {
	r.registrationMu.Lock()
	defer r.registrationMu.Unlock()
	if r.registered {
		return nil
	}
	reg, err := NewSignedRegistration(r.source.GetChainID(), r.signers)
	if err != nil {
		r.fail(err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.poster.RegisterChain(ctx, reg); err != nil && !strings.Contains(err.Error(), "already registered") {
		r.fail(fmt.Errorf("register checkpoint chain: %w", err))
		return err
	}
	r.registered = true
	return nil
}

// OnBlockCommitted is non-blocking and coalesces bursts to the newest tip.
func (r *Runtime) OnBlockCommitted(height uint64) {
	select {
	case r.work <- height:
	default:
		select {
		case <-r.work:
		default:
		}
		select {
		case r.work <- height:
		default:
		}
	}
}

func (r *Runtime) loop() {
	for height := range r.work {
		r.process(height)
	}
}

func (r *Runtime) process(height uint64) {
	r.update(func(s *RuntimeStatus) { s.TipHeight = height; s.UpdatedAt = time.Now().UTC() })
	if err := r.ensureRegistration(); err != nil {
		return
	}
	cp, err := r.builder.OnNewBlock(height)
	if err != nil || cp == nil {
		if err != nil {
			r.fail(err)
		}
		return
	}
	if err := r.builder.Finalize(cp, r.signers); err != nil {
		r.fail(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	out, err := r.poster.PostCheckpoint(ctx, cp)
	cancel()
	if err != nil {
		r.update(func(s *RuntimeStatus) {
			s.LastEndHeight = cp.EndHeight
			s.LastSubmitStatus = SubmitFailed
			s.LastError = err.Error()
			s.UpdatedAt = time.Now().UTC()
		})
		return
	}
	r.update(func(s *RuntimeStatus) {
		s.LastEndHeight = cp.EndHeight
		s.LastSubmitStatus = SubmitSubmitted
		s.LastError = ""
		s.UpdatedAt = time.Now().UTC()
	})
	go r.proveAndVerify(cp, out)
}

func (r *Runtime) proveAndVerify(cp *Checkpoint, oracleResponse map[string]interface{}) {
	blocks, err := r.source.BlocksRange(cp.StartHeight, cp.EndHeight)
	if err != nil {
		r.fail(fmt.Errorf("load proof batch: %w", err))
		return
	}
	var preRoot [32]byte
	if cp.StartHeight > 1 {
		preRoot, err = r.source.AccumRootAt(cp.StartHeight - 1)
		if err != nil {
			r.fail(fmt.Errorf("load proof pre-root: %w", err))
			return
		}
	}
	proof, err := (&HashchainProver{ChainID: cp.ChainID, PreRoot: preRoot}).Prove(blocks)
	if err != nil {
		r.fail(err)
		return
	}
	position, ok := numberAsUint64(oracleResponse["mmr_position"])
	if !ok {
		r.fail(fmt.Errorf("Oracle response missing mmr_position"))
		return
	}
	leafText, _ := oracleResponse["leaf_hash"].(string)
	leafBytes, err := hex.DecodeString(strings.TrimPrefix(leafText, "0x"))
	if err != nil || len(leafBytes) != 32 {
		r.fail(fmt.Errorf("Oracle response has invalid leaf_hash"))
		return
	}
	var leaf [32]byte
	copy(leaf[:], leafBytes)
	request := map[string]interface{}{
		"checkpoint":       map[string]interface{}{"chain_id": cp.ChainID, "start_height": cp.StartHeight, "end_height": cp.EndHeight, "root": cp.Root},
		"transition_proof": proof,
		"mmr":              map[string]interface{}{"leaf_index": position, "leaf_hash": leaf},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := r.poster.PostToLocalVerifier(ctx, request); err != nil {
		r.fail(fmt.Errorf("local verifier: %w", err))
	}
}

func numberAsUint64(value interface{}) (uint64, bool) {
	switch v := value.(type) {
	case float64:
		return uint64(v), v >= 0 && v == float64(uint64(v))
	case uint64:
		return v, true
	case int:
		return uint64(v), v >= 0
	}
	return 0, false
}

func (r *Runtime) fail(err error) {
	r.update(func(s *RuntimeStatus) { s.LastError = err.Error(); s.UpdatedAt = time.Now().UTC() })
}
func (r *Runtime) update(fn func(*RuntimeStatus)) { r.mu.Lock(); defer r.mu.Unlock(); fn(&r.status) }
func (r *Runtime) Status() RuntimeStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.status
	out.TipHeight = r.source.TipHeight()
	return out
}
