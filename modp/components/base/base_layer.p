// base_layer.p - Base Layer State Machine for KNIRVBASE
// Models the foundational data availability and storage layer
// Based on: KNIRVBASE base layer implementation

machine BaseLayerMachine {
    // State variables - Data storage
    var records: map[UUID, BaseRecord];
    var recordsByType: map[string, seq[UUID]];
    var merkleRoots: map[int, Hash];  // height -> root

    // State variables - Data availability proofs
    var daProofs: map[UUID, DAProof];
    var pendingProofs: set[UUID];

    // State variables - Layer state
    var currentHeight: int;
    var currentCommitment: Hash;
    var dataSize: int;

    // Configuration
    var maxRecordSize: int;
    var commitmentInterval: int;
    var proofValidityPeriod: int;

    // Counters
    var recordCounter: int;
    var proofCounter: int;
    var totalDataStored: int;

    // Temp variables for event handlers
    var tmpRecordID: UUID;
    var tmpMerkleProof: seq[Hash];
    var tmpNewRecord: BaseRecord;
    var tmpTypeRecords: seq[UUID];
    var tmpRecord: BaseRecord;
    var tmpProof: DAProof;
    var tmpValid: bool;
    var tmpNewRoot: Hash;

    start state Init {
        entry {
            records = default(map[UUID, BaseRecord]);
            recordsByType = default(map[string, seq[UUID]]);
            merkleRoots = default(map[int, Hash]);

            daProofs = default(map[UUID, DAProof]);
            pendingProofs = default(set[UUID]);

            currentHeight = 0;
            currentCommitment.bytes = default(seq[int]);
            dataSize = 0;

            maxRecordSize = 1048576;  // 1MB
            commitmentInterval = 100;  // Every 100 records
            proofValidityPeriod = 86400000;  // 24 hours

            recordCounter = 0;
            proofCounter = 0;
            totalDataStored = 0;
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
        // DATA STORAGE
        // =====================================================================

        on eStoreData do (payload: (dataType: string, payload: seq[int])) {
            // Check record size
            if (sizeof(payload.payload) > maxRecordSize) {
                send this, eDataStorageFailed, "Record size exceeds maximum";
                return;
            }

            // Create record
            recordCounter = recordCounter + 1;
            tmpRecordID.value = format("record_{0}", recordCounter);

            // Generate merkle proof (simplified)
            tmpMerkleProof = default(seq[Hash]);
            tmpMerkleProof += (0, currentCommitment);

            tmpNewRecord.id = tmpRecordID;
            tmpNewRecord.dataType = payload.dataType;
            tmpNewRecord.payload = payload.payload;
            tmpNewRecord.merkleProof = tmpMerkleProof;
            tmpNewRecord.timestamp.milliseconds = 0;

            // Store record
            records[tmpRecordID] = tmpNewRecord;

            // Index by type
            if (payload.dataType in recordsByType) {
                tmpTypeRecords = recordsByType[payload.dataType];
            } else {
                tmpTypeRecords = default(seq[UUID]);
            }
            tmpTypeRecords += (sizeof(tmpTypeRecords), tmpRecordID);
            recordsByType[payload.dataType] = tmpTypeRecords;

            // Update data size
            totalDataStored = totalDataStored + sizeof(payload.payload);
            dataSize = dataSize + sizeof(payload.payload);

            // Check if we need to commit
            if (recordCounter % commitmentInterval == 0) {
                CommitLayer();
            }

            announce eDataStored, tmpNewRecord;
            send this, eDataStored, tmpNewRecord;
        }

        // =====================================================================
        // DATA RETRIEVAL
        // =====================================================================

        on eRetrieveData do (payload: (recordID: UUID)) {
            if (payload.recordID in records) {
                send this, eDataRetrieved, records[payload.recordID];
            } else {
                send this, eDataNotFound, payload.recordID;
            }
        }

        // =====================================================================
        // DATA AVAILABILITY PROOFS
        // =====================================================================

        on eRequestDAProof do (payload: (recordID: UUID)) {
            // Check record exists
            if (!(payload.recordID in records)) {
                send this, eDAProofFailed, (payload.recordID, "Record not found");
                return;
            }

            tmpRecord = records[payload.recordID];

            // Generate DA proof
            proofCounter = proofCounter + 1;

            tmpProof.recordID = payload.recordID;
            tmpProof.commitmentRoot = currentCommitment;
            tmpProof.proofPath = tmpRecord.merkleProof;
            tmpProof.validator.bytes = default(seq[int]);
            tmpProof.signature = default(seq[int]);

            daProofs[payload.recordID] = tmpProof;

            announce eDAProofGenerated, tmpProof;
            send this, eDAProofGenerated, tmpProof;
        }

        on eVerifyDAProof do (payload: (proof: DAProof)) {
            // Verify proof validity
            tmpValid = true;

            // Check record exists
            if (!(payload.proof.recordID in records)) {
                tmpValid = false;
            }

            // Check commitment root matches
            if (tmpValid) {
                // In real implementation, verify merkle path
                // For model, simplified check
                if (sizeof(payload.proof.proofPath) == 0) {
                    tmpValid = false;
                }
            }

            announce eDAProofVerified, (payload.proof.recordID, tmpValid);
            send this, eDAProofVerified, (payload.proof.recordID, tmpValid);
        }

        // =====================================================================
        // LAYER COMMITMENT
        // =====================================================================

        on eNetworkConfigUpdate do (payload: (config: NetworkConfig)) {
            // Handle config updates
        }

        on eNetworkShutdown do {
            // Final commitment before shutdown
            CommitLayer();
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

    fun CommitLayer() {
        currentHeight = currentHeight + 1;

        // Generate new commitment root (simplified)
        tmpNewRoot.bytes = default(seq[int]);
        // In real implementation, this would be merkle root of all records

        currentCommitment = tmpNewRoot;
        merkleRoots[currentHeight] = tmpNewRoot;

        announce eMonitorAssertion, ("BaseLayer", format("Layer committed at height {0}", currentHeight), true);
    }

    fun GetRecordCount(): int {
        return sizeof(records);
    }

    fun GetDataSize(): int {
        return totalDataStored;
    }

    fun GetCurrentHeight(): int {
        return currentHeight;
    }

    fun GetRecordsByType(dataType: string): seq[UUID] {
        if (dataType in recordsByType) {
            return recordsByType[dataType];
        }
        return default(seq[UUID]);
    }
}
