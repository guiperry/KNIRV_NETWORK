export type CuratedPosting = {
  id: string;
  display_name: string;
  description: string;
  /** Metadata supplied by the bounty-postings endpoint.  The challenge
   * selector does not use these fields, but accepting the complete posting
   * shape keeps callers from having to discard backend data first. */
  domain?: 'security_exploit' | 'code_error';
  status?: 'observation_only' | 'active' | 'retired';
  curated_challenge?: {
    legacy_id: string;
    type: string;
    buggy_code: string;
    context: string;
    hints: string[];
  };
};
export type CuratedChallengeView = {
  title: string;
  description: string;
  buggyCode: string;
  context: string;
};

export function selectCuratedChallenge(
  postings: CuratedPosting[],
  legacyID?: string
): CuratedChallengeView | undefined {
  if (!legacyID) return undefined;
  const posting = postings.find(
    item => item.curated_challenge?.legacy_id === legacyID || item.id === legacyID
  );
  if (!posting?.curated_challenge) return undefined;
  return {
    title: posting.display_name,
    description: posting.description,
    buggyCode: posting.curated_challenge.buggy_code,
    context: posting.curated_challenge.context,
  };
}
