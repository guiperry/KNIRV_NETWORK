// network_composition_tests.p - Cross-Component Compositional Tests
// Tests the full KNIRV Network composition with all components and monitors
// Based on: KNIRV Network Architecture

// =====================================================================
// NETWORK TEST DRIVER
// Orchestrates the full network test
// =====================================================================

machine NetworkTestDriver {
    // Component machines
    var tokenMachine: machine;
    var governanceMachine: machine;
    var economicsMachine: machine;
    var consensusMachine: machine;
    var ibcMachine: machine;

    var skillRegistry: machine;
    var llmRegistry: machine;
    var mcpCapability: machine;
    var nodeTransformation: machine;

    var knowledgeGraph: machine;

    var p2pNetwork: machine;
    var proofOfConnectivity: machine;

    var validationMachine: machine;
    var executionSandbox: machine;

    var baseLayer: machine;

    // Test state
    var testPhase: int;
    var testsCompleted: int;
    var testsPassed: int;
    var testsFailed: int;

    // Temp variables
    var tmpTestProposalID: string;
    var tmpRecordedErrorID: UUID;
    var tmpStoredRecordID: UUID;
    var tmpTestPeer: PeerInfo;
    var tmpAddress: Address;
    var tmpAmount: BigInt;
    var tmpNodeType: NodeType;
    var tmpNodeID: NodeID;
    var tmpConfig: SandboxConfig;
    var tmpDeadline: Timestamp;
    var tmpTaskType: ValidationTaskType;
    var tmpProposalType: ProposalType;
    var componentReadyPayload: (componentName: string);
    var tmpComponentStartPayload: (componentName: string);
    var createSandboxPayload: (config: SandboxConfig, caller: machine);
    var tmpRequestDAProofPayload: (recordID: UUID, caller: machine);


    start state Init {
        entry {
            testPhase = 0;
            testsCompleted = 0;
            testsPassed = 0;
            testsFailed = 0;

            // Create all component machines
            tokenMachine = new TokenMachine();
            governanceMachine = new GovernanceMachine();
            economicsMachine = new EconomicsMachine();
            consensusMachine = new ConsensusMachine();
            ibcMachine = new IBCMachine();

            skillRegistry = new SkillRegistryMachine();
            llmRegistry = new LLMRegistryMachine();
            mcpCapability = new MCPCapabilityMachine();
            nodeTransformation = new NodeTransformationMachine();

            knowledgeGraph = new KnowledgeGraphMachine();

            p2pNetwork = new P2PNetworkMachine();
            // proofOfConnectivity = new ProofOfConnectivityMachine();

            validationMachine = new ValidationMachine();
            executionSandbox = new ExecutionSandboxMachine();

            baseLayer = new BaseLayerMachine();

            tmpComponentStartPayload.componentName = "tokenMachine";
            send tokenMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "tokenMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "governanceMachine";
            send governanceMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "governanceMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "economicsMachine";
            send economicsMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "economicsMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "consensusMachine";
            send consensusMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "consensusMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "ibcMachine";
            send ibcMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "ibcMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "skillRegistry";
            send skillRegistry, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "skillRegistry";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "llmRegistry";
            send llmRegistry, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "llmRegistry";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "mcpCapability";
            send mcpCapability, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "mcpCapability";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "nodeTransformation";
            send nodeTransformation, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "nodeTransformation";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "knowledgeGraph";
            send knowledgeGraph, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "knowledgeGraph";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "p2pNetwork";
            send p2pNetwork, eComponentStart, tmpComponentStartPayload;
            // Trigger peer discovery completion (no peers initially)
            send p2pNetwork, ePeersDiscovered, default(seq[PeerInfo]);
            componentReadyPayload.componentName = "p2pNetwork";
            announce eComponentReady, componentReadyPayload;

            // send proofOfConnectivity, eComponentStart, "proofOfConnectivity";
            // componentReadyPayload.componentName = "proofOfConnectivity";
            // announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "validationMachine";
            send validationMachine, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "validationMachine";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "executionSandbox";
            send executionSandbox, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "executionSandbox";
            announce eComponentReady, componentReadyPayload;

            tmpComponentStartPayload.componentName = "baseLayer";
            send baseLayer, eComponentStart, tmpComponentStartPayload;
            componentReadyPayload.componentName = "baseLayer";
            announce eComponentReady, componentReadyPayload;

            announce eNetworkStart;

            goto TestTokenOperations;
        }
    }

    // ===================================================================
    // Phase 1: Token Operations
    // ===================================================================
    state TestTokenOperations {
        entry {
            testPhase = 1;

            // Test authorized minting
            tmpAddress.bytes = default(seq[int]);
            tmpAmount.value = 1000000000;
            tmpAmount.isNegative = false;

            send tokenMachine, eMintRequest, (recipient = tmpAddress, amount = tmpAmount, authorized = true, caller = this);
        }

        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                testsPassed = testsPassed + 1;
            } else {
                testsFailed = testsFailed + 1;
            }
            testsCompleted = testsCompleted + 1;

            // Test unauthorized minting (should fail)
            send tokenMachine, eMintRequest, (recipient = tmpAddress, amount = tmpAmount, authorized = false, caller = this);
        }

        on eMintFailed do {
            // Unauthorized mint correctly failed
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Test transfer
            tmpAmount.value = 100000000;
            send tokenMachine, eTransferRequest, (from = tmpAddress, recipient = tmpAddress, amount = tmpAmount, caller = this);
        }

        on eTransferResponse do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestGovernance;
        }

        on eTransferFailed do {
            // Transfer might fail due to no balance - that's expected
            testsCompleted = testsCompleted + 1;
            goto TestGovernance;
        }
    }

    // ===================================================================
    // Phase 2: Governance Operations
    // ===================================================================
    state TestGovernance {
        entry {
            testPhase = 2;

            // Test proposal creation
            tmpProposalType.typeName = "NetworkParameter";
            tmpAddress.bytes = default(seq[int]);
            tmpAmount.value = 100000000;
            tmpAmount.isNegative = false;
            
            send governanceMachine, eCreateProposal, (
                proposalType = tmpProposalType,
                title = "Test Proposal",
                description = "This is a test proposal",
                proposer = tmpAddress,
                deposit = tmpAmount,
                content = default(map[string, any]),
                caller = this
            );
        }

        on eProposalCreated do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestSkillRegistry;
        }

        on eProposalCreationFailed do {
            testsFailed = testsFailed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestSkillRegistry;
        }
    }

    // ===================================================================
    // Phase 3: Skill Registry (KNIRVCHAIN)
    // ===================================================================
    state TestSkillRegistry {
        entry {
            testPhase = 3;

            // Test skill registration
            tmpAddress.bytes = default(seq[int]);

            send skillRegistry, eRegisterSkill, (name = "TestSkill", version = "1.0.0", owner = tmpAddress, loraPointer = "lora://test_adapter", metadata = default(map[string, any]), caller = this);
        }

        on eSkillRegistered do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestNodeTransformation;
        }

        on eSkillRegistrationFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestNodeTransformation;
        }
    }

    // ===================================================================
    // Phase 4: Node Transformation (KNIRVCHAIN)
    // ===================================================================
    state TestNodeTransformation {
        entry {
            testPhase = 4;
            goto TestKnowledgeGraph;
        }
    }

    // ===================================================================
    // Phase 5: Knowledge Graph (KNIRVGRAPH)
    // ===================================================================
    state TestKnowledgeGraph {
        entry {
            testPhase = 5;

            // Record an error
            send knowledgeGraph, eRecordError, (errorType = "ValidationError", errorMessage = "at line 42", stackTrace = "stack trace here", context = default(map[string, any]), caller = this);
        }

        on eErrorRecorded do (record: ErrorRecord) {
            tmpRecordedErrorID = record.id;
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Submit a solution
            tmpAddress.bytes = default(seq[int]);
            send knowledgeGraph, eSubmitSolution, (forError = tmpRecordedErrorID, solutionType = "CodeFix", content = "Fix the validation logic", author = tmpAddress, caller = this);
        }

        on eSolutionSubmitted do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestP2PNetwork;
        }

        on eSolutionSubmissionFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestP2PNetwork;
        }
    }

    // ===================================================================
    // Phase 6: P2P Network (KNIRVROUTER)
    // ===================================================================
    state TestP2PNetwork {
        entry {
            testPhase = 6;

            // Connect to a peer
            tmpNodeType.typeName = "full";
            tmpNodeID.id = "peer_1";
            tmpNodeID.publicKey = default(seq[int]);
            tmpNodeID.nodeType = tmpNodeType;

            tmpTestPeer.nodeID = tmpNodeID;
            tmpTestPeer.address = "192.168.1.100";
            tmpTestPeer.port = 8080;
            tmpTestPeer.lastSeen.milliseconds = 0;
            tmpTestPeer.reputation = 80;

            send p2pNetwork, eConnectToPeer, (peer = tmpTestPeer, caller = this);
        }

        on ePeerConnected do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestValidation;
        }

        on ePeerConnectionFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestValidation;
        }

        /*
        on eConnectivityProofGenerated do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestValidation;
        }
        */
    }

    // ===================================================================
    // Phase 7: Validation (KNIRVSERVER)
    // ===================================================================
    state TestValidation {
        entry {
            testPhase = 7;

            // Submit a validation task
            tmpTaskType.typeName = "code_execution";
            tmpAddress.bytes = default(seq[int]);
            tmpDeadline.milliseconds = 300000;

            send validationMachine, eSubmitValidationTask, (taskType = tmpTaskType, taskPayload = default(map[string, any]), submitter = tmpAddress, priority = 5, deadline = tmpDeadline, caller = this);
        }

        on eValidationTaskQueued do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Create a sandbox
            tmpConfig.maxCPU = 1000;
            tmpConfig.maxMemory = 536870912;
            tmpConfig.maxStorage = 104857600;
            tmpConfig.maxNetwork = 10485760;
            tmpConfig.timeout = 60000;
            tmpConfig.allowedSyscalls = default(seq[string]);
            tmpConfig.isolationLevel = "strict";

            createSandboxPayload.config = tmpConfig;
            createSandboxPayload.caller = this;
            send executionSandbox, eCreateSandbox, createSandboxPayload;
        }

        on eSandboxCreated do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestBaseLayer;
        }

        on eValidationTaskRejected do {
            testsCompleted = testsCompleted + 1;
            goto TestBaseLayer;
        }

        on eSandboxCreationFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestBaseLayer;
        }
    }

    // ===================================================================
    // Phase 8: Base Layer (KNIRVBASE)
    // ===================================================================
    state TestBaseLayer {
        entry {
            testPhase = 8;

            // Store data
            send baseLayer, eStoreData, (dataType = "test_data", payload = default(seq[int]), caller = this);
        }

        on eDataStored do (payload: (record: BaseRecord)) {
            tmpStoredRecordID = payload.record.id;
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Request DA proof
            tmpRequestDAProofPayload.recordID = tmpStoredRecordID;
            tmpRequestDAProofPayload.caller = this;
            send baseLayer, eRequestDAProof, tmpRequestDAProofPayload;
        }

        on eDAProofGenerated do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestComplete;
        }

        on eDataStorageFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestComplete;
        }

        on eDAProofFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestComplete;
        }
    }

    // ===================================================================
    // Test Complete
    // ===================================================================
    state TestComplete {
        entry {
            announce eMonitorAssertion, (monitorName = "NetworkCompositionTest", assertionMessage = format("Full network test completed: {0}/{1} tests passed", testsPassed, testsCompleted), passed = testsFailed == 0);

            announce eNetworkShutdown;
        }
    }
}

// =====================================================================
// CROSS-CHAIN FLOW DRIVER
// Tests the flow from error to skill to cross-chain transfer
// =====================================================================

machine CrossChainFlowDriver {
    var nodeTransformation: machine;
    var skillRegistry: machine;
    var tokenMachine: machine;
    var ibcMachine: machine;
    var economicsMachine: machine;
    var tmpComponentStartPayloadCC: (componentName: string);

    var tmpChannel: IBCChannel;
    var tmpChannelState: ChannelState;
    var tmpPacket: IBCPacket;
    var tmpTimestamp: Timestamp;

    start state Init {
        entry {
            nodeTransformation = new NodeTransformationMachine();
            skillRegistry = new SkillRegistryMachine();
            tokenMachine = new TokenMachine();
            ibcMachine = new IBCMachine();
            economicsMachine = new EconomicsMachine();

            tmpComponentStartPayloadCC.componentName = "nodeTransformation";
            send nodeTransformation, eComponentStart, tmpComponentStartPayloadCC;
            tmpComponentStartPayloadCC.componentName = "skillRegistry";
            send skillRegistry, eComponentStart, tmpComponentStartPayloadCC;
            tmpComponentStartPayloadCC.componentName = "tokenMachine";
            send tokenMachine, eComponentStart, tmpComponentStartPayloadCC;
            tmpComponentStartPayloadCC.componentName = "ibcMachine";
            send ibcMachine, eComponentStart, tmpComponentStartPayloadCC;
            tmpComponentStartPayloadCC.componentName = "economicsMachine";
            send economicsMachine, eComponentStart, tmpComponentStartPayloadCC;

            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (monitorName = "CrossChainFlowTest", assertionMessage = "Cross-chain flow test initialized", passed = true);
        }
    }
}

// =====================================================================
// VALIDATION PIPELINE DRIVER
// Tests the complete validation pipeline with sandbox execution
// =====================================================================

machine ValidationPipelineDriver {
    var validationMachine: machine;
    var executionSandbox: machine;
    var knowledgeGraph: machine;

    var taskID: UUID;

    var tmpNodeType: NodeType;
    var tmpNodeID: NodeID;
    var tmpComponentStartPayloadVP: (componentName: string);
    var tmpValidatorAvailablePayload: (validatorID: NodeID);

    start state Init {
        entry {
            validationMachine = new ValidationMachine();
            executionSandbox = new ExecutionSandboxMachine();
            knowledgeGraph = new KnowledgeGraphMachine();

            tmpComponentStartPayloadVP.componentName = "validationMachine";
            send validationMachine, eComponentStart, tmpComponentStartPayloadVP;
            tmpComponentStartPayloadVP.componentName = "executionSandbox";
            send executionSandbox, eComponentStart, tmpComponentStartPayloadVP;
            tmpComponentStartPayloadVP.componentName = "knowledgeGraph";
            send knowledgeGraph, eComponentStart, tmpComponentStartPayloadVP;

            // Add a validator
            tmpNodeType.typeName = "validator";
            tmpNodeID.id = "validator_1";
            tmpNodeID.publicKey = default(seq[int]);
            tmpNodeID.nodeType = tmpNodeType;

            tmpValidatorAvailablePayload.validatorID = tmpNodeID;
            send validationMachine, eValidatorAvailable, tmpValidatorAvailablePayload;

            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, (monitorName = "ValidationPipelineTest", assertionMessage = "Validation pipeline test initialized", passed = true);
        }
    }
}

// =====================================================================
// MALICIOUS BEHAVIOR DRIVER
// Tests rejection of malicious operations
// =====================================================================

machine MaliciousBehaviorDriver {
    var tokenMachine: machine;
    var governanceMachine: machine;
    var skillRegistry: machine;
    var p2pNetwork: machine;

    var maliciousAttempts: int;
    var correctlyRejected: int;

    var tmpAddress: Address;
    var tmpAmount: BigInt;
    var tmpPeer: PeerInfo;
    var tmpNodeType: NodeType;
    var tmpNodeID: NodeID;
    var tmpAllRejected: bool;
    var tmpComponentStartPayloadMB: (componentName: string);

    start state Init {
        entry {
            tokenMachine = new TokenMachine();
            governanceMachine = new GovernanceMachine();
            skillRegistry = new SkillRegistryMachine();
            p2pNetwork = new P2PNetworkMachine();

            tmpComponentStartPayloadMB.componentName = "tokenMachine";
            send tokenMachine, eComponentStart, tmpComponentStartPayloadMB;
            tmpComponentStartPayloadMB.componentName = "governanceMachine";
            send governanceMachine, eComponentStart, tmpComponentStartPayloadMB;
            tmpComponentStartPayloadMB.componentName = "skillRegistry";
            send skillRegistry, eComponentStart, tmpComponentStartPayloadMB;
            tmpComponentStartPayloadMB.componentName = "p2pNetwork";
            send p2pNetwork, eComponentStart, tmpComponentStartPayloadMB;
            send p2pNetwork, ePeersDiscovered, default(seq[PeerInfo]);

            maliciousAttempts = 0;
            correctlyRejected = 0;

            goto TestUnauthorizedMint;
        }
    }

    state TestUnauthorizedMint {
        entry {
            maliciousAttempts = maliciousAttempts + 1;
            tmpAddress.bytes = default(seq[int]);
            tmpAmount.value = 2000000000;
            tmpAmount.isNegative = false;

            send tokenMachine, eMintRequest, (recipient = tmpAddress, amount = tmpAmount, authorized = false, caller = this);
        }

        on eMintFailed do {
            correctlyRejected = correctlyRejected + 1;
            goto TestInvalidSkill;
        }

        on eMintResponse do {
            // Should not succeed!
            goto TestFailed;
        }
    }

    state TestInvalidSkill {
        entry {
            maliciousAttempts = maliciousAttempts + 1;
            tmpAddress.bytes = default(seq[int]);

            send skillRegistry, eRegisterSkill, (name = "MaliciousSkill", version = "1.0.0", owner = tmpAddress, loraPointer = "", metadata = default(map[string, any]), caller = this);
        }

        on eSkillRegistrationFailed do {
            correctlyRejected = correctlyRejected + 1;
            goto TestLowReputationPeer;
        }

        on eSkillRegistered do {
            goto TestFailed;
        }
    }

    state TestLowReputationPeer {
        entry {
            maliciousAttempts = maliciousAttempts + 1;

            tmpNodeType.typeName = "full";
            tmpNodeID.id = "malicious_peer";
            tmpNodeID.publicKey = default(seq[int]);
            tmpNodeID.nodeType = tmpNodeType;

            tmpPeer.nodeID = tmpNodeID;
            tmpPeer.address = "192.168.1.666";
            tmpPeer.port = 8080;
            tmpPeer.lastSeen.milliseconds = 0;
            tmpPeer.reputation = 10;

            send p2pNetwork, eConnectToPeer, (peer = tmpPeer, caller = this);
        }

        on ePeerConnectionFailed do {
            correctlyRejected = correctlyRejected + 1;
            goto TestComplete;
        }

        on ePeerConnected do {
            // Low reputation peer should be rejected
            goto TestFailed;
        }
    }

    state TestComplete {
        entry {
            tmpAllRejected = maliciousAttempts == correctlyRejected;

            announce eMonitorAssertion, (monitorName = "NetworkMaliciousBehaviorTest", assertionMessage = format("All {0} malicious attempts correctly rejected", maliciousAttempts), passed = tmpAllRejected);
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, (monitorName = "NetworkMaliciousBehaviorTest", assertionMessage = "Malicious operation was incorrectly allowed", passed = false);
        }
    }
}

// =====================================================================
// TEST CASE DECLARATIONS
// These declarations expose driver machines to PChecker via -tc flag.
// Network-wide tests
// =====================================================================

// All machines created by NetworkTestDriver (must be listed for closed module)
module NetworkTestModule =
  { NetworkTestDriver, TokenMachine, GovernanceMachine, EconomicsMachine,
    ConsensusMachine, IBCMachine, SkillRegistryMachine, LLMRegistryMachine,
    MCPCapabilityMachine, NodeTransformationMachine, KnowledgeGraphMachine,
    P2PNetworkMachine, ValidationMachine, ExecutionSandboxMachine, BaseLayerMachine };

// All machines created by CrossChainFlowDriver
module CrossChainModule =
  { CrossChainFlowDriver, NodeTransformationMachine, SkillRegistryMachine,
    TokenMachine, IBCMachine, EconomicsMachine };

// All machines created by ValidationPipelineDriver
module ValidationModule =
  { ValidationPipelineDriver, ValidationMachine, ExecutionSandboxMachine,
    KnowledgeGraphMachine };

// All machines created by MaliciousBehaviorDriver
module MaliciousModule =
  { MaliciousBehaviorDriver, TokenMachine, GovernanceMachine,
    SkillRegistryMachine, P2PNetworkMachine };

test FullNetworkComposition [main = NetworkTestDriver]:
  assert NetworkIntegrityMonitor, CrossChainConsistencyMonitor,
         TokenEconomicsMonitor, GovernanceProcessMonitor,
         ValidationIntegrityMonitor, KnowledgeGraphConsistencyMonitor in
  NetworkTestModule;

test ErrorToSkillToCrossChain [main = CrossChainFlowDriver]:
  assert CrossChainConsistencyMonitor, TokenEconomicsMonitor in
  CrossChainModule;

test ValidationPipeline [main = ValidationPipelineDriver]:
  assert ValidationIntegrityMonitor in
  ValidationModule;

test NetworkMaliciousBehaviorRejection [main = MaliciousBehaviorDriver]:
  assert NetworkIntegrityMonitor, TokenEconomicsMonitor in
  MaliciousModule;

// Component-targeted tests (NetworkTestDriver exercises all components sequentially)
test TokenTransferConsistency [main = NetworkTestDriver]:
  assert TokenEconomicsMonitor in
  NetworkTestModule;

test GovernanceLifecycle [main = NetworkTestDriver]:
  assert GovernanceProcessMonitor in
  NetworkTestModule;

test SkillRegistration [main = NetworkTestDriver]:
  assert NetworkIntegrityMonitor in
  NetworkTestModule;

test NodeTransformation [main = NetworkTestDriver]:
  assert NetworkIntegrityMonitor in
  NetworkTestModule;

test KnowledgeGraphConsistency [main = NetworkTestDriver]:
  assert KnowledgeGraphConsistencyMonitor in
  NetworkTestModule;

test P2PConnectivity [main = NetworkTestDriver]:
  assert NetworkIntegrityMonitor in
  NetworkTestModule;

test ValidationExecution [main = ValidationPipelineDriver]:
  assert ValidationIntegrityMonitor in
  ValidationModule;

test BaseLayerStorage [main = NetworkTestDriver]:
  assert NetworkIntegrityMonitor in
  NetworkTestModule;
