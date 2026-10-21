import { createFactualitySlice } from '../../../src/slices/factualitySlice';

describe('createFactualitySlice', () => {
  it('does not fabricate an assessment', () => {
    const slice = createFactualitySlice('The sky is green', { serverType: 'mcp' });
    expect(slice.verified).toBe(false);
    expect(slice.response.refused).toBe(true);
    expect(slice.response.confidence).toBe(0);
    expect(slice.response.citations).toEqual([]);
    expect(slice.citations).toEqual([]);
    expect(slice.response.hallucination_risk).toBeUndefined();
    expect(slice.response.evidence_quality_score).toBeUndefined();
  });
});
