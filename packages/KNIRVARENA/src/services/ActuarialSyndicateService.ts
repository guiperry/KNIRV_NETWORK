export type BountyDomain = 'security_exploit' | 'code_error';

export interface BountyPosting {
  id: string;
  display_name: string;
  description: string;
  domain: BountyDomain;
  status: 'observation_only' | 'active' | 'retired';
  eligibility_policy?: Record<string, string>;
  curated_challenge?: {
    legacy_id: string;
    type: string;
    buggy_code: string;
    context: string;
    hints: string[];
  };
}

export interface SyndicatePool {
  id: string;
  risk_class_id: string;
  currency: string;
  rail: string;
  status: string;
  total_stake: number;
  liquid_balance: number;
  reserved_balance: number;
}

export interface StakeExposure {
  stake: {
    id: string;
    pool_id: string;
    operator_wallet: string;
    deposited_amount: number;
    locked_amount: number;
    withdrawable_amount: number;
    status: string;
  };
  gross_exposure: number;
  realized_loss: number;
}

export interface ExposureSummary {
  wallet: string;
  positions: StakeExposure[];
  gross_exposure: number;
  realized_loss: number;
}

export interface PoolReport {
  active_stake_count: number;
  available_liquidity: number;
  reserve_entries: Array<{
    entity_type: string;
    amount: number;
    direction: string;
    reference_id: string;
  }>;
}

export interface StakeRequest {
  operator_wallet: string;
  node_id: string;
  credential_commitment: string;
  amount: number;
}

export interface ArtifactRequest {
  researcher_wallet: string;
  poc_reference: string;
  report_reference: string;
  poc_hash: string;
  report_hash: string;
  scope_hash: string;
  dve_id: string;
  dve_session_id: string;
}

export interface QuoteRequest {
  requester_wallet: string;
  model_id: string;
  snapshot_id: string;
  pool_id: string;
  feature_values: Record<string, number>;
  confidence: number;
}

export type SignedIntent = { nonce: string; canonical_payload: string; signature: string };
export type MutationAuthorizer = (
  payload: Record<string, unknown>
) => Promise<{ signedIntent: SignedIntent; headers: Record<string, string>; result?: unknown }>;

/** Shared read/mutation boundary for the Bounty Board. No challenge data is
 * duplicated here: postings are owned by KNIRVSERVER. */
export class ActuarialSyndicateService {
  private readonly client: ActuarialClient;

  constructor(baseUrl?: string) {
    this.client = new ActuarialClient(baseUrl ?? getBackendUrl());
  }

  configureMutationAuthorizer(authorize: MutationAuthorizer) {
    this.authorize = authorize;
  }
  private authorize?: MutationAuthorizer;

  private async mutate<T>(
    payload: Record<string, unknown>,
    execute: (signedIntent: SignedIntent) => Promise<T>,
    relay: { path: string; body: Record<string, unknown> }
  ): Promise<T> {
    if (!this.authorize)
      throw new Error('Connect a KNIRV wallet before submitting or claiming a bounty');
    const authorization = await this.authorize({ ...payload, __knirv_relay: relay });
    if (authorization.result !== undefined) return authorization.result as T;
    this.client.setHeaders(authorization.headers);
    try {
      return await execute(authorization.signedIntent);
    } finally {
      // Headers are request-scoped: never reuse another action's
      // idempotency key or bearer token implicitly.
      this.client.setHeaders({});
    }
  }

  async listPostings(domain?: BountyDomain): Promise<BountyPosting[]> {
    const classes = await this.client.riskClasses<BountyPosting[]>();
    return classes.filter(item => !domain || item.domain === domain);
  }

  listPools(): Promise<SyndicatePool[]> {
    return this.client.pools<SyndicatePool[]>();
  }

  getExposure(wallet: string): Promise<ExposureSummary> {
    return this.client.exposure<ExposureSummary>(wallet);
  }

  getPoolReport(poolID: string): Promise<PoolReport> {
    return this.client.poolReport<PoolReport>(poolID);
  }

  async claim(submissionId: string, resolverWallet: string) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate({
      submission_id: submissionId,
      resolver_wallet: resolverWallet,
      idempotency_key,
    }, signed_intent => this.client.claimSubmission(submissionId, {
      resolver_wallet: resolverWallet,
      signed_intent,
    }), { path: `/submissions/${encodeURIComponent(submissionId)}/claim`, body: { resolver_wallet: resolverWallet } });
  }

  async createResearcherSubmission(input: {
    researcher_credential_id: string;
    researcher_wallet: string;
    claimed_risk_class: string;
    idempotency_key: string;
  }) {
    return this.mutate({
      researcher_wallet: input.researcher_wallet,
      claimed_risk_class: input.claimed_risk_class,
      idempotency_key: input.idempotency_key,
    }, signed_intent => this.client.createSubmission({ ...input, signed_intent }), { path: '/submissions', body: input });
  }

  createStake(poolID: string, input: StakeRequest) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate(
      { pool_id: poolID, operator_wallet: input.operator_wallet, node_id: input.node_id, amount: input.amount, idempotency_key },
      signed_intent => this.client.createStake(poolID, { ...input, signed_intent }),
      { path: `/pools/${encodeURIComponent(poolID)}/stakes`, body: { ...input } }
    );
  }

  requestStakeExit(stakeID: string, operatorWallet: string) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate(
      { stake_id: stakeID, operator_wallet: operatorWallet, idempotency_key },
      signed_intent => this.client.requestStakeExit(stakeID, { operator_wallet: operatorWallet, signed_intent }),
      { path: `/stakes/${encodeURIComponent(stakeID)}/exit`, body: { operator_wallet: operatorWallet } }
    );
  }

  createArtifacts(submissionID: string, input: ArtifactRequest) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate(
      { submission_id: submissionID, researcher_wallet: input.researcher_wallet, poc_hash: input.poc_hash, report_hash: input.report_hash, scope_hash: input.scope_hash, idempotency_key },
      signed_intent => this.client.createSubmissionArtifacts(submissionID, { ...input, signed_intent }),
      { path: `/submissions/${encodeURIComponent(submissionID)}/artifacts`, body: { ...input } }
    );
  }

  requestQuote(submissionID: string, input: QuoteRequest) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate(
      { submission_id: submissionID, ...input, idempotency_key },
      signed_intent => this.client.requestQuote(submissionID, { ...input, signed_intent }),
      { path: `/submissions/${encodeURIComponent(submissionID)}/quote`, body: { ...input } }
    );
  }

  getDecision(decisionID: string) { return this.client.decision(decisionID); }
  getSettlement(settlementID: string) { return this.client.settlement(settlementID); }

  createPayoutDestination(wallet: string, country: string, email: string) {
    const idempotency_key = crypto.randomUUID();
    return this.mutate({ wallet, country, email, idempotency_key }, () =>
      this.client.createPayoutDestination({ wallet, country, email }),
      { path: '/payout-destinations', body: { wallet, country, email } }
    );
  }
}

function getBackendUrl(): string {
  try {
    return eval('import.meta').env?.VITE_KNIRVSERVER_URL ?? 'http://localhost:8082';
  } catch {
    return 'http://localhost:8082';
  }
}

export const actuarialSyndicateService = new ActuarialSyndicateService();
import { ActuarialClient } from '@knirv/sdk/actuarial';
