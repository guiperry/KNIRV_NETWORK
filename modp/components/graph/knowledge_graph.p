// knowledge_graph.p - Knowledge Graph State Machine for KNIRVGRAPH
// Models the knowledge graphchain for AI errors and solutions
// Based on: KNIRVGRAPH knowledge graph implementation

machine KnowledgeGraphMachine {
    // State variables - Error tracking
    var errorRecords: map[UUID, ErrorRecord];
    var errorsByType: map[string, seq[UUID]];
    var errorsByHash: map[Hash, UUID];

    // State variables - Solution tracking
    var solutionRecords: map[UUID, SolutionRecord];
    var solutionsByError: map[UUID, seq[UUID]];
    var solutionsByAuthor: map[Address, seq[UUID]];

    // State variables - Pattern detection
    var errorPatterns: map[UUID, ErrorPattern];
    var patternsByType: map[string, seq[UUID]];

    // State variables - General knowledge nodes
    var knowledgeNodes: map[UUID, KnowledgeNode];
    var nodeConnections: map[UUID, seq[UUID]];

    // Counters
    var errorCounter: int;
    var solutionCounter: int;
    var patternCounter: int;
    var nodeCounter: int;

    // Configuration
    var minSolutionEffectiveness: int;  // 0-100
    var patternDetectionThreshold: int;  // occurrences before pattern detected
    var maxSolutionsPerError: int;

    // Temporary variables for event handlers
    var tmpErrorHash: Hash;
    var tmpExistingID: UUID;
    var tmpExistingRecord: ErrorRecord;
    var tmpErrorID: UUID;
    var tmpNewRecord: ErrorRecord;
    var tmpTypeErrors: seq[UUID];
    var tmpErrorRecord: ErrorRecord;
    var tmpSolutionID: UUID;
    var tmpNewSolution: SolutionRecord;
    var tmpErrorSolutions: seq[UUID];
    var tmpAuthorSolutions: seq[UUID];
    var tmpSolutionContext: map[string, any];
    var tmpSolution: SolutionRecord;
    var tmpResultNodes: seq[KnowledgeNode];
    var tmpResultPatterns: seq[ErrorPattern];
    var tmpErrorType: string;
    var tmpErrorIDs: seq[UUID];
    var tmpI: int;
    var tmpErrorIDStr: string;
    var tmpQueryErrorID: UUID;
    var tmpSolutionIDs: seq[UUID];
    var tmpPatternID: UUID;
    var tmpPatternIDs: seq[UUID];
    var tmpPattern: ErrorPattern;
    var tmpErrorTypes: seq[string];
    var tmpNewPattern: ErrorPattern;
    var tmpTypePatterns: seq[UUID];
    var tmpNewNode: KnowledgeNode;
    var tmpConnections: seq[UUID];
    var tmpNode: KnowledgeNode;

    start state Init {
        entry {
            errorRecords = default(map[UUID, ErrorRecord]);
            errorsByType = default(map[string, seq[UUID]]);
            errorsByHash = default(map[Hash, UUID]);

            solutionRecords = default(map[UUID, SolutionRecord]);
            solutionsByError = default(map[UUID, seq[UUID]]);
            solutionsByAuthor = default(map[Address, seq[UUID]]);

            errorPatterns = default(map[UUID, ErrorPattern]);
            patternsByType = default(map[string, seq[UUID]]);

            knowledgeNodes = default(map[UUID, KnowledgeNode]);
            nodeConnections = default(map[UUID, seq[UUID]]);

            errorCounter = 0;
            solutionCounter = 0;
            patternCounter = 0;
            nodeCounter = 0;

            minSolutionEffectiveness = 50;
            patternDetectionThreshold = 5;
            maxSolutionsPerError = 10;
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
        // ERROR RECORDING
        // =====================================================================

        on eRecordError do (payload: (
            errorType: string,
            errorMessage: string,
            context: map[string, any]
        )) {
            // Compute error hash for deduplication
            tmpErrorHash.bytes = default(seq[int]);  // Simplified hash

            // Check for duplicate
            if (tmpErrorHash in errorsByHash) {
                tmpExistingID = errorsByHash[tmpErrorHash];

                tmpExistingRecord = errorRecords[tmpExistingID];
                tmpExistingRecord.occurrences = tmpExistingRecord.occurrences + 1;
                tmpExistingRecord.lastSeen.milliseconds = 0;
                errorRecords[tmpExistingID] = tmpExistingRecord;

                announce eErrorDuplicate, (tmpExistingID, tmpExistingRecord.occurrences);
                send this, eErrorDuplicate, (tmpExistingID, tmpExistingRecord.occurrences);

                // Check for pattern
                CheckForPattern(payload.errorType);
                return;
            }

            // Create new error record
            errorCounter = errorCounter + 1;
            tmpErrorID.value = format("error_rec_{0}", errorCounter);

            tmpNewRecord.id = tmpErrorID;
            tmpNewRecord.errorHash = tmpErrorHash;
            tmpNewRecord.errorType = payload.errorType;
            tmpNewRecord.stackTrace = payload.stackTrace;
            tmpNewRecord.context = payload.context;
            tmpNewRecord.occurrences = 1;
            tmpNewRecord.firstSeen.milliseconds = 0;
            tmpNewRecord.lastSeen.milliseconds = 0;
            tmpNewRecord.solutions = default(seq[UUID]);

            errorRecords[tmpErrorID] = tmpNewRecord;
            errorsByHash[tmpErrorHash] = tmpErrorID;

            // Index by type
            if (payload.errorType in errorsByType) {
                tmpTypeErrors = errorsByType[payload.errorType];
            } else {
                tmpTypeErrors = default(seq[UUID]);
            }
            tmpTypeErrors += (sizeof(tmpTypeErrors), tmpErrorID);
            errorsByType[payload.errorType] = tmpTypeErrors;

            // Create knowledge node
            CreateKnowledgeNode(tmpErrorID, "error", payload.context);

            announce eErrorRecorded, (tmpNewRecord,);
            send this, eErrorRecorded, (tmpNewRecord,);

            // Check for pattern
            CheckForPattern(payload.errorType);
        }

        // =====================================================================
        // SOLUTION MANAGEMENT
        // =====================================================================

        on eSubmitSolution do (payload: (
            forError: UUID,
            solutionType: string,
            content: string,
            author: Address
        )) {
            // Validate error exists
            if (!(payload.forError in errorRecords)) {
                send this, eSolutionSubmissionFailed, ("Error not found",);
                return;
            }

            // Check solution limit
            tmpErrorRecord = errorRecords[payload.forError];

            if (sizeof(tmpErrorRecord.solutions) >= maxSolutionsPerError) {
                send this, eSolutionSubmissionFailed, ("Maximum solutions reached for this error",);
                return;
            }

            // Create solution record
            solutionCounter = solutionCounter + 1;
            tmpSolutionID.value = format("solution_{0}", solutionCounter);

            tmpNewSolution.id = tmpSolutionID;
            tmpNewSolution.forError = payload.forError;
            tmpNewSolution.solutionType = payload.solutionType;
            tmpNewSolution.content = payload.content;
            tmpNewSolution.author = payload.author;
            tmpNewSolution.effectiveness = 0;  // Starts at 0, updated by validation
            tmpNewSolution.validations = 0;
            tmpNewSolution.timestamp.milliseconds = 0;

            solutionRecords[tmpSolutionID] = tmpNewSolution;

            // Link to error
            tmpErrorRecord.solutions += (sizeof(tmpErrorRecord.solutions), tmpSolutionID);
            errorRecords[payload.forError] = tmpErrorRecord;

            // Index by error
            if (payload.forError in solutionsByError) {
                tmpErrorSolutions = solutionsByError[payload.forError];
            } else {
                tmpErrorSolutions = default(seq[UUID]);
            }
            tmpErrorSolutions += (sizeof(tmpErrorSolutions), tmpSolutionID);
            solutionsByError[payload.forError] = tmpErrorSolutions;

            // Index by author
            if (payload.author in solutionsByAuthor) {
                tmpAuthorSolutions = solutionsByAuthor[payload.author];
            } else {
                tmpAuthorSolutions = default(seq[UUID]);
            }
            tmpAuthorSolutions += (sizeof(tmpAuthorSolutions), tmpSolutionID);
            solutionsByAuthor[payload.author] = tmpAuthorSolutions;

            // Create knowledge node
            tmpSolutionContext = default(map[string, any]);
            tmpSolutionContext["solution_type"] = payload.solutionType;
            CreateKnowledgeNode(tmpSolutionID, "solution", tmpSolutionContext);

            // Link nodes
            LinkKnowledgeNodes(payload.forError, tmpSolutionID);

            announce eSolutionSubmitted, (tmpNewSolution,);
            send this, eSolutionSubmitted, (tmpNewSolution,);
        }

        on eValidateSolution do (payload: (
            solutionID: UUID,
            validator: Address,
            effective: bool
        )) {
            if (!(payload.solutionID in solutionRecords)) {
                return;
            }

            tmpSolution = solutionRecords[payload.solutionID];
            tmpSolution.validations = tmpSolution.validations + 1;

            // Update effectiveness score
            if (payload.effective) {
                // Weighted average towards 100
                tmpSolution.effectiveness = (tmpSolution.effectiveness * (tmpSolution.validations - 1) + 100) / tmpSolution.validations;
            } else {
                // Weighted average towards 0
                tmpSolution.effectiveness = (tmpSolution.effectiveness * (tmpSolution.validations - 1)) / tmpSolution.validations;
            }

            solutionRecords[payload.solutionID] = tmpSolution;

            announce eSolutionValidated, (payload.solutionID, tmpSolution.effectiveness);
            send this, eSolutionValidated, (payload.solutionID, tmpSolution.effectiveness);
        }

        // =====================================================================
        // KNOWLEDGE GRAPH QUERIES
        // =====================================================================

        on eQueryKnowledgeGraph do (payload: (
            queryType: string,
            params: map[string, any]
        )) {
            tmpResultNodes = default(seq[KnowledgeNode]);
            tmpResultPatterns = default(seq[ErrorPattern]);

            if (payload.queryType == "errors_by_type") {
                if ("type" in payload.params) {
                    tmpErrorType = payload.params["type"];

                    if (tmpErrorType in errorsByType) {
                        tmpErrorIDs = errorsByType[tmpErrorType];

                        tmpI = 0;
                        while (tmpI < sizeof(tmpErrorIDs) && tmpI < 100) {
                            if (tmpErrorIDs[tmpI] in knowledgeNodes) {
                                tmpResultNodes += (sizeof(tmpResultNodes), knowledgeNodes[tmpErrorIDs[tmpI]]);
                            }
                            tmpI = tmpI + 1;
                        }
                    }
                }
            } else if (payload.queryType == "solutions_for_error") {
                if ("error_id" in payload.params) {
                    tmpErrorIDStr = payload.params["error_id"];
                    tmpQueryErrorID.value = tmpErrorIDStr;

                    if (tmpQueryErrorID in solutionsByError) {
                        tmpSolutionIDs = solutionsByError[tmpQueryErrorID];

                        tmpI = 0;
                        while (tmpI < sizeof(tmpSolutionIDs)) {
                            if (tmpSolutionIDs[tmpI] in knowledgeNodes) {
                                tmpResultNodes += (sizeof(tmpResultNodes), knowledgeNodes[tmpSolutionIDs[tmpI]]);
                            }
                            tmpI = tmpI + 1;
                        }
                    }
                }
            } else if (payload.queryType == "patterns") {
                foreach (patternID in keys(errorPatterns)) {
                    tmpResultPatterns += (sizeof(tmpResultPatterns), errorPatterns[patternID]);
                }
            }

            send this, eKnowledgeGraphResult, (tmpResultNodes, tmpResultPatterns);
        }

        // =====================================================================
        // NODE LINKING
        // =====================================================================

        on eLinkErrorToSolution do (payload: (errorID: UUID, solutionID: UUID)) {
            LinkKnowledgeNodes(payload.errorID, payload.solutionID);
            send this, eErrorSolutionLinked, (payload.errorID, payload.solutionID);
        }

        on eLinkNodes do (payload: (sourceID: UUID, targetID: UUID, linkType: string)) {
            LinkKnowledgeNodes(payload.sourceID, payload.targetID);
            send this, eNodesLinked, (payload.sourceID, payload.targetID);
        }

        on eNetworkShutdown do {
            goto Halted;
        }

        on eEmergencyHalt do {
            goto Halted;
        }
    }

    state Halted {
        on eEmergencyResume do {
            goto Active;
        }
    }

    // =====================================================================
    // HELPER FUNCTIONS
    // =====================================================================

    fun CheckForPattern(errorType: string) {
        if (!(errorType in errorsByType)) {
            return;
        }

        tmpTypeErrors = errorsByType[errorType];

        if (sizeof(tmpTypeErrors) >= patternDetectionThreshold) {
            // Check if pattern already detected
            if (errorType in patternsByType) {
                // Update existing pattern
                tmpPatternIDs = patternsByType[errorType];
                if (sizeof(tmpPatternIDs) > 0) {
                    tmpPatternID = tmpPatternIDs[0];
                    if (tmpPatternID in errorPatterns) {
                        tmpPattern = errorPatterns[tmpPatternID];
                        tmpPattern.frequency = sizeof(tmpTypeErrors);
                        errorPatterns[tmpPatternID] = tmpPattern;

                        announce ePatternUpdated, (tmpPatternID, tmpPattern.frequency);
                    }
                }
            } else {
                // Create new pattern
                patternCounter = patternCounter + 1;
                tmpPatternID.value = format("pattern_{0}", patternCounter);

                tmpErrorTypes = default(seq[string]);
                tmpErrorTypes += (0, errorType);

                tmpNewPattern.id = tmpPatternID;
                tmpNewPattern.patternName = format("Pattern_{0}", errorType);
                tmpNewPattern.description = format("Recurring {0} errors detected", errorType);
                tmpNewPattern.errorTypes = tmpErrorTypes;
                tmpNewPattern.frequency = sizeof(tmpTypeErrors);
                tmpNewPattern.suggestedSolutions = default(seq[UUID]);

                errorPatterns[tmpPatternID] = tmpNewPattern;

                tmpTypePatterns = default(seq[UUID]);
                tmpTypePatterns += (0, tmpPatternID);
                patternsByType[errorType] = tmpTypePatterns;

                announce ePatternDetected, (tmpNewPattern,);
            }
        }
    }

    fun CreateKnowledgeNode(id: UUID, nodeType: string, content: map[string, any]) {
        nodeCounter = nodeCounter + 1;

        tmpNewNode.id = id;
        tmpNewNode.nodeType.typeName = nodeType;
        tmpNewNode.content = content;
        tmpNewNode.embedding = default(seq[float]);
        tmpNewNode.connections = default(seq[UUID]);
        tmpNewNode.createdAt.milliseconds = 0;
        tmpNewNode.updatedAt.milliseconds = 0;

        knowledgeNodes[id] = tmpNewNode;
        nodeConnections[id] = default(seq[UUID]);
    }

    fun LinkKnowledgeNodes(sourceID: UUID, targetID: UUID) {
        if (sourceID in nodeConnections) {
            tmpConnections = nodeConnections[sourceID];
            tmpConnections += (sizeof(tmpConnections), targetID);
            nodeConnections[sourceID] = tmpConnections;

            // Update knowledge node
            if (sourceID in knowledgeNodes) {
                tmpNode = knowledgeNodes[sourceID];
                tmpNode.connections = tmpConnections;
                knowledgeNodes[sourceID] = tmpNode;
            }
        }

        // Bidirectional link
        if (targetID in nodeConnections) {
            tmpConnections = nodeConnections[targetID];
            tmpConnections += (sizeof(tmpConnections), sourceID);
            nodeConnections[targetID] = tmpConnections;

            if (targetID in knowledgeNodes) {
                tmpNode = knowledgeNodes[targetID];
                tmpNode.connections = tmpConnections;
                knowledgeNodes[targetID] = tmpNode;
            }
        }
    }

    // Statistics helpers
    fun GetErrorCount(): int {
        return sizeof(errorRecords);
    }

    fun GetSolutionCount(): int {
        return sizeof(solutionRecords);
    }

    fun GetPatternCount(): int {
        return sizeof(errorPatterns);
    }

    fun GetKnowledgeNodeCount(): int {
        return sizeof(knowledgeNodes);
    }
}
