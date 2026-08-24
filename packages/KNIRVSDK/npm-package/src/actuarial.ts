/**
 * Browser-safe facade for KNIRVSERVER's actuarial API. The Rust core owns the
 * matching service; this facade is intentionally transport-only for WASM apps.
 */
export class ActuarialClient {
  private headers: HeadersInit = {};
  constructor(private readonly baseUrl = "", private readonly fetcher: typeof fetch = fetch) {}
  setHeaders(headers: HeadersInit) { this.headers = headers; return this; }

  private request<T>(path: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(this.headers);
    headers.set("content-type", "application/json");
    new Headers(init?.headers).forEach((value, name) => headers.set(name, value));
    return this.fetcher(`${this.baseUrl}/api/v1/actuarial${path}`, {
      ...init,
      headers,
    }).then(async response => {
      const body = await response.json();
      if (!response.ok) throw new Error(body?.error ?? `actuarial request failed (${response.status})`);
      return body as T;
    });
  }
  riskClasses<T = unknown>() { return this.request<T>("/risk-classes"); }
  pools<T = unknown>() { return this.request<T>("/pools"); }
  pool<T = unknown>(id: string) { return this.request<T>(`/pools/${encodeURIComponent(id)}`); }
  submission<T = unknown>(id: string) { return this.request<T>(`/submissions/${encodeURIComponent(id)}`); }
  decision<T = unknown>(id: string) { return this.request<T>(`/decisions/${encodeURIComponent(id)}`); }
  settlement<T = unknown>(id: string) { return this.request<T>(`/settlements/${encodeURIComponent(id)}`); }
  createSubmission<T = unknown>(body: unknown) { return this.request<T>("/submissions", { method: "POST", body: JSON.stringify(body) }); }
  createStake<T = unknown>(poolID: string, body: unknown) { return this.request<T>(`/pools/${encodeURIComponent(poolID)}/stakes`, { method: "POST", body: JSON.stringify(body) }); }
  requestStakeExit<T = unknown>(stakeID: string, body: unknown) { return this.request<T>(`/stakes/${encodeURIComponent(stakeID)}/exit`, { method: "POST", body: JSON.stringify(body) }); }
  createSubmissionArtifacts<T = unknown>(submissionID: string, body: unknown) { return this.request<T>(`/submissions/${encodeURIComponent(submissionID)}/artifacts`, { method: "POST", body: JSON.stringify(body) }); }
  requestQuote<T = unknown>(submissionID: string, body: unknown) { return this.request<T>(`/submissions/${encodeURIComponent(submissionID)}/quote`, { method: "POST", body: JSON.stringify(body) }); }
  claimSubmission<T = unknown>(id: string, request: { resolver_wallet: string; signed_intent?: unknown }) { return this.request<T>(`/submissions/${encodeURIComponent(id)}/claim`, { method: "POST", body: JSON.stringify(request) }); }
  createPayoutDestination<T = unknown>(body: unknown) { return this.request<T>("/payout-destinations", { method: "POST", body: JSON.stringify(body) }); }
  createRiskClass<T = unknown>(body: unknown) { return this.request<T>("/risk-classes", { method: "POST", body: JSON.stringify(body) }); }
  createEnterpriseCredential<T = unknown>(body: unknown) { return this.request<T>("/credentials/enterprise", { method: "POST", body: JSON.stringify(body) }); }
  exposure<T = unknown>(wallet?: string) {
    const query = wallet ? `?wallet=${encodeURIComponent(wallet)}` : '';
    return this.request<T>(`/me/exposure${query}`);
  }
  poolReport<T = unknown>(id: string) { return this.request<T>(`/reports/pools/${encodeURIComponent(id)}`); }
}
