"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  EvidencePack,
  FinTechValidationRequest,
  FinTechValidationResult,
  FinancialOntology,
  RegulatoryScenario,
  ScenarioValidationResult,
  ExecutionTrajectory,
  ReplayResult,
  NRVTrace,
  FidelityScore,
  SemanticDistance,
  KYCBypassDetection,
  PositionLimitViolation,
  APIResponse,
  RegulatoryCheck,
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export const useFinTechValidator = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Status
  const [status, setStatus] = useState<{ status: string; version: string; ontologies: number } | null>(null);

  // Evidence Packs
  const [evidencePacks, setEvidencePacks] = useState<EvidencePack[]>([]);
  const [selectedEvidencePack, setSelectedEvidencePack] = useState<EvidencePack | null>(null);

  // Ontologies
  const [ontologies, setOntologies] = useState<FinancialOntology[]>([]);

  // Scenarios
  const [scenarios, setScenarios] = useState<RegulatoryScenario[]>([]);

  // Trajectories
  const [trajectories, setTrajectories] = useState<ExecutionTrajectory[]>([]);
  const [selectedTrajectory, setSelectedTrajectory] = useState<ExecutionTrajectory | null>(null);

  // NRV Traces
  const [nrvTraces, setNrvTraces] = useState<NRVTrace[]>([]);
  const [selectedTrace, setSelectedTrace] = useState<NRVTrace | null>(null);

  // Fidelity Scores
  const [fidelityScore, setFidelityScore] = useState<FidelityScore | null>(null);
  const [semanticDistance, setSemanticDistance] = useState<SemanticDistance | null>(null);

  // Detection Results
  const [kycBypassDetection, setKycBypassDetection] = useState<KYCBypassDetection | null>(null);
  const [positionLimitViolations, setPositionLimitViolations] = useState<PositionLimitViolation | null>(null);

  // Validation Results
  const [validationResult, setValidationResult] = useState<FinTechValidationResult | null>(null);

  // Fetch status
  const fetchStatus = useCallback(async () => {
    try {
      const response = await apiRequest<{ status: string; version: string; ontologies: number }>(
        `${API_BASE_URL}/api/fintech/status`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setStatus(response.data);
      }
    } catch (err) {
      console.error('Failed to fetch status:', err);
    }
  }, []);

  // Evidence Pack methods
  const fetchEvidencePacks = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<EvidencePack[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/evidence`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setEvidencePacks(response.data);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch evidence packs';
      setError(msg);
      console.error('Failed to fetch evidence packs:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchEvidencePack = useCallback(async (id: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<EvidencePack> = await apiRequest(
        `${API_BASE_URL}/api/fintech/evidence/${id}`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setSelectedEvidencePack(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Failed to fetch evidence pack');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch evidence pack';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const exportEvidencePack = useCallback(async (id: string) => {
    try {
      const response = await apiRequest(
        `${API_BASE_URL}/api/fintech/evidence/${id}/export`,
        { method: 'GET' }
      );
      return response.success;
    } catch (err) {
      console.error('Failed to export evidence pack:', err);
      return false;
    }
  }, []);

  // Ontology methods
  const fetchOntologies = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<FinancialOntology[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/ontologies`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setOntologies(response.data);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch ontologies';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchOntology = useCallback(async (id: string) => {
    setIsLoading(true);
    try {
      const response: APIResponse<FinancialOntology> = await apiRequest(
        `${API_BASE_URL}/api/fintech/ontologies/${id}`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        return response.data;
      }
      return null;
    } catch (err) {
      console.error('Failed to fetch ontology:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Scenario methods
  const fetchScenarios = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<RegulatoryScenario[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/scenarios`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setScenarios(response.data);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch scenarios';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const runScenarioValidation = useCallback(async (scenarioId: string, testData: Record<string, any>) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<ScenarioValidationResult> = await apiRequest(
        `${API_BASE_URL}/api/fintech/scenarios/validate`,
        {
          method: 'POST',
          body: JSON.stringify({ scenario_id: scenarioId, test_data: testData }),
        }
      );
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.error || 'Failed to run scenario validation');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to run scenario validation';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Trajectory methods
  const fetchTrajectories = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<ExecutionTrajectory[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/trajectories`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setTrajectories(response.data);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch trajectories';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchTrajectory = useCallback(async (id: string) => {
    setIsLoading(true);
    try {
      const response: APIResponse<ExecutionTrajectory> = await apiRequest(
        `${API_BASE_URL}/api/fintech/trajectories/${id}`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setSelectedTrajectory(response.data);
        return response.data;
      }
      return null;
    } catch (err) {
      console.error('Failed to fetch trajectory:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const replayTrajectory = useCallback(async (id: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<ReplayResult> = await apiRequest(
        `${API_BASE_URL}/api/fintech/trajectories/${id}/replay`,
        { method: 'POST' }
      );
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.error || 'Failed to replay trajectory');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to replay trajectory';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // NRV Trace methods
  const fetchNRVTraces = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<NRVTrace[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setNrvTraces(response.data);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch NRV traces';
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchNRVTrace = useCallback(async (id: string) => {
    setIsLoading(true);
    try {
      const response: APIResponse<NRVTrace> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces/${id}`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setSelectedTrace(response.data);
        return response.data;
      }
      return null;
    } catch (err) {
      console.error('Failed to fetch NRV trace:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const scoreFidelity = useCallback(async (traceId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<FidelityScore> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces/${traceId}/score`,
        { method: 'POST' }
      );
      if (response.success && response.data) {
        setFidelityScore(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Failed to calculate fidelity score');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to calculate fidelity score';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const calculateSemanticDistance = useCallback(async (traceId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<SemanticDistance> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces/${traceId}/distance`,
        { method: 'POST' }
      );
      if (response.success && response.data) {
        setSemanticDistance(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Failed to calculate semantic distance');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to calculate semantic distance';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const detectKYCBypass = useCallback(async (traceId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<KYCBypassDetection> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces/${traceId}/detect/kyc-bypass`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setKycBypassDetection(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Failed to detect KYC bypass');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to detect KYC bypass';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const detectPositionLimits = useCallback(async (traceId: string, limits?: Record<string, number>) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<PositionLimitViolation> = await apiRequest(
        `${API_BASE_URL}/api/fintech/nrv/traces/${traceId}/detect/position-limits`,
        {
          method: 'POST',
          body: JSON.stringify({ limits: limits || {} }),
        }
      );
      if (response.success && response.data) {
        setPositionLimitViolations(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Failed to detect position limit violations');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to detect position limit violations';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Validation methods
  const validate = useCallback(async (request: FinTechValidationRequest) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<FinTechValidationResult> = await apiRequest(
        `${API_BASE_URL}/api/fintech/validate`,
        {
          method: 'POST',
          body: JSON.stringify(request),
        }
      );
      if (response.success && response.data) {
        setValidationResult(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Validation failed');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Validation failed';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const validateWithNRV = useCallback(async (request: FinTechValidationRequest) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<FinTechValidationResult> = await apiRequest(
        `${API_BASE_URL}/api/fintech/validate/with-nrv`,
        {
          method: 'POST',
          body: JSON.stringify(request),
        }
      );
      if (response.success && response.data) {
        setValidationResult(response.data);
        return response.data;
      }
      throw new Error(response.error || 'Validation with NRV failed');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Validation with NRV failed';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const complianceCheck = useCallback(async (agentId: string, requirements: string[]) => {
    setIsLoading(true);
    setError(null);
    try {
      const response: APIResponse<RegulatoryCheck[]> = await apiRequest(
        `${API_BASE_URL}/api/fintech/compliance/check`,
        {
          method: 'POST',
          body: JSON.stringify({ agent_id: agentId, requirements }),
        }
      );
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.error || 'Compliance check failed');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Compliance check failed';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Initial data fetch
  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  return {
    // State
    isLoading,
    error,
    status,
    
    // Evidence Packs
    evidencePacks,
    selectedEvidencePack,
    fetchEvidencePacks,
    fetchEvidencePack,
    exportEvidencePack,
    
    // Ontologies
    ontologies,
    fetchOntologies,
    fetchOntology,
    
    // Scenarios
    scenarios,
    fetchScenarios,
    runScenarioValidation,
    
    // Trajectories
    trajectories,
    selectedTrajectory,
    fetchTrajectories,
    fetchTrajectory,
    replayTrajectory,
    
    // NRV Traces
    nrvTraces,
    selectedTrace,
    fetchNRVTraces,
    fetchNRVTrace,
    
    // Fidelity Scoring
    fidelityScore,
    semanticDistance,
    scoreFidelity,
    calculateSemanticDistance,
    
    // Detection
    kycBypassDetection,
    positionLimitViolations,
    detectKYCBypass,
    detectPositionLimits,
    
    // Validation
    validationResult,
    validate,
    validateWithNRV,
    complianceCheck,
  };
};

export default useFinTechValidator;
