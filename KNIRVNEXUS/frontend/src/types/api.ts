/**
 * Standardized API types that match the backend Go models
 * These types ensure consistency between frontend and backend data structures
 */

// Base response structure for all API endpoints
export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  total?: number;
  message?: string;
  error?: string;
  timestamp: string;
}

// DVE Node types - Aligned with backend models/dve.go
export interface DVENode {
  id: string;
  name: string;
  status: "online" | "offline" | "maintenance" | "error";
  tee_type: "sgx" | "sev-snp" | "tdx" | "software";
  stake_amount: number; // int64 from backend
  reputation_score: number;
  location: string;
  ip_address: string;
  public_key: string;
  capabilities: string[];
  last_heartbeat: string; // ISO 8601 timestamp
  created_at: string; // ISO 8601 timestamp
  updated_at: string; // ISO 8601 timestamp

  // Performance metrics
  cpu_usage: number; // float64 from backend
  memory_usage: number; // float64 from backend
  network_latency: number; // int64 from backend

  // Geographic coordinates
  latitude?: number; // float64 from backend
  longitude?: number; // float64 from backend
}

export interface ControllerDevice {
  id: string;
  user_id: string;
  device_type: string;
  device_name: string;
  os_version: string;
  app_version: string;
  capabilities: string[];
  status: "online" | "offline";
  last_seen: string;
  registered_at: string;
}

export interface ControllerStats {
  total_qr_codes: number;
  active_qr_codes: number;
  total_pairing_requests: number;
  pending_pairing_requests: number;
  active_sessions: number;
  total_devices: number;
  online_devices: number;
}

// Fabric types - Aligned with backend models/fabric.go
export interface Fabric {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  type: "WASM" | "LoRA" | "CodeT5" | "SEAL" | "NRN";
  status: "uploaded" | "deployed" | "running" | "stopped" | "error" | "archived";
  file_path: string;
  file_size: number; // int64 from backend
  file_hash: string;
  capabilities: string[];
  dependencies: string[];
  resource_limits?: FabricResourceLimits;
  configuration: Record<string, any>;
  metadata: Record<string, any>;
  tags: string[];
  uploaded_at: string; // ISO 8601 timestamp
  deployed_at?: string; // ISO 8601 timestamp
  last_modified: string; // ISO 8601 timestamp
  last_activity?: string; // ISO 8601 timestamp
  uploaded_by: string;
  deployed_by?: string;
  runtime_instance?: FabricRuntimeInstance;
}

export interface FabricResourceLimits {
  max_memory_mb: number;
  max_cpu_percent: number;
  max_execution_time_seconds: number;
  max_concurrency: number;
  max_disk_mb: number;
  network_access: boolean;
  file_system_access: boolean;
}

export interface FabricRuntimeInstance {
  instance_id: string;
  process_id?: number;
  started_at: string; // ISO 8601 timestamp
  status: "starting" | "running" | "stopping" | "stopped" | "crashed";
  resource_usage?: FabricResourceUsage;
  configuration: Record<string, any>;
  environment: Record<string, string>;
  health_check_url?: string;
  health_status?: "healthy" | "unhealthy" | "unknown";
  last_health_check?: string; // ISO 8601 timestamp
  error_message?: string;
  restart_count: number;
  uptime_seconds: number;
}

export interface FabricResourceUsage {
  cpu_percent: number;
  memory_mb: number;
  disk_read_mb: number;
  disk_write_mb: number;
  network_in_mb: number;
  network_out_mb: number;
  file_descriptors: number;
  threads: number;
  timestamp: string; // ISO 8601 timestamp
}

export interface FabricDeployment {
  id: string;
  model_id: string;
  name: string;
  description: string;
  environment: string;
  replicas: number;
  resource_limits: FabricResourceLimits;
  environment_variables: Record<string, string>;
  auto_restart: boolean;
  restart_policy: "always" | "on-failure" | "never";
  created_at: string; // ISO 8601 timestamp
  updated_at: string; // ISO 8601 timestamp
  created_by: string;
  status: "pending" | "deploying" | "deployed" | "failed" | "stopped";
  instances: FabricRuntimeInstance[];
}

export interface FabricMetrics {
  model_id: string;
  timestamp: string; // ISO 8601 timestamp
  cpu_usage_percent: number;
  memory_usage_mb: number;
  network_in_mbps: number;
  network_out_mbps: number;
  disk_read_mbps: number;
  disk_write_mbps: number;
  requests_per_second: number;
  errors_per_second: number;
  response_time_ms: number;
}

export interface FabricLog {
  id: string;
  model_id: string;
  level: "debug" | "info" | "warn" | "error" | "fatal";
  message: string;
  timestamp: string; // ISO 8601 timestamp
  source: string;
  metadata: Record<string, any>;
}

export interface FabricEvent {
  id: string;
  model_id: string;
  type: string;
  description: string;
  timestamp: string; // ISO 8601 timestamp
  metadata: Record<string, any>;
}

export interface FabricSummary {
  total_models: number;
  running_models: number;
  stopped_models: number;
  error_models: number;
  deployed_models: number;
  uploaded_models: number;
}

export interface FabricTemplate {
  id: string;
  name: string;
  description: string;
  type: string;
  default_configuration: Record<string, any>;
  default_resource_limits: FabricResourceLimits;
  created_at: string; // ISO 8601 timestamp
  created_by: string;
}

export interface FabricAction {
  action: "deploy" | "start" | "stop" | "restart" | "scale";
  parameters?: Record<string, any>;
}

// DNS types - Aligned with backend DNS service
export interface DNSRecord {
  id: string;
  name: string;
  type: string;
  value: string;
  ttl: number;
  zone: string;
  proxied?: boolean;
  priority?: number;
  comment?: string;
  created_at?: string; // ISO 8601 timestamp
  updated_at?: string; // ISO 8601 timestamp
}

export interface DNSZone {
  id: string;
  name: string;
  type: "primary" | "secondary";
  status?: string;
  records_count?: number;
  created_at?: string; // ISO 8601 timestamp
  updated_at?: string; // ISO 8601 timestamp
}

export interface DNSStatus {
  service: string;
  status: "running" | "stopped" | "error";
  zones: number;
  records: number;
  timestamp: string; // ISO 8601 timestamp
  current_ip?: string;
  last_update?: string; // ISO 8601 timestamp
  update_count?: number;
  error_count?: number;
}

export interface CreateDNSRecordRequest {
  name: string;
  type: string;
  value: string;
  ttl?: number;
  zone?: string;
  proxied?: boolean;
  priority?: number;
  comment?: string;
}

export interface UpdateDNSRecordRequest {
  value?: string;
  ttl?: number;
  proxied?: boolean;
  priority?: number;
  comment?: string;
}

// System Health types - Aligned with backend models
export interface SystemHealth {
  status: "healthy" | "degraded" | "unhealthy";
  timestamp: string; // ISO 8601 timestamp
  uptime: number;
  components: Record<string, ComponentHealth>;
  component_summary?: ComponentSummary;
  active_alerts?: number;
  alerts: SystemAlert[];
  metrics?: SystemMetrics;
}

export interface ComponentHealth {
  status: string;
  message: string;
  metrics?: Record<string, any>;
}

export interface ComponentSummary {
  total: number;
  healthy: number;
  degraded: number;
  unhealthy: number;
}

export interface SystemAlert {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  message: string;
  component: string;
  timestamp: string; // ISO 8601 timestamp
  metadata: Record<string, any>;
  acknowledged: boolean;
  acknowledged_by?: string;
  acknowledged_at?: string; // ISO 8601 timestamp
  resolved: boolean;
  resolved_by?: string;
  resolved_at?: string; // ISO 8601 timestamp
}

export interface SystemMetrics {
  cpu_usage_percent: number;
  memory_usage_percent: number;
  disk_usage_percent: number;
  network_in_mbps: number;
  network_out_mbps: number;
  active_connections: number;
  requests_per_second: number;
  errors_per_second: number;
  response_time_ms: number;
}

// Authentication types - Aligned with backend auth models
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  role: string;
  expires_at: string; // ISO 8601 timestamp
}

export interface MeResponse {
  user_id: string;
  username: string;
  role: string;
  issued_at: string; // ISO 8601 timestamp
}

export interface UserProfile {
  id: string;
  email: string;
  username: string;
  role: "admin" | "operator" | "validator" | "viewer";
  permissions: UserPermissions;
  created_at: string; // ISO 8601 timestamp
  updated_at: string; // ISO 8601 timestamp
  last_login: string; // ISO 8601 timestamp
  is_active: boolean;
}

export interface UserPermissions {
  can_manage_users: boolean;
  can_manage_nodes: boolean;
  can_manage_fabric: boolean;
  can_view_metrics: boolean;
  can_manage_dns: boolean;
  can_access_admin_panel: boolean;
}

// DVE Rental types - Aligned with backend models
export interface DVERentalPlan {
  id: string;
  name: string;
  description: string;
  price_per_hour: number; // NRN tokens
  cpu_cores: number;
  memory_gb: number;
  disk_gb: number;
  bandwidth_mbps: number;
  features: string[];
}

export interface DVERental {
  id: string;
  user_id: string;
  plan_id: string;
  node_id: string;
  status: "active" | "expired" | "cancelled";
  start_time: string; // ISO 8601 timestamp
  end_time: string; // ISO 8601 timestamp
  duration_hours: number;
  total_cost: number;
  payment_tx_hash: string;
  cde_access_url?: string;
  cde_credentials?: {
    username: string;
    password: string;
  };
  // ⭐ NEW FIELDS for endpoint access
  container_id?: string;
  ssh_username?: string;
  ssh_port?: number;
  access_token?: string;
  validation_session_id?: string;
  error_res_session_id?: string;
  provisioned_at?: string; // ISO 8601 timestamp
  provisioning_status?: "pending" | "provisioned" | "failed";
  // ⭐ NEW PAYMENT FIELDS
  payment_method_id?: string;
  payment_provider?: "stripe" | "paypal";
  payment_amount?: number; // in cents
  payment_currency?: string;
  payment_status?: "pending" | "completed" | "failed";
  stripe_charge_id?: string;
  paypal_order_id?: string;
  paypal_capture_id?: string;
  receipt_url?: string;
  invoice_id?: string;
  refund_requested?: boolean;
  refunded_amount?: number; // in cents
  payment_failure_reason?: string;
  created_at: string; // ISO 8601 timestamp
  updated_at: string; // ISO 8601 timestamp
}

// TEE Endpoint and Session types - NEW
export interface TEEEndpoint {
  id: string;
  rental_id: string;
  container_id: string;
  endpoint_type: "ssh" | "validation" | "error-resolution";
  host: string;
  port: number;
  protocol: "ssh" | "http" | "https" | "ws";
  credentials?: Credentials;
  status: "active" | "inactive" | "terminated";
  created_at: string; // ISO 8601 timestamp
  expires_at: string; // ISO 8601 timestamp
}

export interface Credentials {
  username?: string;
  private_key?: string; // Only in single-use responses
  token?: string;
  key_fingerprint?: string;
}

export interface SSHSession {
  id: string;
  rental_id: string;
  container_id: string;
  username: string;
  public_key_hash: string;
  private_key_url: string;
  endpoint: string;
  port: number;
  command: string;
  expires_at: string; // ISO 8601 timestamp
  created_at: string; // ISO 8601 timestamp
  last_used?: string; // ISO 8601 timestamp
}

export interface ValidationSession {
  id: string;
  rental_id: string;
  session_token: string;
  endpoint_url: string;
  port: number;
  expires_at: string; // ISO 8601 timestamp
  created_at: string; // ISO 8601 timestamp
  validation_type: string;
}

export interface ErrorResolutionSession {
  id: string;
  rental_id: string;
  session_token: string;
  endpoint_url: string;
  port: number;
  expires_at: string; // ISO 8601 timestamp
  created_at: string; // ISO 8601 timestamp
  supported_error_types: string[];
}

export interface DVEAccessInfo {
  ssh?: SSHSession;
  reasoning_validation?: ValidationSession;
  error_resolution?: ErrorResolutionSession;
  container_info?: {
    container_id: string;
    status: string;
    allocated_resources: {
      cpu_cores: number;
      memory_gb: number;
      disk_gb: number;
    };
  };
}

export interface DVERentalStats {
  total_rentals: number;
  active_rentals: number;
  total_revenue: number;
  average_duration: number;
  popular_plans: Array<{
    plan_id: string;
    plan_name: string;
    rental_count: number;
  }>;
}

export interface RentalRequest {
  plan_id: string;
  duration_hours: number;
  payment_tx_hash: string;
  user_id?: string;
}

export interface ExtendRentalRequest {
  additional_duration: number;
  payment_tx_hash: string;
}

// TEE Security types - Additional types for TEE security system
export interface TEEPerformanceMetrics {
  cpu_usage: number;
  memory_usage: number;
  network_latency: number;
  throughput: number;
  timestamp: string; // ISO 8601 timestamp
}

export interface ThreatAlert {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  description: string;
  detected_at: string; // ISO 8601 timestamp
  resolved: boolean;
  resolved_at?: string; // ISO 8601 timestamp
}

export interface SecurityAudit {
  id: string;
  audit_type: string;
  status: "pending" | "in_progress" | "completed" | "failed";
  started_at: string; // ISO 8601 timestamp
  completed_at?: string; // ISO 8601 timestamp
  findings: string[];
  recommendations: string[];
}

export interface DVENodeFilter {
  status?: string;
  tee_type?: string;
  location?: string;
  min_stake?: number;
  min_reputation?: number;
  capabilities?: string[];
  limit?: number;
}

export interface RegisterNodeRequest {
  name: string;
  tee_type: string;
  stake_amount: number;
  location?: string;
  ip_address?: string;
  public_key?: string;
  capabilities?: string[];
  latitude?: number;
  longitude?: number;
}

// Validation Task types
export interface ValidationTask {
  id: string;
  type: "skillnode" | "base_llm" | "custom";
  status: "pending" | "assigned" | "running" | "completed" | "failed";
  priority: number; // 1-10, higher is more urgent
  skill_code?: string;
  failure_context?: string;
  test_cases: TestCase[];
  required_tee_type: string;
  assigned_node_id?: string;
  requested_by: string;
  parameters: Record<string, any>;
  completion_percentage?: number; // 0-100
  estimated_completion_time?: string; // ISO 8601 timestamp
  created_at: string; // ISO 8601 timestamp
  updated_at: string; // ISO 8601 timestamp
  started_at?: string; // ISO 8601 timestamp
  completed_at?: string; // ISO 8601 timestamp
  timeout_at: string; // ISO 8601 timestamp
}

export interface TestCase {
  id: string;
  name: string;
  description: string;
  input: Record<string, any>;
  expected: Record<string, any>;
  weight: number; // float64 from backend
}

export interface ValidationResult {
  id: string;
  task_id: string;
  validator_node_id: string;
  status: "success" | "failure" | "error";
  score: number; // 0.0 - 1.0
  results: Record<string, any>;
  test_results: TestResult[];
  proof: string;
  tee_attestation: string;
  execution_time: number; // Duration in milliseconds
  created_at: string;
  completed_at: string;
}

export interface TestResult {
  test_case_id: string;
  status: "passed" | "failed" | "error";
  actual_output: Record<string, any>;
  expected_output: Record<string, any>;
  score: number;
  execution_time: number; // Duration in milliseconds
  error_message?: string;
}

export interface ValidationTaskFilter {
  status?: string;
  type?: string;
  priority?: number;
  requested_by?: string;
  created_after?: string; // ISO 8601 timestamp
  created_before?: string; // ISO 8601 timestamp
  limit?: number;
}

export interface CreateTaskRequest {
  type: string;
  priority: number;
  skill_code?: string;
  failure_context?: string;
  test_cases?: TestCase[];
  required_tee_type?: string;
  requested_by?: string;
  parameters?: Record<string, any>;
}

// TEE Security types
export interface TEEAttestation {
  id: string;
  node_id: string;
  tee_type: string;
  status: "valid" | "invalid" | "expired" | "pending";
  quote: string;
  signature: string;
  cert_chain: string;
  measurements: string[];
  created_at: string;
  expires_at: string;
  verified_at?: string;
}

export interface TEESecurityMetrics {
  attestation_status: "verified" | "pending" | "failed";
  security_score: number;
  threats_detected: number;
  last_audit: string;
  active_attestations: number;
  expired_attestations: number;
  failed_verifications: number;
}

// System Health Summary types
export interface SystemHealthSummary {
  id: string;
  overall_status: "healthy" | "degraded" | "critical";
  active_nodes: number;
  total_nodes: number;
  pending_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  average_response_time: number;
  network_latency: number;
  tee_health_score: number;
  timestamp: string;
}

// Network Topology types
export interface NetworkTopology {
  id: string;
  total_peers: number;
  connected_peers: number;
  peers: PeerInfo[];
  connections: ConnectionInfo[];
  timestamp: string;
}

export interface PeerInfo {
  id: string;
  address: string;
  role: string;
  status: string;
  latency: number;
  last_seen: string;
}

export interface ConnectionInfo {
  from_peer: string;
  to_peer: string;
  status: string;
  latency: number;
  bandwidth: number;
  established_at: string;
}

// WebSocket Update types
export interface DVENodeUpdate {
  id: string;
  status: string;
  cpu_usage: number;
  memory_usage: number;
  last_heartbeat: string;
}

export interface ValidationTaskUpdate {
  id: string;
  status: string;
  progress: number;
  assigned_node: string;
  estimated_completion?: string;
}

export interface CognitiveEngineUpdate {
  status: "active" | "idle" | "learning" | "error";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
}

export interface TEESecurityUpdate {
  attestation_status: "verified" | "pending" | "failed";
  security_score: number;
  threats_detected: number;
  last_audit: string;
}

// NRN Staking types
export interface StakingMetrics {
  total_staked: number;
  total_validators: number;
  average_stake: number;
  staking_ratio: number;
  annual_yield: number;
  last_updated: string;
}

export interface StakePosition {
  validator_id: string;
  amount: number;
  status: "active" | "pending" | "unbonding";
  created_at: string;
  updated_at: string;
  rewards_earned: number;
}

// Cognitive Engine types
export interface CognitiveEngineMetrics {
  status: "active" | "idle" | "learning" | "error";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
  model_version: string;
  last_updated: string;
  memory_usage: number;
  processing_speed: number;
}

// Error types
export interface APIError {
  code: string;
  message: string;
  details?: Record<string, any>;
  timestamp: string;
}

// Pagination types
export interface PaginationParams {
  page?: number;
  limit?: number;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

export interface PaginatedResponse<T> extends APIResponse<T[]> {
  page: number;
  limit: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

// ============================================
// FinTech Validator Types (Phases 1-4)
// ============================================

// Phase 1: Evidence Pack types
export interface EvidencePack {
  id: string;
  name: string;
  description: string;
  agent_id: string;
  agent_type: string;
  validation_id: string;
  context_records: ContextRecord[];
  regulatory_checks: RegulatoryCheck[];
  certificates: Certificate[];
  trajectories: string[];
  metadata: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ContextRecord {
  id: string;
  timestamp: string;
  reasoning_step: string;
  action: string;
  input_data: Record<string, any>;
  output_data: Record<string, any>;
  confidence: number;
  embeddings?: number[];
}

export interface RegulatoryCheck {
  id: string;
  rule_id: string;
  rule_name: string;
  status: "passed" | "failed" | "warning" | "error";
  details: string;
  severity: "low" | "medium" | "high" | "critical";
  evidence: Record<string, any>;
  timestamp: string;
}

export interface Certificate {
  id: string;
  type: string;
  issued_at: string;
  expires_at: string;
  issuer: string;
  subject: string;
  serial_number: string;
  signature: string;
  public_key: string;
  metadata: Record<string, any>;
}

// Phase 1: Validation types
export interface FinTechValidationRequest {
  agent_id: string;
  agent_name?: string;
  agent_code?: string;
  validation_type?: string;
  parameters?: Record<string, any>;
  financial_actions?: any[];
  simple_financial_actions?: SimpleFinancialAction[];
  categories?: string[];
  test_cases?: any[];
  requested_by?: string;
}

export interface SimpleFinancialAction {
  action_type: string;
  instrument: string;
  quantity: number;
  price: number;
  timestamp: string;
  account_id: string;
}

export interface FinTechValidationResult {
  validation_id: string;
  evidence_pack_id: string;
  status: string;
  overall_score: number;
  is_compliant: boolean;
  validation_result?: any;
  compliance_result?: any;
  evidence_pack_url?: string;
  completed_at?: string;
  duration?: number;
}

export interface ComplianceResult {
  requirement: string;
  status: "compliant" | "non_compliant" | "not_applicable";
  details: string;
  evidence: Record<string, any>;
}

export interface RegulatoryViolation {
  regulation: string;
  rule_id: string;
  severity: "low" | "medium" | "high" | "critical";
  description: string;
  remediation: string;
  evidence: Record<string, any>;
}

// Phase 2: Ontology types
export interface FinancialOntology {
  id: string;
  name: string;
  version: string;
  description: string;
  domain: string;
  rules: OntologyRule[];
  created_at: string;
  updated_at: string;
}

export interface OntologyRule {
  id: string;
  name: string;
  description: string;
  condition: string;
  action: string;
  severity: "low" | "medium" | "high" | "critical";
  metadata: Record<string, any>;
}

// Phase 2: Scenario types
export interface RegulatoryScenario {
  id: string;
  name: string;
  description: string;
  ontology_id: string;
  test_cases: ScenarioTestCase[];
  metadata: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ScenarioTestCase {
  id: string;
  name: string;
  description: string;
  input_data: Record<string, any>;
  expected_output: Record<string, any>;
  compliance_requirements: string[];
}

export interface ScenarioValidationResult {
  scenario_id: string;
  test_results: TestCaseResult[];
  overall_status: "passed" | "failed" | "partial";
  summary: string;
  timestamp: string;
}

export interface TestCaseResult {
  test_case_id: string;
  status: "passed" | "failed" | "error";
  details: string;
  execution_time_ms: number;
}

// Phase 3: Trajectory types
export interface ExecutionTrajectory {
  id: string;
  agent_id: string;
  session_id: string;
  steps: TrajectoryStep[];
  captured_at: string;
  duration_ms: number;
}

export interface TrajectoryStep {
  step_number: number;
  timestamp: string;
  action: string;
  input_data: Record<string, any>;
  output_data: Record<string, any>;
  state_hash: string;
  stack_trace?: string;
}

export interface ReplayResult {
  trajectory_id: string;
  replay_id: string;
  status: "success" | "failure" | "divergence";
  original_output: Record<string, any>;
  replay_output: Record<string, any>;
  divergence_details?: string;
  execution_time_ms: number;
}

// Phase 4: NRV Trace & Fidelity types
export interface NRVTrace {
  id: string;
  agent_id: string;
  session_id: string;
  reasoning_steps: NRVReasoningStep[];
  financial_context: FinancialContext;
  metadata: Record<string, any>;
  created_at: string;
}

export interface NRVReasoningStep {
  step_number: number;
  timestamp: string;
  reasoning: string;
  action: string;
  amount?: number;
  counterparty?: string;
  instrument?: string;
  position_change?: number;
  confidence: number;
}

export interface FinancialContext {
  transaction_type?: string;
  amount?: number;
  currency?: string;
  counterparties: string[];
  instruments: string[];
  positions: Record<string, number>;
  kyc_status?: string;
  risk_category?: string;
}

export interface FidelityScore {
  trace_id: string;
  overall_score: number;
  risk_level: "low" | "medium" | "high" | "critical";
  component_scores: {
    intent: number;
    action: number;
    compliance: number;
    outcome: number;
  };
  violations: FidelityViolation[];
  recommendations: string[];
  calculated_at: string;
}

export interface FidelityViolation {
  category: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  evidence: string;
}

export interface SemanticDistance {
  trace_id: string;
  distance: number;
  components: {
    intent_distance: number;
    action_distance: number;
    compliance_distance: number;
    outcome_distance: number;
  };
  details: string;
  calculated_at: string;
}

export interface KYCBypassDetection {
  trace_id: string;
  detected: boolean;
  confidence: number;
  indicators: string[];
  evidence: string;
  timestamp: string;
}

export interface PositionLimitViolation {
  trace_id: string;
  detected: boolean;
  violations: PositionViolation[];
  current_positions: Record<string, number>;
  limits: Record<string, number>;
  timestamp: string;
}

export interface PositionViolation {
  instrument: string;
  current_position: number;
  limit: number;
  excess: number;
  severity: "low" | "medium" | "high" | "critical";
}
