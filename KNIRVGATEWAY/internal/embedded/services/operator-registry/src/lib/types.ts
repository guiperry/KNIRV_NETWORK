// src/lib/types.ts

export interface AgentSignature {
  type: string; // e.g., "RsaSignature2018", "EdDsaSAPublicKeySecp256k1"
  created: string; // ISO date string for signature creation
  verificationMethod: string; // DID URL of the public key (e.g., agentDID#key-1)
  proofPurpose: string; // e.g., "assertionMethod", "capabilityDelegation"
  proofValue: string; // The simulated signature value (long hex string)
  simulatedIssuer?: string; // e.g., "NANDA+ANS Trusted CA G1"
  simulatedPublicKey?: string; // e.g., "04:AB:CD:EF:12:34:56:78:90:..." (Hex representation of a public key)
}

export interface EndpointSet {
  "@type": string; // e.g., "nanda:EndpointSet"
  static_endpoint?: string[];
  adaptive_router_url?: string;
}

export interface ProtocolExtension {
  "@type": string; // e.g., "A2AAgentCard"
  name?: string; // Generic name for display
  details?: Record<string, any>; // Protocol-specific fields
}

export interface ProtocolExtensionSet {
  "@type": string; // e.g., "ans:ProtocolExtensionSet"
  a2a?: ProtocolExtension;
  mcp?: ProtocolExtension;
  acp?: ProtocolExtension;
}

// Represents the detailed information about an agent
export interface AgentFacts {
  id: string; // Agent's own DID, primary key (NANDA Identifier)
  ansName?: string; // Full ANS name, e.g., "a2a://myTranslator.DocumentTranslation.LinguaCorp.v1.2.certified" (ANS Capability Address)
  name: string; // User-friendly display name for the agent
  capability: string; // Primary capability for display
  capabilities?: string[]; // Full list of capabilities
  provider?: string;
  version?: string;
  extension?: string; // e.g., "certified" from ANS name
  description?: string;
  endpoints?: EndpointSet;
  attestations?: string[]; // Array of strings, could be DIDs or URLs to VCs
  protocolExtensions?: ProtocolExtensionSet;
  signature?: AgentSignature; // Signature by agent or publisher
  avatarUrl?: string; // Optional URL for agent's avatar image
  dataAiHint?: string; // Keywords for AI-assisted placeholder image generation
  registeredAt?: string; // ISO date string
}

// Represents the pointer to AgentFacts stored in the NANDA+ANS registry
export interface AgentAddr {
  agent_id: string; // DID, should match AgentFacts.id
  facts_url: string; // NANDA pointer to the AgentFacts URL
  private_facts_url?: string;
  adaptive_router_url?: string;
  ttl: number; // Time-to-live for this NANDA record
  signature: AgentSignature; // Signature from the registry shard
}

// Combined Agent type primarily used for UI display and interactions.
// It merges essential fields from AgentFacts and AgentAddr.
export interface Agent extends AgentFacts {
  // AgentFacts fields are inherited.
  // We can add specific fields from AgentAddr if needed directly.
  addr_ttl?: number; // TTL from NANDA AgentAddr
  addr_facts_url?: string; // facts_url from NANDA AgentAddr, distinguishing it if AgentFacts also had a facts_url.
  aiSummary?: string; // To be populated by GenAI

  // KNIRV-ORACLE integration fields
  transactionHash?: string; // Transaction hash from KNIRV-ORACLE registration
  capabilityId?: string; // Capability ID assigned by KNIRV-ORACLE
  oracleStatus?: string; // Registration status from KNIRV-ORACLE (pending, confirmed, failed)
}

// Alias for Operator vocabulary used in the Operator Registry UI
export type Operator = Agent;
