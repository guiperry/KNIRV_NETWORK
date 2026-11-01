/**
 * Facade for KNIRVSERVER's actuarial API. The endpoint table, request
 * construction, retries and error mapping live in the Rust core
 * (`ActuarialService::call`, core/src/actuarial.rs); this class only names the
 * operations and moves bytes through the caller's `fetch`.
 */
import { fetchHost, type BindingConfig, type SdkClient } from "./runtime.js";

type Headers_ = Record<string, string>;

export class ActuarialClient {
  private headers: Headers_ = {};
  private sdk?: Promise<SdkClient>;

  constructor(private readonly baseUrl = "", private readonly fetcher: typeof fetch = (...args) => fetch(...args)) {}

  setHeaders(headers: HeadersInit) {
    const collected: Headers_ = {};
    new Headers(headers).forEach((value, name) => { collected[name] = value; });
    this.headers = collected;
    return this;
  }

  private async client(): Promise<SdkClient> {
    // Imported lazily: index.js re-exports this module.
    this.sdk ??= import("./index.js").then(({ SdkClient }) => SdkClient.init(undefined, { http: fetchHost(this.fetcher) }));
    return this.sdk;
  }

  private config(): BindingConfig {
    const origin = (globalThis as { location?: { origin?: string } }).location?.origin;
    const base = new URL(this.baseUrl || "/", origin).toString().replace(/\/$/, "");
    return { backend_url: base, headers: this.headers };
  }

  private async call<T>(op: string, params: Record<string, unknown> = {}): Promise<T> {
    const sdk = await this.client();
    const response = await sdk.call<T>("actuarial.call", { op, ...params }, this.config());
    if (response.error) {
      let message = response.error.message;
      try {
        const body = JSON.parse(message) as { error?: string };
        if (body?.error) message = body.error;
      } catch {
        // not a JSON error body
      }
      throw new Error(message || `actuarial request failed (${response.error.http_status ?? response.error.code})`);
    }
    return response.payload as T;
  }

  riskClasses<T = unknown>() { return this.call<T>("risk_classes"); }
  pools<T = unknown>() { return this.call<T>("pools"); }
  pool<T = unknown>(id: string) { return this.call<T>("pool", { id }); }
  submission<T = unknown>(id: string) { return this.call<T>("submission", { id }); }
  submissions<T = unknown>(query: { resolver_wallet?: string; status?: string } = {}) { return this.call<T>("submissions", { query }); }
  decision<T = unknown>(id: string) { return this.call<T>("decision", { id }); }
  settlement<T = unknown>(id: string) { return this.call<T>("settlement", { id }); }
  createSubmission<T = unknown>(body: unknown) { return this.call<T>("create_submission", { body }); }
  createStake<T = unknown>(poolID: string, body: unknown) { return this.call<T>("create_stake", { id: poolID, body }); }
  requestStakeExit<T = unknown>(stakeID: string, body: unknown) { return this.call<T>("request_stake_exit", { id: stakeID, body }); }
  createSubmissionArtifacts<T = unknown>(submissionID: string, body: unknown) { return this.call<T>("create_submission_artifacts", { id: submissionID, body }); }
  requestQuote<T = unknown>(submissionID: string, body: unknown) { return this.call<T>("request_quote", { id: submissionID, body }); }
  claimSubmission<T = unknown>(id: string, request: { resolver_wallet: string; signed_intent?: unknown }) { return this.call<T>("claim_submission", { id, body: request }); }
  createPayoutDestination<T = unknown>(body: unknown) { return this.call<T>("create_payout_destination", { body }); }
  createRiskClass<T = unknown>(body: unknown) { return this.call<T>("create_risk_class", { body }); }
  createEnterpriseCredential<T = unknown>(body: unknown) { return this.call<T>("create_enterprise_credential", { body }); }
  exposure<T = unknown>(wallet?: string) { return this.call<T>("exposure", { query: wallet ? { wallet } : {} }); }
  poolReport<T = unknown>(id: string) { return this.call<T>("pool_report", { id }); }
}
