// proof_of_connectivity.p - Proof of Connectivity State Machine for KNIRVROUTER
// Models the Proof-of-Connectivity consensus mechanism
// Based on: KNIRVROUTER PoC implementation

machine ProofOfConnectivityMachine {
    // State variables - Proofs
    var activeProofs: map[NodeID, ConnectivityProof];
    var proofHistory: map[NodeID, seq[ConnectivityProof]];
    var pendingChallenges: map[NodeID, NodeID];  // prover -> challenger

    // State variables - Connectivity scores
    var connectivityScores: map[NodeID, int];  // 0-100
    var lastProofTime: map[NodeID, Timestamp];
    var proofCount: map[NodeID, int];

    // Configuration
    var proofValidityPeriod: int;  // milliseconds
    var challengeResponseTime: int;  // milliseconds
    var minConnectivityScore: int;
    var proofReward: BigInt;
    var challengePenalty: BigInt;

    // Network reference
    var p2pNetwork: machine;

    // Counters
    var totalProofsGenerated: int;
    var totalChallengesIssued: int;
    var challengesPassed: int;
    var challengesFailed: int;

    // Temporary variables for event handlers
    var tmpProverNode: NodeID;
    var tmpProofData: seq[int];
    var tmpNewProof: ConnectivityProof;
    var tmpHistory: seq[ConnectivityProof];
    var tmpIsValid: bool;
    var tmpCurrentScore: int;
    var tmpPassed: bool;
    var tmpLargestSize: int;
    var tmpLargestIdx: int;
    var tmpI: int;
    var tmpPartition: seq[NodeID];
    var tmpJ: int;
    var tmpTimestamp: Timestamp;
    var tmpScore: int;

    start state Init {
        entry {
            activeProofs = default(map[NodeID, ConnectivityProof]);
            proofHistory = default(map[NodeID, seq[ConnectivityProof]]);
            pendingChallenges = default(map[NodeID, NodeID]);

            connectivityScores = default(map[NodeID, int]);
            lastProofTime = default(map[NodeID, Timestamp]);
            proofCount = default(map[NodeID, int]);

            proofValidityPeriod = 3600000;  // 1 hour
            challengeResponseTime = 60000;  // 1 minute
            minConnectivityScore = 50;
            proofReward.value = 1000000;  // 1 NRN
            proofReward.isNegative = false;
            challengePenalty.value = 5000000;  // 5 NRN
            challengePenalty.isNegative = false;

            totalProofsGenerated = 0;
            totalChallengesIssued = 0;
            challengesPassed = 0;
            challengesFailed = 0;
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
        // PROOF GENERATION
        // =====================================================================

        on eGenerateConnectivityProof do (payload: (verifier: NodeID)) {
            // Generate a new connectivity proof
            tmpProverNode.id = "local_prover";
            tmpProverNode.publicKey = default(seq[int]);
            tmpProverNode.nodeType.typeName = "full";

            tmpProofData = default(seq[int]);
            // In real implementation, this would include:
            // - Connected peer list
            // - Route table hash
            // - Network metrics
            // - Cryptographic commitment

            tmpNewProof.prover = tmpProverNode;
            tmpNewProof.verifier = payload.verifier;
            tmpNewProof.proofData = tmpProofData;
            tmpNewProof.timestamp.milliseconds = 0;
            tmpNewProof.validUntil.milliseconds = proofValidityPeriod;
            tmpNewProof.signature = default(seq[int]);

            // Store proof
            activeProofs[tmpProverNode] = tmpNewProof;

            // Update proof count
            if (tmpProverNode in proofCount) {
                proofCount[tmpProverNode] = proofCount[tmpProverNode] + 1;
            } else {
                proofCount[tmpProverNode] = 1;
            }

            // Add to history
            if (tmpProverNode in proofHistory) {
                tmpHistory = proofHistory[tmpProverNode];
            } else {
                tmpHistory = default(seq[ConnectivityProof]);
            }
            tmpHistory += (sizeof(tmpHistory), tmpNewProof);
            proofHistory[tmpProverNode] = tmpHistory;

            tmpTimestamp.milliseconds = 0;
            lastProofTime[tmpProverNode] = tmpTimestamp;
            totalProofsGenerated = totalProofsGenerated + 1;

            announce eConnectivityProofGenerated, (tmpNewProof,);
            send this, eConnectivityProofGenerated, (tmpNewProof,);
        }

        // =====================================================================
        // PROOF VERIFICATION
        // =====================================================================

        on eVerifyConnectivityProof do (payload: (proof: ConnectivityProof)) {
            tmpIsValid = true;

            // Verification checks:
            // 1. Proof is not expired
            // 2. Signature is valid
            // 3. Proof data is consistent
            // 4. Prover has sufficient connectivity

            // Check if prover has connectivity score
            if (payload.proof.prover in connectivityScores) {
                if (connectivityScores[payload.proof.prover] < minConnectivityScore) {
                    tmpIsValid = false;
                }
            }

            // Update connectivity score based on proof
            if (tmpIsValid) {
                if (payload.proof.prover in connectivityScores) {
                    // Increase score for valid proof (max 100)
                    tmpCurrentScore = connectivityScores[payload.proof.prover];
                    if (tmpCurrentScore < 100) {
                        connectivityScores[payload.proof.prover] = tmpCurrentScore + 5;
                        if (connectivityScores[payload.proof.prover] > 100) {
                            connectivityScores[payload.proof.prover] = 100;
                        }
                    }
                } else {
                    connectivityScores[payload.proof.prover] = 75;  // Initial score
                }
            }

            announce eConnectivityProofVerified, (payload.proof, tmpIsValid);
            send this, eConnectivityProofVerified, (payload.proof, tmpIsValid);
        }

        // =====================================================================
        // CONNECTIVITY CHALLENGES
        // =====================================================================

        on eConnectivityChallenged do (payload: (prover: NodeID, challenger: NodeID)) {
            // Check if prover has an active proof
            if (!(payload.prover in activeProofs)) {
                // No proof - automatic failure
                UpdateScoreForFailure(payload.prover);
                challengesFailed = challengesFailed + 1;

                send this, eConnectivityChallengeResult, (payload.prover, false);
                return;
            }

            // Store pending challenge
            pendingChallenges[payload.prover] = payload.challenger;
            totalChallengesIssued = totalChallengesIssued + 1;

            // In real system, would wait for response
            // For model, auto-respond based on connectivity score
            tmpPassed = false;

            if (payload.prover in connectivityScores) {
                tmpPassed = connectivityScores[payload.prover] >= minConnectivityScore;
            }

            if (tmpPassed) {
                UpdateScoreForSuccess(payload.prover);
                challengesPassed = challengesPassed + 1;
            } else {
                UpdateScoreForFailure(payload.prover);
                challengesFailed = challengesFailed + 1;
            }

            pendingChallenges -= payload.prover;

            announce eConnectivityChallengeResult, (payload.prover, tmpPassed);
            send this, eConnectivityChallengeResult, (payload.prover, tmpPassed);
        }

        // =====================================================================
        // TOPOLOGY INTEGRATION
        // =====================================================================

        on eTopologySnapshotReady do (payload: (snapshot: TopologySnapshot)) {
            // Update connectivity scores based on topology
            // Nodes with higher connectivity get better scores
            // This is a simplified model
        }

        on eNetworkPartitionDetected do (payload: (partitions: seq[seq[NodeID]])) {
            // Reduce connectivity scores for nodes in smaller partitions
            if (sizeof(payload.partitions) > 1) {
                // Find largest partition
                tmpLargestSize = 0;
                tmpLargestIdx = 0;

                tmpI = 0;
                while (tmpI < sizeof(payload.partitions)) {
                    if (sizeof(payload.partitions[tmpI]) > tmpLargestSize) {
                        tmpLargestSize = sizeof(payload.partitions[tmpI]);
                        tmpLargestIdx = tmpI;
                    }
                    tmpI = tmpI + 1;
                }

                // Penalize nodes not in largest partition
                tmpI = 0;
                while (tmpI < sizeof(payload.partitions)) {
                    if (tmpI != tmpLargestIdx) {
                        tmpPartition = payload.partitions[tmpI];

                        tmpJ = 0;
                        while (tmpJ < sizeof(tmpPartition)) {
                            UpdateScoreForFailure(tmpPartition[tmpJ]);
                            tmpJ = tmpJ + 1;
                        }
                    }
                    tmpI = tmpI + 1;
                }
            }
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

    fun UpdateScoreForSuccess(nodeID: NodeID) {
        if (nodeID in connectivityScores) {
            tmpScore = connectivityScores[nodeID];
            tmpScore = tmpScore + 10;
            if (tmpScore > 100) {
                tmpScore = 100;
            }
            connectivityScores[nodeID] = tmpScore;
        } else {
            connectivityScores[nodeID] = 80;
        }
    }

    fun UpdateScoreForFailure(nodeID: NodeID) {
        if (nodeID in connectivityScores) {
            tmpScore = connectivityScores[nodeID];
            tmpScore = tmpScore - 20;
            if (tmpScore < 0) {
                tmpScore = 0;
            }
            connectivityScores[nodeID] = tmpScore;
        } else {
            connectivityScores[nodeID] = 30;
        }
    }

    // Statistics helpers
    fun GetConnectivityScore(nodeID: NodeID): int {
        if (nodeID in connectivityScores) {
            return connectivityScores[nodeID];
        }
        return 0;
    }

    fun GetTotalProofsGenerated(): int {
        return totalProofsGenerated;
    }

    fun GetChallengeSuccessRate(): int {
        if (totalChallengesIssued == 0) {
            return 100;
        }
        return (challengesPassed * 100) / totalChallengesIssued;
    }
}
