export interface FactualAssertion {
  text: string;
  confidence: number;
  evidence: Array<{ source: string; snippet: string; score: number }>;
}

export interface FactualitySlice {
  assertions: FactualAssertion[];
  provenance: { generatedBy: string; timestamp: number };
}

// Simple factuality slice stub: extract short sentences and create mock evidence
export function createFactualitySlice(text: string, context: Record<string, unknown> = {}): FactualitySlice {
  const sentences = text.split(/\.|\n|!/).map(s => s.trim()).filter(Boolean).slice(0, 5);

  const assertions = sentences.map((s, i) => ({
    text: s,
    confidence: Math.max(0.1, Math.min(0.99, 1 - i * 0.15)),
    evidence: [
      { source: 'user', snippet: s.slice(0, 120), score: 0.6 },
      { source: 'context', snippet: JSON.stringify(context).slice(0, 120), score: 0.4 }
    ]
  }));

  return {
    assertions,
    provenance: { generatedBy: 'factuality-slice-stub', timestamp: Date.now() }
  };
}
