export interface EnrollmentVectorResult {
    passed: boolean;
    skipped: boolean;
    reason?: string;
}
export declare function runBootnodeEnrollmentVectors(): Promise<EnrollmentVectorResult>;
