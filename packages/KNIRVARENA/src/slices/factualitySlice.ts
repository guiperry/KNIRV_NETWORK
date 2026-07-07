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
}

// Simple factuality slice stub: produce the standardized schema with mock citations
export function createFactualitySlice(questionOrText: string, context: Record<string, unknown> = {}): FactualitySlice {
  const answerText = questionOrText.length > 0 ? `Stub answer for: ${questionOrText}` : 'No answer';
  const citations: Citation[] = [
    { id: 'source:user:1', source: 'user', snippet: questionOrText.slice(0, 140), score: 0.6 },
    { id: 'source:context:1', source: 'context', snippet: JSON.stringify(context).slice(0, 140), score: 0.4 }
  ];

  const response: FactualityAnswer = {
    answer: answerText,
    citations: citations.map(c => c.id),
    confidence: 0.85,
    refused: false,
    hallucination_risk: 0.05,
    evidence_quality_score: 0.6
  };

  return {
    response,
    citations,
    provenance: { generatedBy: 'factuality-slice-stub', model: 'stub-v0', timestamp: Date.now() }
  };
}
