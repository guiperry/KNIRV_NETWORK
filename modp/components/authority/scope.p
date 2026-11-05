// scope.p - scope and tier authority (AUTHORITY_SPEC.md, verify() tail).
// A record is accepted only when, all at once:
//   - the operator is registered,
//   - T1 is scoped to the operator's own address,
//   - T2/T3 are signed by a root operator,
//   - the session certificate's scope covers the record scope,
//   - the operator's registered grant covers the record scope.
// Scopes are ints; an operator's own address is its id; -1 is the wildcard.
//
// `defect` selects a deliberately broken variant, one per rule above, so each rule
// is shown to matter (a monitor assertion that no variant can trip proves nothing):
//   0 correct, 1 ignores the session certificate scope, 2 ignores the registered
//   grant, 3 lets a non-root operator sign T2/T3, 4 lets T1 act outside the
//   operator's own address, 5 accepts an unregistered operator.
//
// Workload coverage: operator 1 is root with a wildcard grant, 2 and 3 are non-root
// with their own grant, 4 is root with a NARROW grant (so the registered-grant rule
// is the deciding check for it), and 5 is never registered.

event eAuthScopeRegister: (op: int, isRoot: bool, grant: int);
event eAuthScopeRegistered: (op: int, isRoot: bool, grant: int);
event eAuthScopeAction: (op: int, tier: int, scope: int, sessionScope: int);
event eAuthScopeAccepted: (op: int, tier: int, scope: int, sessionScope: int);
event eAuthScopeRejected: (op: int, tier: int, scope: int, sessionScope: int);

type AuthScopeOperator = (isRoot: bool, grant: int);

machine AuthorityScopeMachine {
    var operators: map[int, AuthScopeOperator];
    var defect: int;

    start state Init {
        entry (cfg: (defect: int)) {
            operators = default(map[int, AuthScopeOperator]);
            defect = cfg.defect;
            goto Active;
        }
    }

    state Active {
        on eAuthScopeRegister do (r: (op: int, isRoot: bool, grant: int)) {
            operators[r.op] = (isRoot = r.isRoot, grant = r.grant);
            announce eAuthScopeRegistered, r;
        }

        on eAuthScopeAction do (a: (op: int, tier: int, scope: int, sessionScope: int)) {
            var reg: AuthScopeOperator;
            if (a.op in operators) {
                reg = operators[a.op];
            } else {
                if (defect != 5) {
                    announce eAuthScopeRejected, a;
                    return;
                }
                reg = (isRoot = true, grant = -1);
            }
            if (defect != 3 && a.tier >= 2 && !reg.isRoot) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (defect != 4 && a.tier == 1 && a.scope != a.op) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (defect != 1 && a.sessionScope != -1 && a.sessionScope != a.scope) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (defect != 2 && reg.grant != -1 && reg.grant != a.scope) {
                announce eAuthScopeRejected, a;
                return;
            }
            announce eAuthScopeAccepted, a;
        }
    }
}

spec AuthorityScopeMonitor observes eAuthScopeRegistered, eAuthScopeAccepted {
    var operators: map[int, AuthScopeOperator];

    start state Monitoring {
        entry {
            operators = default(map[int, AuthScopeOperator]);
        }

        on eAuthScopeRegistered do (r: (op: int, isRoot: bool, grant: int)) {
            operators[r.op] = (isRoot = r.isRoot, grant = r.grant);
        }

        on eAuthScopeAccepted do (a: (op: int, tier: int, scope: int, sessionScope: int)) {
            var reg: AuthScopeOperator;
            assert a.op in operators, "accepted action from an unregistered operator";
            reg = operators[a.op];
            assert a.tier < 2 || reg.isRoot, "T2/T3 accepted from a non-root operator";
            assert a.tier != 1 || a.scope == a.op, "T1 accepted outside the operator's own scope";
            assert a.sessionScope == -1 || a.sessionScope == a.scope,
                "accepted scope is not covered by the session certificate";
            assert reg.grant == -1 || reg.grant == a.scope,
                "accepted scope is not covered by the operator's registered authority";
        }
    }
}

machine AuthorityScopeWorkload {
    var scopeMachine: machine;
    var i: int;
    var who: int;
    var tier: int;
    var scope: int;
    var sess: int;

    start state Init {
        entry (cfg: (defect: int)) {
            scopeMachine = new AuthorityScopeMachine((defect = cfg.defect,));
            send scopeMachine, eAuthScopeRegister, (op = 1, isRoot = true, grant = -1);
            send scopeMachine, eAuthScopeRegister, (op = 2, isRoot = false, grant = 2);
            send scopeMachine, eAuthScopeRegister, (op = 3, isRoot = false, grant = 3);
            send scopeMachine, eAuthScopeRegister, (op = 4, isRoot = true, grant = 4);
            i = 0;
            while (i < 10) {
                who = 1 + choose(5);
                tier = 1 + choose(3);
                scope = 1 + choose(4);
                sess = choose(6) - 1;
                send scopeMachine, eAuthScopeAction, (op = who, tier = tier, scope = scope, sessionScope = sess);
                i = i + 1;
            }
        }
    }
}

machine AuthorityScopeMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 0,)); } }
}
machine AuthorityScopeBugMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 1,)); } }
}
machine AuthorityGrantIgnoredBugMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 2,)); } }
}
machine AuthorityTierRootIgnoredBugMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 3,)); } }
}
machine AuthorityT1OwnScopeIgnoredBugMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 4,)); } }
}
machine AuthorityUnregisteredAcceptedBugMain {
    start state Init { entry { new AuthorityScopeWorkload((defect = 5,)); } }
}

module AuthorityScopeModule = { AuthorityScopeMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityScopeBugModule = { AuthorityScopeBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityGrantIgnoredBugModule = { AuthorityGrantIgnoredBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityTierRootIgnoredBugModule = { AuthorityTierRootIgnoredBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityT1OwnScopeIgnoredBugModule = { AuthorityT1OwnScopeIgnoredBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityUnregisteredAcceptedBugModule = { AuthorityUnregisteredAcceptedBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };

test AuthorityScope [main = AuthorityScopeMain]:
    assert AuthorityScopeMonitor in AuthorityScopeModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it).
// Each proves AuthorityScopeMonitor detects one broken scope rule. None of these names
// starts with another test's name, because P selects tests by prefix.
test BrokenAuthorityScope [main = AuthorityScopeBugMain]:
    assert AuthorityScopeMonitor in AuthorityScopeBugModule;
test BrokenAuthorityGrantIgnored [main = AuthorityGrantIgnoredBugMain]:
    assert AuthorityScopeMonitor in AuthorityGrantIgnoredBugModule;
test BrokenAuthorityTierRootIgnored [main = AuthorityTierRootIgnoredBugMain]:
    assert AuthorityScopeMonitor in AuthorityTierRootIgnoredBugModule;
test BrokenAuthorityT1OwnScopeIgnored [main = AuthorityT1OwnScopeIgnoredBugMain]:
    assert AuthorityScopeMonitor in AuthorityT1OwnScopeIgnoredBugModule;
test BrokenAuthorityUnregisteredAccepted [main = AuthorityUnregisteredAcceptedBugMain]:
    assert AuthorityScopeMonitor in AuthorityUnregisteredAcceptedBugModule;
