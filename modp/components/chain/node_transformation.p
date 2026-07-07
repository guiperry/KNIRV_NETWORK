// node_transformation.p - Node Transformation Mining State Machine for KNIRVCHAIN
// Models the three core transformation flows:
// 1. ErrorNode -> SkillNode Mining -> LoRA Adapter Pointer
// 2. ContextNode -> CapabilityNode Minting -> MCP Server Pointer
// 3. IdeaNode -> PropertyNode Making -> Inference NFT Pointer
// Based on: KNIRVCHAIN node transformation implementation

machine NodeTransformationMachine {
    // Error nodes pending transformation
    var errorNodes: map[UUID, ErrorNode];
    var pendingSkillMining: map[UUID, Address];  // errorNodeID -> trainer

    // Context nodes pending transformation
    var contextNodes: map[UUID, ContextNode];
    var pendingCapabilityMinting: map[UUID, Address];  // contextNodeID -> creator

    // Idea nodes pending transformation
    var ideaNodes: map[UUID, IdeaNode];
    var ideaVotes: map[UUID, int];
    var pendingPropertyMaking: map[UUID, Address];  // ideaNodeID -> owner

    // Completed transformations
    var skillNodes: map[UUID, SkillNode];
    var capabilityNodes: map[UUID, CapabilityNode];
    var propertyNodes: map[UUID, PropertyNode];

    // Counters
    var errorNodeCounter: int;
    var contextNodeCounter: int;
    var ideaNodeCounter: int;
    var skillNodeCounter: int;
    var capabilityNodeCounter: int;
    var propertyNodeCounter: int;

    // Configuration
    var skillMiningReward: BigInt;
    var capabilityMintingReward: BigInt;
    var propertyMakingReward: BigInt;
    var minIdeaVotesForProperty: int;
    var validationThreshold: int;  // 0-100 score required

    // References
    var skillRegistry: machine;
    var mcpCapability: machine;
    var tokenMachine: machine;

    // Temp variables for event handlers
    var tmpErrorNodeID: UUID;
    var tmpNewErrorNode: ErrorNode;
    var tmpErrorNode: ErrorNode;
    var tmpContextNodeID: UUID;
    var tmpNewContextNode: ContextNode;
    var tmpIdeaNodeID: UUID;
    var tmpNewIdeaNode: IdeaNode;
    var tmpIdeaNode: IdeaNode;
    var tmpSkillNodeID: UUID;
    var tmpNewSkillNode: SkillNode;
    var tmpCapabilityNodeID: UUID;
    var tmpNewCapabilityNode: CapabilityNode;
    var tmpPropertyNodeID: UUID;
    var tmpNewPropertyNode: PropertyNode;
    var tmpTaskID: UUID;
    var tmpTrainer: Address;
    var tmpTimestamp: Timestamp;
    var tmpStatus: NodeStatus;
    var tmpResult: ValidationResult;
    var tmpValidator: NodeID;
    var tmpNodeType: NodeType;
    var tmpResources: ResourceUsage;
    var validationResultPayload: (result: ValidationResult);
    var tmpSkillMiningFailed: SkillMiningFailedData;
    var tmpCapMintingFailed: CapabilityMintingFailedData;
    var tmpPropMakingFailed: PropertyMakingFailedData;
    var iterErrorNodeID: UUID;
    var iterContextNodeID: UUID;
    var iterIdeaNodeID: UUID;
    var tmpSkillNodeMined: SkillNodeMinedData;
    var tmpIdeaVoted: IdeaVotedData;

    start state Init {
        entry {
            errorNodes = default(map[UUID, ErrorNode]);
            pendingSkillMining = default(map[UUID, Address]);
            contextNodes = default(map[UUID, ContextNode]);
            pendingCapabilityMinting = default(map[UUID, Address]);
            ideaNodes = default(map[UUID, IdeaNode]);
            ideaVotes = default(map[UUID, int]);
            pendingPropertyMaking = default(map[UUID, Address]);

            skillNodes = default(map[UUID, SkillNode]);
            capabilityNodes = default(map[UUID, CapabilityNode]);
            propertyNodes = default(map[UUID, PropertyNode]);

            errorNodeCounter = 0;
            contextNodeCounter = 0;
            ideaNodeCounter = 0;
            skillNodeCounter = 0;
            capabilityNodeCounter = 0;
            propertyNodeCounter = 0;

            skillMiningReward.value = 100000000;  // 100 NRN
            skillMiningReward.isNegative = false;
            capabilityMintingReward.value = 50000000;  // 50 NRN
            capabilityMintingReward.isNegative = false;
            propertyMakingReward.value = 25000000;  // 25 NRN
            propertyMakingReward.isNegative = false;
            minIdeaVotesForProperty = 10;
            validationThreshold = 70;
        }

        on eComponentStart do {
            goto Active;
        }

        on eNetworkStart do {
            goto Active;
        }
    }

    state Active {
        // =====================================================================
        // FLOW 1: ErrorNode -> SkillNode Mining
        // =====================================================================

        // Step 1: Submit error (creates ErrorNode)
        on eSubmitError do (payload: (
            errorType: string,
            errorMessage: string,
            context: map[string, any],
            sourceModel: UUID
        )) {
            errorNodeCounter = errorNodeCounter + 1;
            tmpErrorNodeID.value = format("error_{0}", errorNodeCounter);

            tmpTimestamp.milliseconds = 0;
            tmpStatus.status = "pending";
            tmpNewErrorNode.id = tmpErrorNodeID;
            tmpNewErrorNode.errorType = payload.errorType;
            tmpNewErrorNode.errorMessage = payload.errorMessage;
            tmpNewErrorNode.contextPayload = payload.context;
            tmpNewErrorNode.sourceModel = payload.sourceModel;
            tmpNewErrorNode.timestamp = tmpTimestamp;
            tmpNewErrorNode.status = tmpStatus;

            errorNodes[tmpErrorNodeID] = tmpNewErrorNode;

            announce eErrorNodeCreated, tmpNewErrorNode;
            send this, eErrorNodeCreated, tmpNewErrorNode;
        }

        // Step 2: Mine skill from error
        on eMineSkillFromError do (payload: (
            errorNodeID: UUID,
            trainer: Address,
            loraAdapter: string
        )) {
            // Validate error node exists
            if (!(payload.errorNodeID in errorNodes)) {
                tmpSkillMiningFailed.errorNodeID = payload.errorNodeID;
                tmpSkillMiningFailed.reason = "Error node not found";
                send this, eSkillMiningFailed, tmpSkillMiningFailed;
                return;
            }

            tmpErrorNode = errorNodes[payload.errorNodeID];

            // Check not already being mined
            if (payload.errorNodeID in pendingSkillMining) {
                tmpSkillMiningFailed.errorNodeID = payload.errorNodeID;
                tmpSkillMiningFailed.reason = "Already being mined";
                send this, eSkillMiningFailed, tmpSkillMiningFailed;
                return;
            }

            // Validate LoRA adapter
            if (payload.loraAdapter == "") {
                tmpSkillMiningFailed.errorNodeID = payload.errorNodeID;
                tmpSkillMiningFailed.reason = "Invalid LoRA adapter";
                send this, eSkillMiningFailed, tmpSkillMiningFailed;
                return;
            }

            // Mark as pending mining
            pendingSkillMining[payload.errorNodeID] = payload.trainer;

            // Simulate validation (in real system, this would involve validation network)
            // For now, auto-validate
            goto ValidatingSkill;
        }

        // Step 3: Complete skill mining (after validation)
        on eValidationComplete do (payload: (result: ValidationResult)) {
            // Check if this is for skill mining
            tmpTaskID = payload.result.taskID;

            // For skill mining validation
            if (payload.result.score >= validationThreshold && payload.result.passed) {
                // Find which error node this is for
                foreach (iterErrorNodeID in keys(pendingSkillMining)) {
                    tmpTrainer = pendingSkillMining[iterErrorNodeID];

                    // Create skill node
                    skillNodeCounter = skillNodeCounter + 1;
                    tmpSkillNodeID.value = format("skillnode_{0}", skillNodeCounter);

                    tmpErrorNode = errorNodes[iterErrorNodeID];

                    tmpTimestamp.milliseconds = 0;
                    tmpNewSkillNode.id = tmpSkillNodeID;
                    tmpNewSkillNode.derivedFrom = iterErrorNodeID;
                    tmpNewSkillNode.loraAdapter = "lora://trained_adapter";  // Would be real pointer
                    tmpNewSkillNode.trainer = tmpTrainer;
                    tmpNewSkillNode.validationScore = payload.result.score;
                    tmpNewSkillNode.reward = skillMiningReward;
                    tmpNewSkillNode.timestamp = tmpTimestamp;

                    skillNodes[tmpSkillNodeID] = tmpNewSkillNode;

                    // Update error node status
                    tmpErrorNode.status.status = "transformed";
                    errorNodes[iterErrorNodeID] = tmpErrorNode;

                    // Remove from pending
                    pendingSkillMining -= iterErrorNodeID;

                    // Announce success with reward
                    tmpSkillNodeMined.skillNode = tmpNewSkillNode;
                    tmpSkillNodeMined.reward = skillMiningReward;
                    announce eSkillNodeMined, tmpSkillNodeMined;
                    send this, eSkillNodeMined, tmpSkillNodeMined;

                    break;
                }
            }
        }

        // =====================================================================
        // FLOW 2: ContextNode -> CapabilityNode Minting
        // =====================================================================

        // Step 1: Submit context (creates ContextNode)
        on eSubmitContext do (payload: (
            contextData: map[string, any],
            sourceInteraction: UUID
        )) {
            contextNodeCounter = contextNodeCounter + 1;
            tmpContextNodeID.value = format("context_{0}", contextNodeCounter);

            tmpTimestamp.milliseconds = 0;
            tmpNewContextNode.id = tmpContextNodeID;
            tmpNewContextNode.contextData = payload.contextData;
            tmpNewContextNode.sourceInteraction = payload.sourceInteraction;
            tmpNewContextNode.timestamp = tmpTimestamp;

            contextNodes[tmpContextNodeID] = tmpNewContextNode;

            announce eContextNodeCreated, tmpNewContextNode;
            send this, eContextNodeCreated, tmpNewContextNode;
        }

        // Step 2: Mint capability from context
        on eMintCapabilityFromContext do (payload: (
            contextNodeID: UUID,
            creator: Address,
            mcpServer: string
        )) {
            // Validate context node exists
            if (!(payload.contextNodeID in contextNodes)) {
                tmpCapMintingFailed.contextNodeID = payload.contextNodeID;
                tmpCapMintingFailed.reason = "Context node not found";
                send this, eCapabilityMintingFailed, tmpCapMintingFailed;
                return;
            }

            // Validate MCP server pointer
            if (payload.mcpServer == "") {
                tmpCapMintingFailed.contextNodeID = payload.contextNodeID;
                tmpCapMintingFailed.reason = "Invalid MCP server pointer";
                send this, eCapabilityMintingFailed, tmpCapMintingFailed;
                return;
            }

            // Create capability node
            capabilityNodeCounter = capabilityNodeCounter + 1;
            tmpCapabilityNodeID.value = format("capnode_{0}", capabilityNodeCounter);

            tmpTimestamp.milliseconds = 0;
            tmpNewCapabilityNode.id = tmpCapabilityNodeID;
            tmpNewCapabilityNode.derivedFrom = payload.contextNodeID;
            tmpNewCapabilityNode.mcpServer = payload.mcpServer;
            tmpNewCapabilityNode.creator = payload.creator;
            tmpNewCapabilityNode.usageCount = 0;
            tmpNewCapabilityNode.timestamp = tmpTimestamp;

            capabilityNodes[tmpCapabilityNodeID] = tmpNewCapabilityNode;

            announce eCapabilityNodeMinted, tmpNewCapabilityNode;
            send this, eCapabilityNodeMinted, tmpNewCapabilityNode;
        }

        // =====================================================================
        // FLOW 3: IdeaNode -> PropertyNode Making
        // =====================================================================

        // Step 1: Submit idea (creates IdeaNode)
        on eSubmitIdea do (payload: (
            ideaContent: string,
            proposer: Address
        )) {
            ideaNodeCounter = ideaNodeCounter + 1;
            tmpIdeaNodeID.value = format("idea_{0}", ideaNodeCounter);

            tmpTimestamp.milliseconds = 0;
            tmpNewIdeaNode.id = tmpIdeaNodeID;
            tmpNewIdeaNode.ideaContent = payload.ideaContent;
            tmpNewIdeaNode.proposer = payload.proposer;
            tmpNewIdeaNode.votes = 0;
            tmpNewIdeaNode.timestamp = tmpTimestamp;

            ideaNodes[tmpIdeaNodeID] = tmpNewIdeaNode;
            ideaVotes[tmpIdeaNodeID] = 0;

            announce eIdeaNodeCreated, tmpNewIdeaNode;
            send this, eIdeaNodeCreated, tmpNewIdeaNode;
        }

        // Step 2: Vote on idea
        on eVoteOnIdea do (payload: (
            ideaNodeID: UUID,
            voter: Address,
            support: bool
        )) {
            if (!(payload.ideaNodeID in ideaNodes)) {
                return;
            }

            tmpIdeaNode = ideaNodes[payload.ideaNodeID];

            if (payload.support) {
                tmpIdeaNode.votes = tmpIdeaNode.votes + 1;
                ideaVotes[payload.ideaNodeID] = ideaVotes[payload.ideaNodeID] + 1;
            }

            ideaNodes[payload.ideaNodeID] = tmpIdeaNode;

            tmpIdeaVoted.ideaNodeID = payload.ideaNodeID;
            tmpIdeaVoted.votes = tmpIdeaNode.votes;
            announce eIdeaVoted, tmpIdeaVoted;
            send this, eIdeaVoted, tmpIdeaVoted;
        }

        // Step 3: Make property from idea (requires sufficient votes)
        on eMakePropertyFromIdea do (payload: (
            ideaNodeID: UUID,
            owner: Address,
            inferenceNFT: string,
            royaltyRate: int
        )) {
            // Validate idea node exists
            if (!(payload.ideaNodeID in ideaNodes)) {
                tmpPropMakingFailed.ideaNodeID = payload.ideaNodeID;
                tmpPropMakingFailed.reason = "Idea node not found";
                send this, ePropertyMakingFailed, tmpPropMakingFailed;
                return;
            }

            tmpIdeaNode = ideaNodes[payload.ideaNodeID];

            // Check minimum votes
            if (tmpIdeaNode.votes < minIdeaVotesForProperty) {
                tmpPropMakingFailed.ideaNodeID = payload.ideaNodeID;
                tmpPropMakingFailed.reason = format("Insufficient votes: {0}/{1}", tmpIdeaNode.votes, minIdeaVotesForProperty);
                send this, ePropertyMakingFailed, tmpPropMakingFailed;
                return;
            }

            // Validate inference NFT pointer
            if (payload.inferenceNFT == "") {
                tmpPropMakingFailed.ideaNodeID = payload.ideaNodeID;
                tmpPropMakingFailed.reason = "Invalid inference NFT pointer";
                send this, ePropertyMakingFailed, tmpPropMakingFailed;
                return;
            }

            // Validate royalty rate (0-100%)
            if (payload.royaltyRate < 0 || payload.royaltyRate > 100) {
                tmpPropMakingFailed.ideaNodeID = payload.ideaNodeID;
                tmpPropMakingFailed.reason = "Invalid royalty rate";
                send this, ePropertyMakingFailed, tmpPropMakingFailed;
                return;
            }

            // Create property node
            propertyNodeCounter = propertyNodeCounter + 1;
            tmpPropertyNodeID.value = format("property_{0}", propertyNodeCounter);

            tmpTimestamp.milliseconds = 0;
            tmpNewPropertyNode.id = tmpPropertyNodeID;
            tmpNewPropertyNode.derivedFrom = payload.ideaNodeID;
            tmpNewPropertyNode.inferenceNFT = payload.inferenceNFT;
            tmpNewPropertyNode.owner = payload.owner;
            tmpNewPropertyNode.royaltyRate = payload.royaltyRate;
            tmpNewPropertyNode.timestamp = tmpTimestamp;

            propertyNodes[tmpPropertyNodeID] = tmpNewPropertyNode;

            announce ePropertyNodeMade, tmpNewPropertyNode;
            send this, ePropertyNodeMade, tmpNewPropertyNode;
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state ValidatingSkill {
        entry {
            // Auto-complete validation for model
            // In real system, would wait for validation network
            tmpTaskID.value = "validation_task";
            tmpNodeType.typeName = "validator";
            tmpValidator.id = "validator_1";
            tmpValidator.publicKey = default(seq[int]);
            tmpValidator.nodeType = tmpNodeType;
            tmpResources.cpuMillis = 100;
            tmpResources.memoryBytes = 1024;
            tmpResources.storageBytes = 0;
            tmpResources.networkBytes = 0;
            tmpTimestamp.milliseconds = 0;

            tmpResult.taskID = tmpTaskID;
            tmpResult.validator = tmpValidator;
            tmpResult.passed = true;
            tmpResult.score = 85;
            tmpResult.detailsPayload = default(map[string, any]);
            tmpResult.executionTime = 1000;
            tmpResult.resourcesUsed = tmpResources;
            tmpResult.timestamp = tmpTimestamp;

            validationResultPayload.result = tmpResult;
            send this, eValidationComplete, validationResultPayload;
            goto Active;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Active;
        }
    }

    // Helper functions
    fun GetErrorNodeCount(): int {
        return sizeof(errorNodes);
    }

    fun GetSkillNodeCount(): int {
        return sizeof(skillNodes);
    }

    fun GetCapabilityNodeCount(): int {
        return sizeof(capabilityNodes);
    }

    fun GetPropertyNodeCount(): int {
        return sizeof(propertyNodes);
    }

    fun GetTransformationStats(): map[string, int] {
        var stats: map[string, int];
        stats = default(map[string, int]);
        stats["error_nodes"] = sizeof(errorNodes);
        stats["skill_nodes"] = sizeof(skillNodes);
        stats["context_nodes"] = sizeof(contextNodes);
        stats["capability_nodes"] = sizeof(capabilityNodes);
        stats["idea_nodes"] = sizeof(ideaNodes);
        stats["property_nodes"] = sizeof(propertyNodes);
        return stats;
    }
}
