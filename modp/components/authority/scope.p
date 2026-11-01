// scope.p - scope and tier authority (AUTHORITY_SPEC.md, verify() tail).
// A record is accepted only when, all at once:
//   - the operator is registered,
//   - T1 is scoped to the operator's own address,
//   - T2/T3 are signed by a root operator,
//   - the session certificate's scope covers the record scope,
//   - the operator's registered grant covers the record scope.
// Scopes are ints; an operator's own address is its id; -1 is the wildcard.

event eAuthScopeRegister: (op: int, isRoot: bool, grant: int);
event eAuthScopeRegistered: (op: int, isRoot: bool, grant: int);
event eAuthScopeAction: (op: int, tier: int, scope: int, sessionScope: int);
event eAuthScopeAccepted: (op: int, tier: int, scope: int, sessionScope: int);
event eAuthScopeRejected: (op: int, tier: int, scope: int, sessionScope: int);

type AuthScopeOperator = (isRoot: bool, grant: int);

machine AuthorityScopeMachine {
    var operators: map[int, AuthScopeOperator];
    var skipSessionScope: bool;

    start state Init {
        // skipSessionScope=true is the deliberately broken variant (cross-scope
        // acceptance through a narrower session certificate).
        entry (cfg: (skipSessionScope: bool)) {
            operators = default(map[int, AuthScopeOperator]);
            skipSessionScope = cfg.skipSessionScope;
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
            if (!(a.op in operators)) {
                announce eAuthScopeRejected, a;
                return;
            }
            reg = operators[a.op];
            if (a.tier >= 2 && !reg.isRoot) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (a.tier == 1 && a.scope != a.op) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (!skipSessionScope && a.sessionScope != -1 && a.sessionScope != a.scope) {
                announce eAuthScopeRejected, a;
                return;
            }
            if (reg.grant != -1 && reg.grant != a.scope) {
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
        entry (cfg: (skipSessionScope: bool)) {
            scopeMachine = new AuthorityScopeMachine((skipSessionScope = cfg.skipSessionScope,));
            send scopeMachine, eAuthScopeRegister, (op = 1, isRoot = true, grant = -1);
            send scopeMachine, eAuthScopeRegister, (op = 2, isRoot = false, grant = 2);
            send scopeMachine, eAuthScopeRegister, (op = 3, isRoot = false, grant = 3);
            i = 0;
            while (i < 8) {
                who = 1 + choose(3);
                tier = 1 + choose(3);
                scope = 1 + choose(3);
                sess = choose(5) - 1;
                send scopeMachine, eAuthScopeAction, (op = who, tier = tier, scope = scope, sessionScope = sess);
                i = i + 1;
            }
        }
    }
}

machine AuthorityScopeMain {
    start state Init {
        entry {
            new AuthorityScopeWorkload((skipSessionScope = false,));
        }
    }
}

machine AuthorityScopeBugMain {
    start state Init {
        entry {
            new AuthorityScopeWorkload((skipSessionScope = true,));
        }
    }
}

module AuthorityScopeModule = { AuthorityScopeMain, AuthorityScopeWorkload, AuthorityScopeMachine };
module AuthorityScopeBugModule = { AuthorityScopeBugMain, AuthorityScopeWorkload, AuthorityScopeMachine };

test AuthorityScope [main = AuthorityScopeMain]:
    assert AuthorityScopeMonitor in AuthorityScopeModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it): proves AuthorityScopeMonitor can detect cross-scope acceptance.
test BrokenAuthorityScope [main = AuthorityScopeBugMain]:
    assert AuthorityScopeMonitor in AuthorityScopeBugModule;
