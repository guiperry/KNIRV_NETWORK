/**
 * Browser-safe facade for KNIRVSERVER's actuarial API. The Rust core owns the
 * matching service; this facade is intentionally transport-only for WASM apps.
 */
export class ActuarialClient {
    baseUrl;
    fetcher;
    headers = {};
    constructor(baseUrl = "", fetcher = fetch) {
        this.baseUrl = baseUrl;
        this.fetcher = fetcher;
    }
    setHeaders(headers) { this.headers = headers; return this; }
    request(path, init) {
        const headers = new Headers(this.headers);
        headers.set("content-type", "application/json");
        new Headers(init?.headers).forEach((value, name) => headers.set(name, value));
        return this.fetcher(`${this.baseUrl}/api/v1/actuarial${path}`, {
            ...init,
            headers,
        }).then(async (response) => {
            const body = await response.json();
            if (!response.ok)
                throw new Error(body?.error ?? `actuarial request failed (${response.status})`);
            return body;
        });
    }
    riskClasses() { return this.request("/risk-classes"); }
    pools() { return this.request("/pools"); }
    pool(id) { return this.request(`/pools/${encodeURIComponent(id)}`); }
    submission(id) { return this.request(`/submissions/${encodeURIComponent(id)}`); }
    decision(id) { return this.request(`/decisions/${encodeURIComponent(id)}`); }
    settlement(id) { return this.request(`/settlements/${encodeURIComponent(id)}`); }
    createSubmission(body) { return this.request("/submissions", { method: "POST", body: JSON.stringify(body) }); }
    createStake(poolID, body) { return this.request(`/pools/${encodeURIComponent(poolID)}/stakes`, { method: "POST", body: JSON.stringify(body) }); }
    requestStakeExit(stakeID, body) { return this.request(`/stakes/${encodeURIComponent(stakeID)}/exit`, { method: "POST", body: JSON.stringify(body) }); }
    createSubmissionArtifacts(submissionID, body) { return this.request(`/submissions/${encodeURIComponent(submissionID)}/artifacts`, { method: "POST", body: JSON.stringify(body) }); }
    requestQuote(submissionID, body) { return this.request(`/submissions/${encodeURIComponent(submissionID)}/quote`, { method: "POST", body: JSON.stringify(body) }); }
    claimSubmission(id, request) { return this.request(`/submissions/${encodeURIComponent(id)}/claim`, { method: "POST", body: JSON.stringify(request) }); }
    createPayoutDestination(body) { return this.request("/payout-destinations", { method: "POST", body: JSON.stringify(body) }); }
    createRiskClass(body) { return this.request("/risk-classes", { method: "POST", body: JSON.stringify(body) }); }
    createEnterpriseCredential(body) { return this.request("/credentials/enterprise", { method: "POST", body: JSON.stringify(body) }); }
    exposure(wallet) {
        const query = wallet ? `?wallet=${encodeURIComponent(wallet)}` : '';
        return this.request(`/me/exposure${query}`);
    }
    poolReport(id) { return this.request(`/reports/pools/${encodeURIComponent(id)}`); }
}
