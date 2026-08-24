import { describe, expect, it } from '@jest/globals';
import { selectCuratedChallenge } from '../actuarialChallenge';

describe('selectCuratedChallenge', () => {
  it('prefers the seeded backend challenge matched by legacy ID', () => {
    const result = selectCuratedChallenge(
      [
        {
          id: 'rc-1',
          display_name: 'Backend challenge',
          description: 'server owned',
          domain: 'code_error',
          status: 'active',
          curated_challenge: {
            legacy_id: 'ml-001',
            type: 'Memory Leak',
            buggy_code: 'leak()',
            context: 'fix leak',
            hints: [],
          },
        },
      ],
      'ml-001'
    );
    expect(result).toEqual({
      title: 'Backend challenge',
      description: 'server owned',
      buggyCode: 'leak()',
      context: 'fix leak',
    });
  });
  it('does not select unseeded postings', () =>
    expect(selectCuratedChallenge([], 'ml-001')).toBeUndefined());
});
