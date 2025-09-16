export interface FeasibilityReport {
  exists: boolean;
  similar: Array<{ id: string; score: number; summary: string }>;
  feasibilityScore: number;
  provenance: { generatedBy: string; timestamp: number };
}

// Simple feasibility slice stub: compare to a small local token set
export function createFeasibilitySlice(title: string, description: string, existingItems: Array<{ id: string; text: string }> = []): FeasibilityReport {
  // naive similarity: common word overlap
  const norm = (s: string) => s.toLowerCase().replace(/[^a-z0-9\s]/g, '').split(/\s+/).filter(Boolean);
  const descWords = norm(`${title} ${description}`);

  const similar = existingItems.map(item => {
    const itemWords = norm(item.text || '');
    const intersection = descWords.filter(w => itemWords.includes(w)).length;
    const union = new Set([...descWords, ...itemWords]).size || 1;
    const score = intersection / union;
    return { id: item.id, score, summary: item.text.slice(0, 140) };
  }).filter(s => s.score > 0).sort((a,b) => b.score - a.score).slice(0,5);

  const exists = similar.length > 0 && similar[0].score > 0.6;
  const feasibilityScore = Math.round((1 - (similar.length > 0 ? similar[0].score : 0)) * 100);

  return {
    exists,
    similar,
    feasibilityScore,
    provenance: { generatedBy: 'feasibility-slice-stub', timestamp: Date.now() }
  };
}
