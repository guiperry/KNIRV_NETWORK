export const DVE_CONSTANTS = {
  HEARTBEAT_INTERVAL_MS: 45_000,
  HEARTBEAT_TIMEOUT_MS: 90_000,
  WS_RECONNECT_BASE_DELAY_MS: 1_000,
  WS_RECONNECT_MAX_DELAY_MS: 30_000,
  MAX_CONCURRENT_TASKS: 1,
  MAX_TASKS_PER_MINUTE: 10,
  TASK_PAYLOAD_MAX_BYTES: 100_000,
  TASK_EXECUTION_TIMEOUT_MS: 10_000,
  BADGE_SYNC_INTERVAL_MS: 300_000,
  DVE_API_PATH: '/api/dve',
  WS_PATH: '/api/dve/browser/ws',
} as const;

export const DVE_TASK_TYPES = [
  'policy-check',
  'signature-verify',
  'reasoning-simple',
  'skill-lint',
] as const;

export type DVETaskType = typeof DVE_TASK_TYPES[number];

export const DVE_TRUST_TIERS = ['standard', 'verified', 'root'] as const;
export type DVETrustTier = typeof DVE_TRUST_TIERS[number];
