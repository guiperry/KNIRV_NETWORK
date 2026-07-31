import { useState, useCallback, useEffect } from 'react';

export interface FintechStatus {
  enabled: boolean;
  running: boolean;
  version: string;
  ontologies: number;
  compliance_features: {
    kyc: boolean;
    aml: boolean;
    sec: boolean;
    basel: boolean;
    scenarios: boolean;
  };
}

// ------------------------------------------------------------------
// Type definitions matching the fintech API response shapes
// ------------------------------------------------------------------

export interface NRVTrace {
  id: string;
  agent_id: string;
  agent_name?: string;
  validation_id?: string;
  status: string;
  step_count: number;
  fidelity_score?: number;
  created_at: string;
  steps?: unknown[];
}

export interface FidelityScore {
  trace_id: string;
  overall_score: number;
  semantic_distance: number;
  regulatory_alignment: number;
  intent_accuracy: number;
  decision_quality: number;
}

export interface SemanticDistance {
  trace_id: string;
  distance: number;
  expected_outcome: string;
}

export interface KYCBypassDetection {
  trace_id: string;
  indicators: unknown[];
  has_violations: boolean;
}

export interface EvidencePack {
  id: string;
  agent_id: string;
  type: string;
  status: string;
  created_at: string;
}

export interface ExecutionTrajectory {
  id: string;
  agent_id: string;
  status: string;
  event_count: number;
  created_at: string;
}

export interface ValidationResult {
  validation_id: string;
  status: string;
  is_compliant: boolean;
  overall_score: number;
}

export interface PositionLimitViolation {
  trace_id: string;
  indicators: unknown[];
  has_violations: boolean;
}

const defaultStatus: FintechStatus = {
  enabled: false,
  running: false,
  version: '1.0.0',
  ontologies: 0,
  compliance_features: {
    kyc: false,
    aml: false,
    sec: false,
    basel: false,
    scenarios: false,
  },
};

function useFinTechValidator() {
  const [status, setStatus] = useState<FintechStatus>(defaultStatus);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [evidencePacks, setEvidencePacks] = useState<EvidencePack[]>([]);
  const [nrvTraces, setNrvTraces] = useState<NRVTrace[]>([]);
  const [selectedTrace, setSelectedTrace] = useState<NRVTrace | null>(null);
  const [fidelityScore, setFidelityScore] = useState<FidelityScore | null>(null);
  const [semanticDistance, setSemanticDistance] = useState<SemanticDistance | null>(null);
  const [kycBypassDetection, setKycBypassDetection] = useState<KYCBypassDetection | null>(null);
  const [positionLimitViolations, setPositionLimitViolations] = useState<PositionLimitViolation | null>(null);
  const [trajectories, setTrajectories] = useState<ExecutionTrajectory[]>([]);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);

  // Fetch plugin status (always available regardless of enabled state)
  const fetchStatus = useCallback(async () => {
    try {
      const res = await fetch('/api/fintech/status');
      if (res.ok) {
        const data: FintechStatus = await res.json();
        setStatus(data);
      }
    } catch (e) {
      console.error('Failed to fetch fintech status', e);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchStatus();
    const interval = window.setInterval(() => void fetchStatus(), 15000);
    return () => window.clearInterval(interval);
  }, [fetchStatus]);

  const isPluginEnabled = status.enabled && status.running;

  const fetchNRVTraces = useCallback(async () => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/fintech/nrv/traces');
      if (!res.ok) throw new Error(`Failed to fetch NRV traces: ${res.statusText}`);
      const data = await res.json();
      setNrvTraces(data.traces ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const fetchNRVTrace = useCallback(async (traceId: string) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fintech/nrv/traces/${traceId}`);
      if (!res.ok) throw new Error(`Failed to fetch NRV trace: ${res.statusText}`);
      const data: NRVTrace = await res.json();
      setSelectedTrace(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const fetchEvidencePacks = useCallback(async () => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/fintech/evidence');
      if (!res.ok) throw new Error(`Failed to fetch evidence packs: ${res.statusText}`);
      const data = await res.json();
      setEvidencePacks(data.evidence_packs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const fetchTrajectories = useCallback(async () => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/fintech/trajectories');
      if (!res.ok) throw new Error(`Failed to fetch trajectories: ${res.statusText}`);
      const data = await res.json();
      setTrajectories(data.trajectories ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const scoreFidelity = useCallback(async (traceId: string, expectedOutcome?: string) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fintech/nrv/traces/${traceId}/score`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ expected_outcome: expectedOutcome ?? '' }),
      });
      if (!res.ok) throw new Error(`Failed to score fidelity: ${res.statusText}`);
      const data = await res.json();
      setFidelityScore(data.fidelity ?? null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const calculateSemanticDistance = useCallback(async (traceId: string, expectedOutcome?: string) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fintech/nrv/traces/${traceId}/distance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ expected_outcome: expectedOutcome ?? '' }),
      });
      if (!res.ok) throw new Error(`Failed to calculate distance: ${res.statusText}`);
      const data: SemanticDistance = await res.json();
      setSemanticDistance(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const detectKYCBypass = useCallback(async (traceId: string) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fintech/nrv/traces/${traceId}/detect/kyc-bypass`);
      if (!res.ok) throw new Error(`Failed to detect KYC bypass: ${res.statusText}`);
      const data: KYCBypassDetection = await res.json();
      setKycBypassDetection(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const detectPositionLimits = useCallback(async (traceId: string, limits?: Record<string, number>) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/fintech/nrv/traces/${traceId}/detect/position-limits`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ limits: limits ?? {} }),
      });
      if (!res.ok) throw new Error(`Failed to detect position limits: ${res.statusText}`);
      const data: PositionLimitViolation = await res.json();
      setPositionLimitViolations(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  const validate = useCallback(async (agentId: string, agentCode?: string) => {
    if (!isPluginEnabled) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/fintech/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, agent_code: agentCode }),
      });
      if (!res.ok) throw new Error(`Validation failed: ${res.statusText}`);
      const data: ValidationResult = await res.json();
      setValidationResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsLoading(false);
    }
  }, [isPluginEnabled]);

  return {
    status,
    isLoading,
    error,
    evidencePacks,
    nrvTraces,
    selectedTrace,
    fidelityScore,
    semanticDistance,
    kycBypassDetection,
    positionLimitViolations,
    trajectories,
    validationResult,
    fetchStatus,
    fetchNRVTraces,
    fetchNRVTrace,
    fetchEvidencePacks,
    fetchTrajectories,
    scoreFidelity,
    calculateSemanticDistance,
    detectKYCBypass,
    detectPositionLimits,
    validate,
    // Convenience derived state for plugin gating
    isPluginEnabled,
  };
}

export default useFinTechValidator;
