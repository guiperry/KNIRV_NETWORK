
// src/lib/agent-service.ts
import type { Agent, AgentSignature } from '@/lib/types';
import { db } from '@/lib/firebase';
import { collection, getDocs, doc, getDoc, writeBatch, setDoc } from 'firebase/firestore';

// KNIRV-ORACLE configuration
const KNIRV_ORACLE_URL = process.env.NEXT_PUBLIC_KNIRVCHAIN_URL || 'http://localhost:1317';

// Interface for KNIRV-ORACLE registration response
interface KNIRVCHAINRegistrationResponse {
  transactionHash: string;
  capabilityId: string;
  status: string;
  message?: string;
}

// Function to register agent/operator with KNIRV-ORACLE and get transaction hash
export const registerAgentWithKNIRVCHAIN = async (agent: Agent): Promise<KNIRVCHAINRegistrationResponse> => {
  try {
    const registrationData = {
      agentId: agent.id,
      name: agent.name,
      capability: agent.capability,
      capabilities: agent.capabilities,
      provider: agent.provider,
      version: agent.version,
      description: agent.description,
      factsUrl: agent.addr_facts_url,
      ansName: agent.ansName,
      registeredAt: agent.registeredAt
    };

    const response = await fetch(`${KNIRV_ORACLE_URL}/register-capability`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'NANDA-ANS-Registry/1.0'
      },
      body: JSON.stringify(registrationData)
    });

    if (!response.ok) {
      throw new Error(`KNIRV-ORACLE registration failed: ${response.status} ${response.statusText}`);
    }

    const result = await response.json();

    return {
      transactionHash: result.transactionHash || result.transaction_hash,
      capabilityId: result.capabilityId || result.capability_id,
      status: result.status || 'pending',
      message: result.message
    };
  } catch (error) {
    console.error('Error registering agent with KNIRV-ORACLE:', error);
    throw new Error(`Failed to register with KNIRV-ORACLE: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
};

const generateMockSignatureForAgent = (agentId: string, createdDate?: string, issuer?: string, type?: string): AgentSignature => {
  const creationDate = createdDate || new Date().toISOString();
  return {
    type: type || "RsaSignature2018",
    created: creationDate,
    verificationMethod: `${agentId}#key-pki-1`,
    proofPurpose: "assertionMethod",
    proofValue: Array(128).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''), // Longer mock signature
    simulatedIssuer: issuer || "NANDA+ANS Trusted CA G1",
    simulatedPublicKey: `04:${Array(64).fill(0).map(() => Math.floor(Math.random() * 256).toString(16).padStart(2, '0')).join(':').toUpperCase()}`,
  };
};

// The mockAgents array can be kept for fallback or initial seeding.
export const mockAgents: Agent[] = [
  {
    id: 'did:nanda:agent-translator-001',
    name: 'LinguaBot Pro',
    ansName: 'a2a://LinguaBotPro.DocumentTranslation.PolyglotAI.v2.1.certified',
    capability: 'Document Translation',
    capabilities: ['Document Translation', 'Language Identification', 'Text Summarization'],
    provider: 'PolyglotAI',
    version: '2.1',
    extension: 'certified',
    description: 'Advanced AI for translating documents in over 100 languages with high accuracy and context preservation. Supports various file formats and offers secure processing. Includes simulated PKI signature for trust demonstration.',
    endpoints: {
      "@type": "nanda:EndpointSet",
      static_endpoint: ["https://api.polyglotai.com/translate/v2.1"],
      adaptive_router_url: "https://router.polyglotai.com/translate"
    },
    attestations: ["did:nanda:attestation-polyglot-hipaa-v1", "did:nanda:attestation-polyglot-iso27001-v1"],
    protocolExtensions: {
      "@type": "ans:ProtocolExtensionSet",
      a2a: { "@type": "A2AAgentCard", name: "LinguaBot A2A Interface", details: { supportedFormats: ["pdf", "docx", "txt"] } },
      mcp: { "@type": "MCPToolDescription", name: "LinguaBot MCP Tool", details: { maxFileSize: "10MB" } }
    },
    avatarUrl: 'https://placehold.co/100x100.png',
    dataAiHint: 'robot language',
    registeredAt: new Date(2024, 0, 15).toISOString(), // Jan 15, 2024
    addr_ttl: 3600,
    addr_facts_url: "https://polyglotai.com/.well-known/agent-facts-linguabot.jsonld",
    signature: generateMockSignatureForAgent('did:nanda:agent-translator-001', new Date(2024, 0, 14).toISOString(), "PolyglotAI CA")
  },
  {
    id: 'did:nanda:agent-dataprotector-002',
    name: 'GuardianShield',
    ansName: 'acp://GuardianShield.DataAnonymization.SecureDataCorp.v1.0.enterprise',
    capability: 'Data Anonymization',
    capabilities: ['Data Anonymization', 'PII Detection', 'Privacy Policy Enforcement'],
    provider: 'SecureDataCorp',
    version: '1.0',
    extension: 'enterprise',
    description: 'Enterprise-grade data anonymization agent ensuring GDPR and CCPA compliance. Uses advanced techniques to protect sensitive information while preserving data utility. Includes simulated PKI signature for trust demonstration.',
    endpoints: {
      "@type": "nanda:EndpointSet",
      static_endpoint: ["https://api.securedatacorp.com/anonymize/v1.0"]
    },
    attestations: ["did:nanda:attestation-securedata-gdpr-v2", "did:nanda:attestation-securedata-soc2-v1"],
    protocolExtensions: {
      "@type": "ans:ProtocolExtensionSet",
      acp: { "@type": "ACPProfile", name: "GuardianShield ACP", details: { complianceLevel: "Strict" } }
    },
    avatarUrl: 'https://placehold.co/100x100.png',
    dataAiHint: 'shield security',
    registeredAt: new Date(2023, 10, 20).toISOString(), // Nov 20, 2023
    addr_ttl: 7200,
    addr_facts_url: "https://securedatacorp.com/.well-known/agent-facts-guardian.jsonld",
    signature: generateMockSignatureForAgent('did:nanda:agent-dataprotector-002', new Date(2023, 10, 19).toISOString(), "SecureDataCorp CA", "EcdsaSecp256k1Signature2019")
  },
  {
    id: 'did:nanda:agent-artcreator-003',
    name: 'PixelDreamer',
    ansName: 'mcp://PixelDreamer.ImageGeneration.ArtifyInc.v3.beta',
    capability: 'Image Generation',
    capabilities: ['Image Generation from Text', 'Style Transfer', 'Image Upscaling'],
    provider: 'ArtifyInc',
    version: '3.0',
    extension: 'beta',
    description: 'A creative AI agent that generates stunning and unique images from textual descriptions. Explore various artistic styles and resolutions. Includes simulated PKI signature for trust demonstration.',
    endpoints: {
      "@type": "nanda:EndpointSet",
      static_endpoint: ["https://api.artifyinc.com/imagegen/v3"],
      adaptive_router_url: "https://router.artifyinc.com/imagegen"
    },
    attestations: [],
    protocolExtensions: {
      "@type": "ans:ProtocolExtensionSet",
      mcp: { "@type": "MCPToolDescription", name: "PixelDreamer MCP", details: { outputResolutions: ["512x512", "1024x1024", "2048x2048"] } }
    },
    avatarUrl: 'https://placehold.co/100x100.png',
    dataAiHint: 'abstract art',
    registeredAt: new Date(2024, 2, 1).toISOString(), // Mar 1, 2024
    addr_ttl: 1800,
    addr_facts_url: "https://artifyinc.com/.well-known/agent-facts-pixel.jsonld",
    signature: generateMockSignatureForAgent('did:nanda:agent-artcreator-003', new Date(2024, 1, 28).toISOString(), "ArtifyInc CA")
  },
  {
    id: 'did:nanda:agent-codehelper-004',
    name: 'DevMentor',
    ansName: 'a2a://DevMentor.CodeGeneration.CodeGenius.v1.5',
    capability: 'Code Generation',
    capabilities: ['Code Generation', 'Debugging Assistance', 'API Documentation Lookup'],
    provider: 'CodeGenius',
    version: '1.5',
    description: 'An AI assistant for developers, providing code snippets, debugging help, and quick access to API documentation. Supports Python, JavaScript, and Java. Includes simulated PKI signature for trust demonstration.',
    endpoints: {
      "@type": "nanda:EndpointSet",
      static_endpoint: ["https://api.codegenius.com/devmentor/v1.5"]
    },
    attestations: ["did:nanda:attestation-codegenius-opensource-contrib-v1"],
    protocolExtensions: {
      "@type": "ans:ProtocolExtensionSet",
      a2a: { "@type": "A2AAgentCard", name: "DevMentor A2A", details: { supportedLanguages: ["Python", "JavaScript", "Java"] } }
    },
    avatarUrl: 'https://placehold.co/100x100.png',
    dataAiHint: 'code computer',
    registeredAt: new Date(2024, 1, 10).toISOString(), // Feb 10, 2024
    addr_ttl: 3600,
    addr_facts_url: "https://codegenius.com/.well-known/agent-facts-devmentor.jsonld",
    signature: generateMockSignatureForAgent('did:nanda:agent-codehelper-004', new Date(2024, 1, 9).toISOString(), "CodeGenius Community CA")
  },
  {
    id: 'did:nanda:agent-dbquery-005',
    name: 'DataOracle',
    ansName: 'acp://DataOracle.DatabaseQuery.QueryWorks.v1.2.trusted',
    capability: 'Database Query',
    capabilities: ['SQL Query Execution', 'NoSQL Data Retrieval', 'Data Aggregation'],
    provider: 'QueryWorks Inc.',
    version: '1.2',
    extension: 'trusted',
    description: 'A powerful AI agent for securely querying various types of databases and retrieving structured data. Supports natural language queries and returns results in multiple formats. Includes simulated PKI signature for trust demonstration.',
    endpoints: {
      "@type": "nanda:EndpointSet",
      static_endpoint: ["https://api.queryworks.com/query/v1.2"],
      adaptive_router_url: "https://router.queryworks.com/query"
    },
    attestations: ["did:nanda:attestation-queryworks-pci-dss-v1"],
    protocolExtensions: {
      "@type": "ans:ProtocolExtensionSet",
      acp: { "@type": "ACPProfile", name: "DataOracle ACP", details: { supportedDatabases: ["PostgreSQL", "MongoDB", "MySQL"] } }
    },
    avatarUrl: 'https://placehold.co/100x100.png',
    dataAiHint: 'database tech',
    registeredAt: new Date(2023, 8, 5).toISOString(), // Sep 5, 2023
    addr_ttl: 3000,
    addr_facts_url: "https://queryworks.com/.well-known/agent-facts-dataoracle.jsonld",
    signature: generateMockSignatureForAgent('did:nanda:agent-dbquery-005', new Date(2023, 8, 4).toISOString(), "QueryWorks Trusted CA", "JsonWebSignature2020")
  }
];


export const getAgentById = async (id: string): Promise<Agent | undefined> => {
  try {
    const agentDocRef = doc(db, "agents", id);
    const agentSnap = await getDoc(agentDocRef);

    if (agentSnap.exists()) {
      return { id: agentSnap.id, ...agentSnap.data() } as Agent;
    } else {
      console.warn(`Agent with ID ${id} not found in Firestore. Checking mock data.`);
      // Fallback to mock data if not found in Firestore
      const mockAgent = mockAgents.find(agent => agent.id === id);
      if (mockAgent) {
        return mockAgent;
      }
      console.warn(`Agent with ID ${id} not found in mock data either.`);
      return undefined;
    }
  } catch (error) {
    console.error("Error fetching agent by ID from Firestore:", error);
    // Fallback or rethrow
    const mockAgent = mockAgents.find(agent => agent.id === id);
    if (mockAgent) {
        console.warn(`Falling back to mock agent with ID ${id} due to Firestore error.`);
        return mockAgent;
    }
    return undefined;
  }
};

// Operator-named alias for backward compatibility and clarity
export const getOperatorById = getAgentById;

export const getAllAgents = async (): Promise<Agent[]> => {
  const FETCH_TIMEOUT = 3000; // 3 seconds

  const timeoutPromise = new Promise<never>((_, reject) =>
    setTimeout(() => reject(new Error('Firestore fetch timed out')), FETCH_TIMEOUT)
  );

  const firestoreFetchAndCombineOperation = async (): Promise<Agent[]> => {
    let agentsListFromFirestore: Agent[] = [];
    const agentsCollectionRef = collection(db, "agents");

    try {
        const agentSnapshot = await getDocs(agentsCollectionRef);
        agentsListFromFirestore = agentSnapshot.docs.map(docSnap => ({ id: docSnap.id, ...docSnap.data() } as Agent));

        // Seeding logic (only if Firestore is completely empty and mockAgents has items)
        if (agentsListFromFirestore.length === 0 && mockAgents.length > 0) {
                console.log("No operators found in Firestore. Seeding database with mock data...");
            const batch = writeBatch(db);
            mockAgents.forEach(agent => {
                const agentRef = doc(db, "agents", agent.id);
                // Ensure all fields are defined, especially registeredAt which might be generated
                // And ensure signature is part of the seeded data
                const agentData = { 
                  ...agent, 
                  registeredAt: agent.registeredAt || new Date().toISOString(),
                  signature: agent.signature || generateMockSignatureForAgent(agent.id, agent.registeredAt) 
                };
                batch.set(agentRef, agentData);
            });
            await batch.commit();
            console.log("Mock data seeded successfully to Firestore.");
            // After seeding, Firestore now contains the mockAgents.
            // It's important to reflect this in the data returned by this operation.
            agentsListFromFirestore = mockAgents.map(agent => ({
                ...agent,
                registeredAt: agent.registeredAt || new Date().toISOString(),
                signature: agent.signature || generateMockSignatureForAgent(agent.id, agent.registeredAt)
            }));
        }
    } catch (dbError) {
        console.error("Error fetching or seeding agents from Firestore:", dbError);
        // If there's an error with Firestore, agentsListFromFirestore will remain empty or partially filled.
        // The merge logic below will handle this gracefully.
    }
    
    // Start with a deep copy of mockAgents to ensure all hardcoded agents are present.
    const combinedAgents = mockAgents.map(agent => ({...agent})); 
    const mockAgentIds = new Set(mockAgents.map(agent => agent.id));

    // Add agents from Firestore that are not already in mockAgents (based on ID)
    // This ensures that if Firestore has additional agents, they are included.
    agentsListFromFirestore.forEach(dbAgent => {
        if (!mockAgentIds.has(dbAgent.id)) {
            combinedAgents.push({...dbAgent});
        } else {
            // If agent from DB is also in mock, ensure the DB version (potentially more up-to-date) is used
            // or merge them. For simplicity, we can prioritize the DB version if it exists for a mock ID.
            // However, current logic prioritizes mock data and only adds *new* from DB.
            // To ensure DB data for existing mock IDs can be shown, we might need to update existing ones.
            // For now, let's keep it simple: mockAgents are the base, unique DB agents are added.
            // If a mockAgent was updated in DB, this logic doesn't automatically reflect it unless mock is removed.
        }
    });

    return combinedAgents;
  };

  try {
    // Race the fetch operation against the timeout
    const result = await Promise.race([firestoreFetchAndCombineOperation(), timeoutPromise]);
    return result;
  } catch (error: any) {
    let fallbackMessage = "Falling back to returning local mock agents due to an unexpected error or timeout.";
    if (error && error.message === 'Firestore fetch timed out') {
      fallbackMessage = `Firestore operation timed out after ${FETCH_TIMEOUT / 1000} seconds. Falling back to mock agents.`;
      console.warn(fallbackMessage);
    } else {
      console.error("Error during Firestore operation or timeout:", error);
    }
    // If timeout or any other error, return only a copy of mockAgents
    return mockAgents.map(agent => ({...agent}));
  }
};

// Operator-named alias
export const getAllOperators = getAllAgents;

// Function to register a new agent with KNIRV-ORACLE integration
export const registerNewAgent = async (agentData: Partial<Agent>): Promise<{ agent: Agent; transactionHash: string; capabilityId: string }> => {
  try {
    // Generate agent ID if not provided
    const agentId = agentData.id || `did:nanda:agent-${Date.now()}`;

    // Create complete agent object
    const agent: Agent = {
      id: agentId,
      name: agentData.name || '',
      ansName: agentData.ansName || '',
      capability: agentData.capability || '',
      capabilities: agentData.capabilities || [],
      provider: agentData.provider || '',
      version: agentData.version || '1.0',
      extension: agentData.extension,
      description: agentData.description || '',
      endpoints: agentData.endpoints || {
        "@type": "nanda:EndpointSet",
        static_endpoint: []
      },
      attestations: agentData.attestations || [],
      protocolExtensions: agentData.protocolExtensions || {
        "@type": "ans:ProtocolExtensionSet"
      },
      avatarUrl: agentData.avatarUrl || 'https://placehold.co/100x100.png',
      dataAiHint: agentData.dataAiHint || '',
      registeredAt: new Date().toISOString(),
      addr_ttl: agentData.addr_ttl || 3600,
      addr_facts_url: agentData.addr_facts_url || '',
      signature: generateMockSignatureForAgent(agentId)
    };

    // Register with KNIRV-ORACLE first to get transaction hash
    console.log('Registering agent with KNIRV-ORACLE:', agent.name);
    const oracleResponse = await registerAgentWithKNIRVCHAIN(agent);

    // Add transaction hash and capability ID to agent data
    const agentWithTransaction = {
      ...agent,
      transactionHash: oracleResponse.transactionHash,
      capabilityId: oracleResponse.capabilityId,
      oracleStatus: oracleResponse.status
    };

    // Save to Firestore with transaction information
    const agentRef = doc(db, "agents", agentId);
    await setDoc(agentRef, agentWithTransaction);

  console.log('Operator registered successfully:', {
      id: agentId,
      transactionHash: oracleResponse.transactionHash,
      capabilityId: oracleResponse.capabilityId
    });

    return {
      agent: agentWithTransaction,
      transactionHash: oracleResponse.transactionHash,
      capabilityId: oracleResponse.capabilityId
    };
  } catch (error) {
    console.error('Error registering new agent:', error);
    throw error;
  }
};

// Operator-named alias
export const registerNewOperator = registerNewAgent;
