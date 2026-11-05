package factslice

import (
	"context"
	"encoding/json"
	"testing"

	"knirvmodels/pkg/evidence"
)

type okWitnesser struct{}

func (okWitnesser) Witness(_ context.Context, e evidence.WitnessEntry) (evidence.WitnessAttestation, error) {
	return evidence.WitnessAttestation{Kind: "test", Ref: "assertion-" + e.BatchID, Detail: json.RawMessage(`{}`)}, nil
}

// The attested rung reads the index's own provenance: a span counts once its batch root has
// been witnessed, and not before.
func TestAClaimReachesAttestedAndAcceptedOnlyAfterItsRootsAreWitnessed(t *testing.T) {
	ix := testIndex(t)
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Witness = ix.Attested

	r := e.CheckClaim(context.Background(), parisClaim(), claimReq())
	if r.Verdict != Supported || r.Witnessed || r.Gate != GateSeen {
		t.Fatalf("before witnessing: %+v", r)
	}
	if n, err := ix.Witnesses().Drain(context.Background(), okWitnesser{}, 0); err != nil || n == 0 {
		t.Fatalf("%d %v", n, err)
	}
	r = e.CheckClaim(context.Background(), parisClaim(), claimReq())
	if !r.Witnessed || r.Gate != GateAccepted {
		t.Fatalf("after witnessing: %+v", r)
	}
}

// Done-when: a quarantine-only claim is capped below attested until its batch is witnessed.
func TestAQuarantineOnlyClaimIsCappedBelowAttestedUntilItsBatchIsWitnessed(t *testing.T) {
	ix := testIndexEmpty(t)
	for _, d := range []evidence.Doc{
		{ID: "n", Title: "NASA", URL: "https://www.nasa.gov/moon", Text: "The Moon is 384400 kilometres from the Earth according to NASA."},
		{ID: "e", Title: "ESA", URL: "https://www.esa.int/moon", Text: "The Moon is 384400 kilometres from the Earth according to ESA."},
	} {
		if _, err := ix.Ingest(evidence.Batch{Tier: evidence.TierQuarantine, Docs: []evidence.Doc{d}}); err != nil {
			t.Fatal(err)
		}
	}
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Witness = ix.Attested
	req := Request{K: 6, Tiers: []evidence.Tier{evidence.TierQuarantine}}

	r := e.CheckClaim(context.Background(), moonClaim(), req)
	if r.Verdict != Supported || r.ReferenceTier || r.Witnessed || r.Gate >= GateAttested {
		t.Fatalf("unwitnessed web evidence must stay below attested: %+v", r)
	}
	// The quarantine batches are not due until their schedule; witness them explicitly.
	for _, h := range r.Evidence {
		p, err := ix.InclusionProof(h.SpanID)
		if err != nil {
			t.Fatal(err)
		}
		if err := ix.Witnesses().MarkWitnessed(p.Root, evidence.WitnessAttestation{Kind: "test", Ref: p.BatchID}); err != nil {
			t.Fatal(err)
		}
	}
	r = e.CheckClaim(context.Background(), moonClaim(), req)
	if !r.Witnessed || r.Gate != GateAttested {
		t.Fatalf("witnessed web evidence reaches attested but not accepted (it is not reference tier): %+v", r)
	}
}

func TestWithoutAWitnesserNothingIsAttested(t *testing.T) {
	ix := testIndex(t)
	e := engineFor(ix, citing("SUPPORTED", 0.97), fittedIdentity())
	e.Witness = ix.Attested // the queue is full of pending roots and nothing drains it
	if r := e.CheckClaim(context.Background(), parisClaim(), claimReq()); r.Witnessed || r.Gate > GateSeen {
		t.Fatalf("%+v", r)
	}
}
