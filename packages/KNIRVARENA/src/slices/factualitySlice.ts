// Factuality Slice types aligned with factuality_slice_integration.md
export interface Citation {
  id: string; // evidence id or hash
  source: string; // e.g., 'wikipedia', 'arxiv', 'user', 'ipfs'
  snippet: string; // short excerpt used as evidence
  score: number; // confidence/score for this citation
}

export interface FactualityAnswer {
  answer: string;
  citations: string[]; // array of citation ids
  confidence: number; // 0-1
  refused: boolean;
  hallucination_risk?: number; // optional 0-1
  evidence_quality_score?: number; // optional 0-1
}

export interface FactualitySlice {
  response: FactualityAnswer;
  citations: Citation[]; // detailed citation objects
  provenance: { generatedBy: string; model?: string; timestamp: number };
  /** false = nothing has checked this; confidence/citations carry no information. */
  verified?: boolean;
}

// Placeholder until the node's evidence-backed factuality check is wired in (three_layer_inference.md,
// Phase 4). It does NOT assess anything: it returns the standardized schema marked unverified and
// refused, with no citations and no scores. The earlier stub reported confidence 0.85, hallucination
// risk 0.05 and mock citations, which downstream code could mistake for a real assessment (F7).
export function createFactualitySlice(questionOrText: string, context: Record<string, unknown> = {}): FactualitySlice {
  void context;
  const response: FactualityAnswer = {
    answer: questionOrText,
    citations: [],
    confidence: 0,
    refused: true
  };

  return {
    response,
    citations: [],
    provenance: { generatedBy: 'unverified-placeholder', timestamp: Date.now() },
    verified: false
  };
}
