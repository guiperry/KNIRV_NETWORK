// replication_machine.p - standby follower invariants for
// ORACLE_REPLICATION_SPEC.md section 9.
//
// The machine models the decision half of a follower: a head is observed, then
// either applied (the next link in the chain), ignored, or it diverges. Leaf
// and state-log verification happen in the Oracle; what is modelled here is the
// ordering rule that must hold before any of that data is written.

event eHeadObserved: (headSeq: int, mmrSize: int, prevOk: bool, extendsMmr: bool);
event eHeadApplied: (headSeq: int, mmrSize: int, observedMax: int);
event eReplicationRejected: (headSeq: int);
event eReplicationDiverged: (headSeq: int);
event ePromote: (headSeq: int);
event ePromoted: (headSeq: int);
event ePromotionRefused: (headSeq: int);
event eDivergenceCleared: (headSeq: int);

machine ReplicationMachine {
    var appliedSeq: int;
    var appliedMmr: int;
    var observedMax: int;
    var diverged: bool;
    // allowBadApply=true is the deliberately broken variant: it applies a head
    // that does not extend the local MMR. Only the negative test uses it.
    var allowBadApply: bool;

    start state Init {
        entry (cfg: (allowBadApply: bool)) {
            appliedSeq = 0;
            appliedMmr = 0;
            observedMax = 0;
            diverged = false;
            allowBadApply = cfg.allowBadApply;
            goto Following;
        }
    }

    state Following {
        on eHeadObserved do (h: (headSeq: int, mmrSize: int, prevOk: bool, extendsMmr: bool)) {
            if (h.headSeq > observedMax) {
                observedMax = h.headSeq;
            }
            if (diverged) {
                announce eReplicationRejected, (headSeq = h.headSeq,);
                return;
            }
            // A gap is recoverable off-model (page heads?after_seq); a broken
            // link is not.
            if (!h.prevOk) {
                diverged = true;
                announce eReplicationDiverged, (headSeq = h.headSeq,);
                return;
            }
            if (!h.extendsMmr && !allowBadApply) {
                diverged = true;
                announce eReplicationDiverged, (headSeq = h.headSeq,);
                return;
            }
            if (h.headSeq != appliedSeq + 1) {
                announce eReplicationRejected, (headSeq = h.headSeq,);
                return;
            }
            appliedSeq = h.headSeq;
            appliedMmr = h.mmrSize;
            announce eHeadApplied, (headSeq = appliedSeq, mmrSize = appliedMmr, observedMax = observedMax);
        }

        on ePromote do (p: (headSeq: int)) {
            if (diverged || p.headSeq > appliedSeq) {
                announce ePromotionRefused, (headSeq = p.headSeq,);
                return;
            }
            announce ePromoted, (headSeq = p.headSeq,);
        }

        on eDivergenceCleared do (c: (headSeq: int)) {
            // Absorbing until an operator or governance action clears it.
            diverged = false;
            announce eDivergenceCleared, (headSeq = c.headSeq,);
        }
    }
}

// The monitor does not trust the machine: it re-derives the invariants from
// announced events alone.
spec ReplicationMonitor observes eHeadApplied, ePromoted {
    var appliedSeq: int;
    var appliedMmr: int;
    var diverged: bool;

    start state Monitoring {
        entry {
            appliedSeq = 0;
            appliedMmr = 0;
            diverged = false;
        }

        on eHeadApplied do (h: (headSeq: int, mmrSize: int, observedMax: int)) {
            assert h.headSeq == appliedSeq + 1, "applied heads must be exactly the next link";
            assert h.headSeq <= h.observedMax, "a standby never applies a head it did not observe";
            assert h.mmrSize >= appliedMmr, "an applied head must extend, never shrink, the MMR";
            appliedSeq = h.headSeq;
            appliedMmr = h.mmrSize;
        }

        on ePromoted do (p: (headSeq: int)) {
            assert !diverged, "a diverged standby must never be promoted";
            assert p.headSeq <= appliedSeq, "promotion requires an applied head at or beyond the vote point";
        }
    }
}

machine ReplicationWorkload {
    var follower: machine;
    var nextSeq: int;
    var mmr: int;
    var i: int;
    var broken: bool;

    start state Init {
        entry (cfg: (allowBadApply: bool)) {
            broken = cfg.allowBadApply;
            follower = new ReplicationMachine((allowBadApply = cfg.allowBadApply,));
            nextSeq = 1;
            mmr = 10;
            i = 0;
            while (i < 6) {
                mmr = mmr + 3;
                send follower, eHeadObserved, (headSeq = nextSeq, mmrSize = mmr, prevOk = true, extendsMmr = true);
                if (broken && choose(2) == 0) {
                    // A non-extending head: the broken machine applies it.
                    send follower, eHeadObserved, (headSeq = nextSeq + 1, mmrSize = mmr - 1, prevOk = true, extendsMmr = false);
                    nextSeq = nextSeq + 2;
                } else {
                    nextSeq = nextSeq + 1;
                }
                i = i + 1;
            }
            send follower, ePromote, (headSeq = nextSeq - 1,);
        }
    }
}

machine ReplicationMain {
    start state Init {
        entry {
            new ReplicationWorkload((allowBadApply = false,));
        }
    }
}

machine ReplicationBugMain {
    start state Init {
        entry {
            new ReplicationWorkload((allowBadApply = true,));
        }
    }
}

module ReplicationModule = { ReplicationMain, ReplicationWorkload, ReplicationMachine };
module ReplicationBugModule = { ReplicationBugMain, ReplicationWorkload, ReplicationMachine };

test ReplicationFollower [main = ReplicationMain]:
    assert ReplicationMonitor in ReplicationModule;

// Expected to FAIL (Broken* so the positive prefix never runs it): proves the
// monitor catches an applied non-extending head.
test BrokenReplicationNonExtendingApply [main = ReplicationBugMain]:
    assert ReplicationMonitor in ReplicationBugModule;
