// rollout.p - observe -> enforce rollout, per tier (AUTHORITY_SPEC.md).
//   observe: a record failure is counted and logged; nothing is ever rejected.
//   enforce: an action with no verifying record is rejected, with one
//            exception: a break-glass record (long-term key, csprng entropy)
//            is accepted but always flagged.
// "valid" means the action carries a record that verifies.
//
// `defect` selects a deliberately broken variant for the negative tests:
//   0 correct, 1 rejects while observing, 2 accepts an unverified action
//   while enforcing, 3 accepts a break-glass record without flagging it.

event eAuthRolloutSetMode: (tier: int, enforce: bool);
event eAuthRolloutModeApplied: (tier: int, enforce: bool);
event eAuthRolloutAction: (id: int, tier: int, hasRecord: bool, recordVerifies: bool, breakGlass: bool);
event eAuthRolloutAccepted: (id: int, tier: int, valid: bool, breakGlass: bool, flagged: bool);
event eAuthRolloutRejected: (id: int, tier: int, valid: bool);

machine AuthorityRolloutMachine {
    var modes: map[int, bool];
    var defect: int;
    var observedViolations: int;

    start state Init {
        entry (cfg: (defect: int)) {
            modes = default(map[int, bool]);
            defect = cfg.defect;
            observedViolations = 0;
            goto Active;
        }
    }

    state Active {
        on eAuthRolloutSetMode do (m: (tier: int, enforce: bool)) {
            modes[m.tier] = m.enforce;
            announce eAuthRolloutModeApplied, m;
        }

        on eAuthRolloutAction do (a: (id: int, tier: int, hasRecord: bool, recordVerifies: bool, breakGlass: bool)) {
            var valid: bool;
            var enforcing: bool;
            var flagged: bool;
            valid = a.hasRecord && a.recordVerifies;
            enforcing = false;
            if (a.tier in modes) {
                enforcing = modes[a.tier];
            }
            flagged = valid && a.breakGlass;
            if (defect == 3) {
                flagged = false;
            }

            if (!enforcing) {
                if (!valid) {
                    observedViolations = observedViolations + 1;
                    if (defect == 1) {
                        announce eAuthRolloutRejected, (id = a.id, tier = a.tier, valid = valid);
                        return;
                    }
                }
                announce eAuthRolloutAccepted, (id = a.id, tier = a.tier, valid = valid, breakGlass = a.breakGlass, flagged = flagged);
                return;
            }

            if (valid || defect == 2) {
                announce eAuthRolloutAccepted, (id = a.id, tier = a.tier, valid = valid, breakGlass = a.breakGlass, flagged = flagged);
            } else {
                announce eAuthRolloutRejected, (id = a.id, tier = a.tier, valid = valid);
            }
        }
    }
}

spec AuthorityRolloutMonitor observes eAuthRolloutModeApplied, eAuthRolloutAccepted, eAuthRolloutRejected {
    var modes: map[int, bool];

    start state Monitoring {
        entry {
            modes = default(map[int, bool]);
        }

        on eAuthRolloutModeApplied do (m: (tier: int, enforce: bool)) {
            modes[m.tier] = m.enforce;
        }

        on eAuthRolloutAccepted do (a: (id: int, tier: int, valid: bool, breakGlass: bool, flagged: bool)) {
            var enforcing: bool;
            enforcing = false;
            if (a.tier in modes) {
                enforcing = modes[a.tier];
            }
            assert !enforcing || a.valid,
                "under enforce, an action without a verifying record was accepted";
            assert !(a.valid && a.breakGlass) || a.flagged,
                "a break-glass record was accepted without being flagged";
        }

        on eAuthRolloutRejected do (a: (id: int, tier: int, valid: bool)) {
            var enforcing: bool;
            enforcing = false;
            if (a.tier in modes) {
                enforcing = modes[a.tier];
            }
            assert enforcing, "observe mode must never reject";
            assert !a.valid, "an action with a verifying record was rejected";
        }
    }
}

machine AuthorityRolloutWorkload {
    var m: machine;
    var i: int;
    var nextID: int;
    var has: bool;
    var verifies: bool;
    var glass: bool;

    start state Init {
        entry (cfg: (defect: int)) {
            m = new AuthorityRolloutMachine((defect = cfg.defect,));
            nextID = 1;
            i = 0;
            while (i < 10) {
                if (choose(4) == 0) {
                    send m, eAuthRolloutSetMode, (tier = 1 + choose(3), enforce = $);
                } else {
                    has = $;
                    verifies = $;
                    glass = $;
                    send m, eAuthRolloutAction, (id = nextID, tier = 1 + choose(3), hasRecord = has, recordVerifies = verifies, breakGlass = glass);
                    nextID = nextID + 1;
                }
                i = i + 1;
            }
        }
    }
}

machine AuthorityRolloutMain {
    start state Init { entry { new AuthorityRolloutWorkload((defect = 0,)); } }
}
machine AuthorityRolloutObserveRejectBugMain {
    start state Init { entry { new AuthorityRolloutWorkload((defect = 1,)); } }
}
machine AuthorityRolloutEnforceAcceptBugMain {
    start state Init { entry { new AuthorityRolloutWorkload((defect = 2,)); } }
}
machine AuthorityRolloutBreakGlassBugMain {
    start state Init { entry { new AuthorityRolloutWorkload((defect = 3,)); } }
}

module AuthorityRolloutModule = { AuthorityRolloutMain, AuthorityRolloutWorkload, AuthorityRolloutMachine };
module AuthorityRolloutObserveRejectBugModule = { AuthorityRolloutObserveRejectBugMain, AuthorityRolloutWorkload, AuthorityRolloutMachine };
module AuthorityRolloutEnforceAcceptBugModule = { AuthorityRolloutEnforceAcceptBugMain, AuthorityRolloutWorkload, AuthorityRolloutMachine };
module AuthorityRolloutBreakGlassBugModule = { AuthorityRolloutBreakGlassBugMain, AuthorityRolloutWorkload, AuthorityRolloutMachine };

test AuthorityRollout [main = AuthorityRolloutMain]:
    assert AuthorityRolloutMonitor in AuthorityRolloutModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it): each proves the monitor detects one broken rollout rule.
test BrokenAuthorityRolloutObserveReject [main = AuthorityRolloutObserveRejectBugMain]:
    assert AuthorityRolloutMonitor in AuthorityRolloutObserveRejectBugModule;
test BrokenAuthorityRolloutEnforceAccept [main = AuthorityRolloutEnforceAcceptBugMain]:
    assert AuthorityRolloutMonitor in AuthorityRolloutEnforceAcceptBugModule;
test BrokenAuthorityRolloutBreakGlass [main = AuthorityRolloutBreakGlassBugMain]:
    assert AuthorityRolloutMonitor in AuthorityRolloutBreakGlassBugModule;
