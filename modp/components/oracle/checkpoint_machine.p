// checkpoint_machine.p - append-only optimistic checkpoint/finality protocol
// Models merkle-math.md Phases 1-7: provisional -> final|rejected with a
// bounded proof window. MMR leaves are append-only; status is an indexed view.

event eCheckpointSubmit: (checkpointID: int, finalByHeight: int, quorumValid: bool, caller: machine);
event eCheckpointProvisional: (checkpointID: int, leafIndex: int);
event eCheckpointAdmissionRejected: (checkpointID: int, reason: string);
event eCheckpointFinalitySubmit: (checkpointID: int, proofValid: bool, attestationQuorum: bool, caller: machine);
event eCheckpointFinalized: (checkpointID: int, leafIndex: int);
event eCheckpointRejected: (checkpointID: int, leafIndex: int, reason: string);
event eCheckpointFinalityRejected: (checkpointID: int, reason: string);
event eCheckpointOracleTick;
event eCheckpointLeafAppended: (leafIndex: int, checkpointID: int, leafKind: int);

type CheckpointModelRecord = (
    status: int,          // 0=provisional, 1=final, 2=rejected
    checkpointLeaf: int,
    finalByHeight: int,
    terminalLeaf: int
);

machine CheckpointMachine {
    var oracleHeight: int;
    var records: map[int, CheckpointModelRecord];
    var mmrLog: seq[(checkpointID: int, leafKind: int)];
    var nextLeaf: int;
    var id: int;
    var rec: CheckpointModelRecord;

    start state Init {
        entry {
            oracleHeight = 0;
            records = default(map[int, CheckpointModelRecord]);
            mmrLog = default(seq[(checkpointID: int, leafKind: int)]);
            nextLeaf = 0;
        }

        on eComponentStart do {
            goto Active;
        }
    }

    state Active {
        on eCheckpointSubmit do (payload: (checkpointID: int, finalByHeight: int, quorumValid: bool, caller: machine)) {
            if (!payload.quorumValid) {
                send payload.caller, eCheckpointAdmissionRejected, (checkpointID = payload.checkpointID, reason = "author quorum invalid");
                return;
            }
            if (payload.checkpointID in records) {
                send payload.caller, eCheckpointAdmissionRejected, (checkpointID = payload.checkpointID, reason = "duplicate checkpoint");
                return;
            }

            rec.status = 0;
            rec.checkpointLeaf = nextLeaf;
            rec.finalByHeight = payload.finalByHeight;
            rec.terminalLeaf = -1;
            records[payload.checkpointID] = rec;
            mmrLog += (sizeof(mmrLog), (checkpointID = payload.checkpointID, leafKind = 1));
            announce eCheckpointLeafAppended, (leafIndex = nextLeaf, checkpointID = payload.checkpointID, leafKind = 1);
            send payload.caller, eCheckpointProvisional, (checkpointID = payload.checkpointID, leafIndex = nextLeaf);
            nextLeaf = nextLeaf + 1;
        }

        on eCheckpointFinalitySubmit do (payload: (checkpointID: int, proofValid: bool, attestationQuorum: bool, caller: machine)) {
            if (!(payload.checkpointID in records)) {
                send payload.caller, eCheckpointFinalityRejected, (checkpointID = payload.checkpointID, reason = "checkpoint missing");
                return;
            }
            rec = records[payload.checkpointID];
            if (rec.status != 0) {
                send payload.caller, eCheckpointFinalityRejected, (checkpointID = payload.checkpointID, reason = "checkpoint already terminal");
                return;
            }
            if (oracleHeight > rec.finalByHeight || !payload.proofValid || !payload.attestationQuorum) {
                rec.status = 2;
                rec.terminalLeaf = nextLeaf;
                records[payload.checkpointID] = rec;
                mmrLog += (sizeof(mmrLog), (checkpointID = payload.checkpointID, leafKind = 3));
                announce eCheckpointLeafAppended, (leafIndex = nextLeaf, checkpointID = payload.checkpointID, leafKind = 3);
                send payload.caller, eCheckpointRejected, (checkpointID = payload.checkpointID, leafIndex = nextLeaf, reason = "proof/window failure");
                nextLeaf = nextLeaf + 1;
                return;
            }

            rec.status = 1;
            rec.terminalLeaf = nextLeaf;
            records[payload.checkpointID] = rec;
            mmrLog += (sizeof(mmrLog), (checkpointID = payload.checkpointID, leafKind = 2));
            announce eCheckpointLeafAppended, (leafIndex = nextLeaf, checkpointID = payload.checkpointID, leafKind = 2);
            send payload.caller, eCheckpointFinalized, (checkpointID = payload.checkpointID, leafIndex = nextLeaf);
            nextLeaf = nextLeaf + 1;
        }

        on eCheckpointOracleTick do {
            oracleHeight = oracleHeight + 1;
            foreach (id in keys(records)) {
                rec = records[id];
                if (rec.status == 0 && oracleHeight > rec.finalByHeight) {
                    rec.status = 2;
                    rec.terminalLeaf = nextLeaf;
                    records[id] = rec;
                    mmrLog += (sizeof(mmrLog), (checkpointID = id, leafKind = 3));
                    announce eCheckpointLeafAppended, (leafIndex = nextLeaf, checkpointID = id, leafKind = 3);
                    announce eCheckpointRejected, (checkpointID = id, leafIndex = nextLeaf, reason = "window-miss");
                    nextLeaf = nextLeaf + 1;
                }
            }
        }
    }
}

spec CheckpointLifecycleMonitor observes
    eCheckpointProvisional, eCheckpointFinalized, eCheckpointRejected,
    eCheckpointLeafAppended
{
    var statuses: map[int, int];
    var expectedLeaf: int;

    start state Monitoring {
        entry {
            statuses = default(map[int, int]);
            expectedLeaf = 0;
        }

        on eCheckpointLeafAppended do (payload: (leafIndex: int, checkpointID: int, leafKind: int)) {
            assert payload.leafIndex == expectedLeaf, "MMR leaf indices must be append-only and contiguous";
            expectedLeaf = expectedLeaf + 1;
        }

        on eCheckpointProvisional do (payload: (checkpointID: int, leafIndex: int)) {
            assert !(payload.checkpointID in statuses), "checkpoint cannot be admitted twice";
            statuses[payload.checkpointID] = 0;
        }

        on eCheckpointFinalized do (payload: (checkpointID: int, leafIndex: int)) {
            assert payload.checkpointID in statuses, "only an admitted checkpoint can finalize";
            assert statuses[payload.checkpointID] == 0, "finality is terminal";
            statuses[payload.checkpointID] = 1;
        }

        on eCheckpointRejected do (payload: (checkpointID: int, leafIndex: int, reason: string)) {
            assert payload.checkpointID in statuses, "only an admitted checkpoint can be rejected";
            assert statuses[payload.checkpointID] == 0, "rejection is terminal";
            statuses[payload.checkpointID] = 2;
        }
    }
}

machine CheckpointProtocolDriver {
    var checkpoint: machine;
    var componentStart: (componentName: string);

    start state Init {
        entry {
            checkpoint = new CheckpointMachine();
            componentStart.componentName = "checkpointMachine";
            send checkpoint, eComponentStart, componentStart;
            send checkpoint, eCheckpointSubmit, (checkpointID = 1, finalByHeight = 2, quorumValid = true, caller = this);
            goto AwaitFirstProvisional;
        }
    }

    state AwaitFirstProvisional {
        on eCheckpointProvisional do (payload: (checkpointID: int, leafIndex: int)) {
            send checkpoint, eCheckpointFinalitySubmit, (checkpointID = payload.checkpointID, proofValid = true, attestationQuorum = true, caller = this);
            goto AwaitFinal;
        }
    }

    state AwaitFinal {
        on eCheckpointFinalized do (payload: (checkpointID: int, leafIndex: int)) {
            send checkpoint, eCheckpointSubmit, (checkpointID = 2, finalByHeight = 1, quorumValid = true, caller = this);
            goto AwaitExpiringProvisional;
        }
    }

    state AwaitExpiringProvisional {
        on eCheckpointProvisional do (payload: (checkpointID: int, leafIndex: int)) {
            send checkpoint, eCheckpointOracleTick;
            send checkpoint, eCheckpointOracleTick;
            goto AwaitRejection;
        }
    }

    state AwaitRejection {
        on eCheckpointRejected do (payload: (checkpointID: int, leafIndex: int, reason: string)) {
            send checkpoint, eCheckpointFinalitySubmit, (checkpointID = payload.checkpointID, proofValid = true, attestationQuorum = true, caller = this);
            goto AwaitTerminalGuard;
        }
    }

    state AwaitTerminalGuard {
        on eCheckpointFinalityRejected do (payload: (checkpointID: int, reason: string)) {
            goto Done;
        }
    }

    state Done { }
}

module CheckpointProtocolModule = { CheckpointProtocolDriver, CheckpointMachine };

test CheckpointFinalityProtocol [main = CheckpointProtocolDriver]:
    assert CheckpointLifecycleMonitor in CheckpointProtocolModule;
