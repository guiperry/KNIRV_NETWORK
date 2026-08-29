/**
 * Browser-safe facade for KNIRVSERVER's actuarial API. The Rust core owns the
 * matching service; this facade is intentionally transport-only for WASM apps.
 */
export declare class ActuarialClient {
    private readonly baseUrl;
    private readonly fetcher;
    private headers;
    constructor(baseUrl?: string, fetcher?: typeof fetch);
    setHeaders(headers: HeadersInit): this;
    private request;
    riskClasses<T = unknown>(): Promise<T>;
    pools<T = unknown>(): Promise<T>;
    pool<T = unknown>(id: string): Promise<T>;
    submission<T = unknown>(id: string): Promise<T>;
    submissions<T = unknown>(query?: {
        resolver_wallet?: string;
        status?: string;
    }): Promise<T>;
    decision<T = unknown>(id: string): Promise<T>;
    settlement<T = unknown>(id: string): Promise<T>;
    createSubmission<T = unknown>(body: unknown): Promise<T>;
    createStake<T = unknown>(poolID: string, body: unknown): Promise<T>;
    requestStakeExit<T = unknown>(stakeID: string, body: unknown): Promise<T>;
    createSubmissionArtifacts<T = unknown>(submissionID: string, body: unknown): Promise<T>;
    requestQuote<T = unknown>(submissionID: string, body: unknown): Promise<T>;
    claimSubmission<T = unknown>(id: string, request: {
        resolver_wallet: string;
        signed_intent?: unknown;
    }): Promise<T>;
    createPayoutDestination<T = unknown>(body: unknown): Promise<T>;
    createRiskClass<T = unknown>(body: unknown): Promise<T>;
    createEnterpriseCredential<T = unknown>(body: unknown): Promise<T>;
    exposure<T = unknown>(wallet?: string): Promise<T>;
    poolReport<T = unknown>(id: string): Promise<T>;
}
