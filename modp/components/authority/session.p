// session.p - session-key retirement (AUTHORITY_SPEC.md, session ceremony).
// A retired session key stays valid only for records issued at or before its
// retirement time. A record's issue time is its certificate's not_before. The
// first retirement of a key wins; a later one never widens the window.

event eAuthSessionRetire: (op: int, session: int, retiredAt: int);
event eAuthSessionRetired: (op: int, session: int, retiredAt: int);
event eAuthSessionRecord: (op: int, session: int, issuedAt: int);
event eAuthSessionAccepted: (op: int, session: int, issuedAt: int);
event eAuthSessionRejected: (op: int, session: int, issuedAt: int);

machine AuthoritySessionMachine {
    var retired: map[int, int];
    var ignoreRetirement: bool;

    start state Init {
        // ignoreRetirement=true is the deliberately broken variant.
        entry (cfg: (ignoreRetirement: bool)) {
            retired = default(map[int, int]);
            ignoreRetirement = cfg.ignoreRetirement;
            goto Active;
        }
    }

    state Active {
        on eAuthSessionRetire do (r: (op: int, session: int, retiredAt: int)) {
            var key: int;
            key = r.op * 100 + r.session;
            if (key in retired) {
                return;
            }
            retired[key] = r.retiredAt;
            announce eAuthSessionRetired, r;
        }

        on eAuthSessionRecord do (r: (op: int, session: int, issuedAt: int)) {
            var key: int;
            key = r.op * 100 + r.session;
            if (!ignoreRetirement && key in retired && r.issuedAt > retired[key]) {
                announce eAuthSessionRejected, r;
                return;
            }
            announce eAuthSessionAccepted, r;
        }
    }
}

spec AuthoritySessionMonitor observes eAuthSessionRetired, eAuthSessionAccepted {
    var retired: map[int, int];

    start state Monitoring {
        entry {
            retired = default(map[int, int]);
        }

        on eAuthSessionRetired do (r: (op: int, session: int, retiredAt: int)) {
            retired[r.op * 100 + r.session] = r.retiredAt;
        }

        on eAuthSessionAccepted do (r: (op: int, session: int, issuedAt: int)) {
            var key: int;
            key = r.op * 100 + r.session;
            assert !(key in retired) || r.issuedAt <= retired[key],
                "a retired session key signed a record issued after its retirement";
        }
    }
}

machine AuthoritySessionWorkload {
    var m: machine;
    var i: int;

    start state Init {
        entry (cfg: (ignoreRetirement: bool)) {
            m = new AuthoritySessionMachine((ignoreRetirement = cfg.ignoreRetirement,));
            i = 0;
            while (i < 9) {
                if (choose(3) == 0) {
                    send m, eAuthSessionRetire, (op = choose(2), session = choose(2), retiredAt = choose(10));
                } else {
                    send m, eAuthSessionRecord, (op = choose(2), session = choose(2), issuedAt = choose(10));
                }
                i = i + 1;
            }
        }
    }
}

machine AuthoritySessionMain {
    start state Init {
        entry {
            new AuthoritySessionWorkload((ignoreRetirement = false,));
        }
    }
}

machine AuthoritySessionBugMain {
    start state Init {
        entry {
            new AuthoritySessionWorkload((ignoreRetirement = true,));
        }
    }
}

module AuthoritySessionModule = { AuthoritySessionMain, AuthoritySessionWorkload, AuthoritySessionMachine };
module AuthoritySessionBugModule = { AuthoritySessionBugMain, AuthoritySessionWorkload, AuthoritySessionMachine };

test AuthoritySessionRetirement [main = AuthoritySessionMain]:
    assert AuthoritySessionMonitor in AuthoritySessionModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it): proves AuthoritySessionMonitor can detect a retired key still signing.
test BrokenAuthoritySessionRetirement [main = AuthoritySessionBugMain]:
    assert AuthoritySessionMonitor in AuthoritySessionBugModule;
