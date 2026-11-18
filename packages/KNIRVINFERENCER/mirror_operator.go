package inferencer

import (
	"context"
	"fmt"
	"time"
)

// Operator generation through the mirror path (KNIRV_MIRRORED_GENERATION=1).
//
// InferenceService reads the flag once at construction. When it is on, default
// generation requests (no model and no instruction) run GenerateMirrored through the
// harness's HEAD model instead of the routed Harness.Do(KindGenerate) chain. When it is
// off, nothing in this file runs and the existing path is unchanged.
//
// Caching rule: a mirrored result is written to the response cache only when its winner
// was verified by HASHER. A REFUTED, unverified, degraded or errored winner is returned
// to the caller but never cached. A fallback to the routed chain (no mirror candidate)
// is not arbitrated or verified, so it is not cached either while the flag is on.

// envMirroredGeneration is the environment variable that routes operator generation
// through the mirror path. Only the value "1" turns it on.
const envMirroredGeneration = "KNIRV_MIRRORED_GENERATION"

// MirrorGenerate runs one mirrored generation through the harness. It returns the
// winning text and whether that text may be written to the response cache. Every call
// appends a Trace whose Mirror field holds the MirrorTrace, including calls that fail.
func (h *Harness) MirrorGenerate(ctx context.Context, prompt string) (string, bool, error) {
	start := time.Now()
	tr := Trace{ID: newTraceID(), Time: start.UTC(), Kind: KindGenerate}
	h.bumpMirror(func(c *MirrorCounters) { c.Runs++ })

	m, registered := h.Model("head")
	hm, isHead := m.(*HeadModel)
	if !registered || !isHead || hm.client == nil {
		err := fmt.Errorf("%w: mirror generation needs the head model", ErrNoRoute)
		tr.Error = err.Error()
		h.bumpMirror(func(c *MirrorCounters) { c.Unavailable++ })
		h.finish(Result{}, tr, start, err)
		return "", false, err
	}

	mr, mt, err := runMirrored(ctx, hm.client, prompt, "", nil, "", nil)
	tr.Mirror = mt
	h.noteMirrorCandidates(mt)
	if err != nil {
		tr.Error = err.Error()
		h.bumpMirror(func(c *MirrorCounters) { c.NoCandidate++ })
		h.finish(Result{}, tr, start, err)
		return "", false, err
	}

	cacheable := mirrorCacheable(mr)
	tr.Chosen, tr.Version = mr.Winner, mr.Version
	tr.Reason = "mirror arbitration: " + mr.Winner
	tr.Cached = cacheable
	h.bumpMirror(func(c *MirrorCounters) {
		c.Arbitrated++
		if mr.Winner == "head" {
			c.WonHead++
		} else {
			c.WonHeart++
		}
		if !cacheable {
			c.NotCached++
		}
	})
	h.finish(Result{Text: mr.Text, Model: mr.Winner, Version: mr.Version}, tr, start, nil)
	return mr.Text, cacheable, nil
}

// mirrorCacheable reports whether the arbitrated winner's text may be cached: the winner
// must have a HASHER verdict that is neither REFUTED, unverified, degraded nor errored.
// A result without a trace is never cacheable.
func mirrorCacheable(mr *MirroredResult) bool {
	if mr == nil || mr.Trace == nil || mr.Refuted {
		return false
	}
	for _, c := range mr.Trace.Candidates {
		if c.Model != mr.Winner {
			continue
		}
		return c.Verified && !c.Refuted && !c.Unverified && !c.Degraded && c.Error == ""
	}
	return false
}

// noteMirrorCandidates counts what HASHER said about each candidate of one mirror run.
func (h *Harness) noteMirrorCandidates(mt *MirrorTrace) {
	if mt == nil {
		return
	}
	h.bumpMirror(func(c *MirrorCounters) {
		for _, cand := range mt.Candidates {
			if cand.Refuted {
				c.RefutedCandidates++
			}
			if cand.Unverified {
				c.UnverifiedCandidates++
			}
			if cand.Degraded {
				c.DegradedCandidates++
			}
		}
	})
}

// noteMirrorFallback counts a mirror failure that was answered by the routed chain.
func (h *Harness) noteMirrorFallback() {
	h.bumpMirror(func(c *MirrorCounters) { c.FallbackToRouted++ })
}

func (h *Harness) bumpMirror(f func(*MirrorCounters)) {
	h.mirrorMu.Lock()
	defer h.mirrorMu.Unlock()
	f(&h.mirrorCounts)
}
