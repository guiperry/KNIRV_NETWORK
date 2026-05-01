import z from "zod";

// Workflow status types
export const WorkflowStatusSchema = z.enum(['active', 'idle', 'error', 'deploying']);
export type WorkflowStatus = z.infer<typeof WorkflowStatusSchema>;

// Workflow configuration
export const WorkflowConfigSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.enum(['CodeT5', 'SEAL', 'LoRA', 'NRN']),
  status: WorkflowStatusSchema,
  tasks: z.number().int(),
  performance: z.number().min(0).max(100),
  lastActive: z.string(),
  config: z.record(z.any()).optional(),
});
export type WorkflowConfig = z.infer<typeof WorkflowConfigSchema>;

// User Delegation Certificate (UDC)
export const UDCSchema = z.object({
  id: z.string(),
  userId: z.string(),
  status: z.enum(['valid', 'expired', 'revoked', 'pending']),
  issuedAt: z.string(),
  expiresAt: z.string(),
  permissions: z.array(z.string()),
  signature: z.string(),
});
export type UDC = z.infer<typeof UDCSchema>;

// NRN (Neural Resource Network) transaction
export const NRNTransactionSchema = z.object({
  id: z.string(),
  userId: z.string(),
  workflowId: z.string(),
  amount: z.number(),
  type: z.enum(['consumption', 'reward', 'transfer']),
  timestamp: z.string(),
  description: z.string().optional(),
});
export type NRNTransaction = z.infer<typeof NRNTransactionSchema>;

// Skill definition
export const SkillSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  category: z.enum(['automation', 'analysis', 'communication', 'computation']),
  complexity: z.number().min(1).max(10),
  nrnCost: z.number(),
  requirements: z.array(z.string()),
  isActive: z.boolean(),
});
export type Skill = z.infer<typeof SkillSchema>;

// SEAL Loop status
export const SEALLoopStatusSchema = z.object({
  isActive: z.boolean(),
  currentCycle: z.number(),
  nextCycleAt: z.string(),
  optimizations: z.number(),
  failureDetections: z.number(),
  solutionsProposed: z.number(),
});
export type SEALLoopStatus = z.infer<typeof SEALLoopStatusSchema>;

// === Wallet Alignment Types ===

export interface DVEWallet {
  dveID: string;
  dveName: string;
  walletAddress: string;
  derivationIndex: number;
  teeType: "sgx" | "sev-snp" | "tdx" | "software" | "browser-extension";
  status: "online" | "offline" | "maintenance" | "error";
  stakeAmount: number;
  reputationScore: number;
  capabilities: string[];
  attachedPolicies: string[];
  badgeNFTIDs: string[];
  supervisorAgentID: string;
  dveURI: string;
}

export interface VaultAccount {
  address: string;
  nrnBalance: number;
  usdValue: number;
  dveWallets: DVEWallet[];
  isRootNode: boolean;
}

export interface AgentMessage {
  id: string;
  role: "user" | "agent" | "system" | "cognitive";
  content: string;
  timestamp: string;
  dveID?: string;
  taskID?: string;
  streaming?: boolean;
}

export type DVEStatus = "online" | "offline" | "maintenance" | "error";
export type TEEType = "sgx" | "sev-snp" | "tdx" | "software" | "browser-extension";

export interface DVECreation {
  id: string;
  name: string;
  ownerAddress: string;
  walletAddress: string;
  teeType: TEEType;
  status: DVEStatus;
  stakeAmount: number;
  reputationScore: number;
  capabilities: string[];
  attachedPolicyIDs: string[];
  badgeNFTIDs: string[];
  supervisorAgentID: string;
  createdAt: string;
  updatedAt: string;
  dveURI: string;
}

export interface CognitiveEngineStatus {
  active: boolean;
  rootNodeID: string;
  dveFleetSize: number;
  dveOnline: number;
  oracleLoaded: boolean;
}
