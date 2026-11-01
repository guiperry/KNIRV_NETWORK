/**
 * Facade for KNIRVSERVER's actuarial API. The endpoint table, request
 * construction, retries and error mapping live in the Rust core
 * (`ActuarialService::call`, core/src/actuarial.rs); this class only names the
 * operations and moves bytes through the caller's `fetch`.
 */
import { fetchHost } from "./runtime.js";
export class ActuarialClient {
    baseUrl;
    fetcher;
    headers = {};
    sdk;
    constructor(baseUrl = "", fetcher = (...args) => fetch(...args)) {
        this.baseUrl = baseUrl;
        this.fetcher = fetcher;
    }
    setHeaders(headers) {
        const collected = {};
        new Headers(headers).forEach((value, name) => { collected[name] = value; });
        this.headers = collected;
        return this;
    }
    async client() {
        // Imported lazily: index.js re-exports this module.
        this.sdk ??= import("./index.js").then(({ SdkClient }) => SdkClient.init(undefined, { http: fetchHost(this.fetcher) }));
        return this.sdk;
    }
    config() {
        const origin = globalThis.location?.origin;
        const base = new URL(this.baseUrl || "/", origin).toString().replace(/\/$/, "");
        return { backend_url: base, headers: this.headers };
    }
    async call(op, params = {}) {
        const sdk = await this.client();
        const response = await sdk.call("actuarial.call", { op, ...params }, this.config());
        if (response.error) {
            let message = response.error.message;
            try {
                const body = JSON.parse(message);
                if (body?.error)
                    message = body.error;
            }
            catch {
                // not a JSON error body
            }
            throw new Error(message || `actuarial request failed (${response.error.http_status ?? response.error.code})`);
        }
        return response.payload;
    }
    riskClasses() { return this.call("risk_classes"); }
    pools() { return this.call("pools"); }
    pool(id) { return this.call("pool", { id }); }
    submission(id) { return this.call("submission", { id }); }
    submissions(query = {}) { return this.call("submissions", { query }); }
    decision(id) { return this.call("decision", { id }); }
    settlement(id) { return this.call("settlement", { id }); }
    createSubmission(body) { return this.call("create_submission", { body }); }
    createStake(poolID, body) { return this.call("create_stake", { id: poolID, body }); }
    requestStakeExit(stakeID, body) { return this.call("request_stake_exit", { id: stakeID, body }); }
    createSubmissionArtifacts(submissionID, body) { return this.call("create_submission_artifacts", { id: submissionID, body }); }
    requestQuote(submissionID, body) { return this.call("request_quote", { id: submissionID, body }); }
    claimSubmission(id, request) { return this.call("claim_submission", { id, body: request }); }
    createPayoutDestination(body) { return this.call("create_payout_destination", { body }); }
    createRiskClass(body) { return this.call("create_risk_class", { body }); }
    createEnterpriseCredential(body) { return this.call("create_enterprise_credential", { body }); }
    exposure(wallet) { return this.call("exposure", { query: wallet ? { wallet } : {} }); }
    poolReport(id) { return this.call("pool_report", { id }); }
}
