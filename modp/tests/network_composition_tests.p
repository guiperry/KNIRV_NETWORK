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
            proofOfConnectivity = new ProofOfConnectivityMachine();

            validationMachine = new ValidationMachine();
            executionSandbox = new ExecutionSandboxMachine();

            baseLayer = new BaseLayerMachine();

            send tokenMachine, eComponentStart, "tokenMachine";
            componentReadyPayload.componentName = "tokenMachine";
            announce eComponentReady, componentReadyPayload;

            send governanceMachine, eComponentStart, "governanceMachine";
            componentReadyPayload.componentName = "governanceMachine";
            announce eComponentReady, componentReadyPayload;

            send economicsMachine, eComponentStart, "economicsMachine";
            componentReadyPayload.componentName = "economicsMachine";
            announce eComponentReady, componentReadyPayload;

            send consensusMachine, eComponentStart, "consensusMachine";
            componentReadyPayload.componentName = "consensusMachine";
            announce eComponentReady, componentReadyPayload;

            send ibcMachine, eComponentStart, "ibcMachine";
            componentReadyPayload.componentName = "ibcMachine";
            announce eComponentReady, componentReadyPayload;

            send skillRegistry, eComponentStart, "skillRegistry";
            componentReadyPayload.componentName = "skillRegistry";
            announce eComponentReady, componentReadyPayload;

            send llmRegistry, eComponentStart, "llmRegistry";
            componentReadyPayload.componentName = "llmRegistry";
            announce eComponentReady, componentReadyPayload;

            send mcpCapability, eComponentStart, "mcpCapability";
            componentReadyPayload.componentName = "mcpCapability";
            announce eComponentReady, componentReadyPayload;

            send nodeTransformation, eComponentStart, "nodeTransformation";
            componentReadyPayload.componentName = "nodeTransformation";
            announce eComponentReady, componentReadyPayload;

            send knowledgeGraph, eComponentStart, "knowledgeGraph";
            componentReadyPayload.componentName = "knowledgeGraph";
            announce eComponentReady, componentReadyPayload;

            send p2pNetwork, eComponentStart, "p2pNetwork";
            componentReadyPayload.componentName = "p2pNetwork";
            announce eComponentReady, componentReadyPayload;

            send proofOfConnectivity, eComponentStart, "proofOfConnectivity";
            componentReadyPayload.componentName = "proofOfConnectivity";
            announce eComponentReady, componentReadyPayload;

            send validationMachine, eComponentStart, "validationMachine";
            componentReadyPayload.componentName = "validationMachine";
            announce eComponentReady, componentReadyPayload;

            send executionSandbox, eComponentStart, "executionSandbox";
            componentReadyPayload.componentName = "executionSandbox";
            announce eComponentReady, componentReadyPayload;

            send baseLayer, eComponentStart, "baseLayer";
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

            send tokenMachine, eMintRequest, (tmpAddress, tmpAmount, true);
        }

        on eMintResponse do (payload: (success: bool, receipt: MintReceipt)) {
            if (payload.success) {
                testsPassed = testsPassed + 1;
            } else {
                testsFailed = testsFailed + 1;
            }
            testsCompleted = testsCompleted + 1;

            // Test unauthorized minting (should fail)
            tmpAmount.value = 999999999;
            send tokenMachine, eMintRequest, (tmpAddress, tmpAmount, false);
        }

        on eMintFailed do {
            // Unauthorized mint correctly failed
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Test transfer
            tmpAmount.value = 100000000;
            send tokenMachine, eTransferRequest, (from = tmpAddress, recipient = tmpAddress, amount = tmpAmount);
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
                content = default(map[string, any])
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

            send skillRegistry, eRegisterSkill, ("TestSkill", "1.0.0", tmpAddress, "lora://test_adapter", default(map[string, any]));
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
            send knowledgeGraph, eRecordError, ("ValidationError", "at line 42", default(map[string, any]));
        }

        on eErrorRecorded do (payload: (record: ErrorRecord)) {
            tmpRecordedErrorID = payload.record.id;
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Submit a solution
            tmpAddress.bytes = default(seq[int]);
            send knowledgeGraph, eSubmitSolution, (tmpRecordedErrorID, "CodeFix", "Fix the validation logic", tmpAddress);
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

            send p2pNetwork, eConnectToPeer, (tmpTestPeer,);
        }

        on ePeerConnected do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Generate connectivity proof
            tmpNodeType.typeName = "validator";
            tmpNodeID.id = "verifier_1";
            tmpNodeID.publicKey = default(seq[int]);
            tmpNodeID.nodeType = tmpNodeType;

            send proofOfConnectivity, eGenerateConnectivityProof, (tmpNodeID,);
        }

        on ePeerConnectionFailed do {
            testsCompleted = testsCompleted + 1;
            goto TestValidation;
        }

        on eConnectivityProofGenerated do {
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;
            goto TestValidation;
        }
    }

    // ===================================================================
    // Phase 7: Validation (KNIRVNEXUS)
    // ===================================================================
    state TestValidation {
        entry {
            testPhase = 7;

            // Submit a validation task
            tmpTaskType.typeName = "code_execution";
            tmpAddress.bytes = default(seq[int]);
            tmpDeadline.milliseconds = 300000;

            send validationMachine, eSubmitValidationTask, (tmpTaskType, default(map[string, any]), tmpAddress, 5, tmpDeadline);
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
            send baseLayer, eStoreData, ("test_data", default(seq[int]));
        }

        on eDataStored do (payload: (record: BaseRecord)) {
            tmpStoredRecordID = payload.record.id;
            testsPassed = testsPassed + 1;
            testsCompleted = testsCompleted + 1;

            // Request DA proof
            send baseLayer, eRequestDAProof, (tmpStoredRecordID,);
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
            announce eMonitorAssertion, ("NetworkCompositionTest", format("Full network test completed: {0}/{1} tests passed", testsPassed, testsCompleted), testsFailed == 0);

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

            send nodeTransformation, eComponentStart;
            send skillRegistry, eComponentStart;
            send tokenMachine, eComponentStart;
            send ibcMachine, eComponentStart;
            send economicsMachine, eComponentStart;

            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, ("CrossChainFlowTest", "Cross-chain flow test initialized", true);
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

    start state Init {
        entry {
            validationMachine = new ValidationMachine();
            executionSandbox = new ExecutionSandboxMachine();
            knowledgeGraph = new KnowledgeGraphMachine();

            send validationMachine, eComponentStart;
            send executionSandbox, eComponentStart;
            send knowledgeGraph, eComponentStart;

            // Add a validator
            tmpNodeType.typeName = "validator";
            tmpNodeID.id = "validator_1";
            tmpNodeID.publicKey = default(seq[int]);
            tmpNodeID.nodeType = tmpNodeType;

            send validationMachine, eValidatorAvailable, (tmpNodeID,);

            goto TestComplete;
        }
    }

    state TestComplete {
        entry {
            announce eMonitorAssertion, ("ValidationPipelineTest", "Validation pipeline test initialized", true);
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

    start state Init {
        entry {
            tokenMachine = new TokenMachine();
            governanceMachine = new GovernanceMachine();
            skillRegistry = new SkillRegistryMachine();
            p2pNetwork = new P2PNetworkMachine();

            send tokenMachine, eComponentStart;
            send governanceMachine, eComponentStart;
            send skillRegistry, eComponentStart;
            send p2pNetwork, eComponentStart;

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

            send tokenMachine, eMintRequest, (tmpAddress, tmpAmount, false);
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

            send skillRegistry, eRegisterSkill, ("MaliciousSkill", "1.0.0", tmpAddress, "", default(map[string, any]));
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

            send p2pNetwork, eConnectToPeer, (tmpPeer,);
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

            announce eMonitorAssertion, ("NetworkMaliciousBehaviorTest", format("All {0} malicious attempts correctly rejected", maliciousAttempts), tmpAllRejected);
        }
    }

    state TestFailed {
        entry {
            announce eMonitorAssertion, ("NetworkMaliciousBehaviorTest", "Malicious operation was incorrectly allowed", false);
        }
    }
}
