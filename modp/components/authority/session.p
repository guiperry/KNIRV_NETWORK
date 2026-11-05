// session.p - session-key retirement (AUTHORITY_SPEC.md section 4 and section 10 D1).
//
// INVARIANT (plan 1.5): a retired session key is valid only for records ISSUED at
// or before its retirement time. `issuedAt` is an explicit issue time on the
// record, and `certStart` is the not_before of the certificate that signed it
// (certStart <= issuedAt).
//
// IMPLEMENTATION GAP, pending owner decision D1 (not decided here): the
// implementation has NO authenticated issue time. The record signature covers only
// the action digest, so ntime is free (AUTHORITY_SPEC.md section 2.3). Verify
// can therefore only reject a retired session whose CERTIFICATE starts after the
// retirement (cert.not_before > retired_at). A record signed after retirement,
// inside a certificate that started earlier, is accepted. So the implementation
// cannot enforce this invariant per record yet.
//
// This file models the invariant as the plan states it (rule 0) and keeps the
// current implementation rule (rule 2) as a separate variant that is expected to
// violate it. Rule 2 shows that checking the certificate start alone cannot
// satisfy the invariant, whatever D1 decides; it does not decide D1.
//
// The first retirement of a key wins; a later one never widens the window.

event eAuthSessionRetire: (op: int, session: int, retiredAt: int);
event eAuthSessionRetired: (op: int, session: int, retiredAt: int);
event eAuthSessionRecord: (op: int, session: int, issuedAt: int, certStart: int);
event eAuthSessionAccepted: (op: int, session: int, issuedAt: int, certStart: int);
event eAuthSessionRejected: (op: int, session: int, issuedAt: int, certStart: int);

machine AuthoritySessionMachine {
    var retired: map[int, int];
    var rule: int;

    start state Init {
        // rule 0: the plan's invariant, reject when issuedAt > retiredAt.
        // rule 1: deliberately broken, retirement is ignored.
        // rule 2: what the implementation does today, reject only when the
        //         certificate starts after the retirement (certStart > retiredAt).
        entry (cfg: (rule: int)) {
            retired = default(map[int, int]);
            rule = cfg.rule;
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

        on eAuthSessionRecord do (r: (op: int, session: int, issuedAt: int, certStart: int)) {
            var key: int;
            key = r.op * 100 + r.session;
            if (key in retired) {
                if (rule == 0 && r.issuedAt > retired[key]) {
                    announce eAuthSessionRejected, r;
                    return;
                }
                if (rule == 2 && r.certStart > retired[key]) {
                    announce eAuthSessionRejected, r;
                    return;
                }
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

        on eAuthSessionAccepted do (r: (op: int, session: int, issuedAt: int, certStart: int)) {
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
    var began: int;

    start state Init {
        entry (cfg: (rule: int)) {
            m = new AuthoritySessionMachine((rule = cfg.rule,));
            i = 0;
            while (i < 9) {
                if (choose(3) == 0) {
                    send m, eAuthSessionRetire, (op = choose(2), session = choose(2), retiredAt = choose(10));
                } else {
                    // A record is issued at or after the start of its certificate.
                    began = choose(10);
                    send m, eAuthSessionRecord, (op = choose(2), session = choose(2), issuedAt = began + choose(4), certStart = began);
                }
                i = i + 1;
            }
        }
    }
}

machine AuthoritySessionMain {
    start state Init {
        entry {
            new AuthoritySessionWorkload((rule = 0,));
        }
    }
}

machine AuthoritySessionBugMain {
    start state Init {
        entry {
            new AuthoritySessionWorkload((rule = 1,));
        }
    }
}

machine AuthoritySessionCertStartOnlyMain {
    start state Init {
        entry {
            new AuthoritySessionWorkload((rule = 2,));
        }
    }
}

module AuthoritySessionModule = { AuthoritySessionMain, AuthoritySessionWorkload, AuthoritySessionMachine };
module AuthoritySessionBugModule = { AuthoritySessionBugMain, AuthoritySessionWorkload, AuthoritySessionMachine };
module AuthoritySessionCertStartOnlyModule = { AuthoritySessionCertStartOnlyMain, AuthoritySessionWorkload, AuthoritySessionMachine };

test AuthoritySessionRetirement [main = AuthoritySessionMain]:
    assert AuthoritySessionMonitor in AuthoritySessionModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it): proves AuthoritySessionMonitor can detect a retired key still signing.
test BrokenAuthoritySessionRetirement [main = AuthoritySessionBugMain]:
    assert AuthoritySessionMonitor in AuthoritySessionBugModule;

// Expected to FAIL. Models the implementation's CURRENT rule (certificate start
// only, rule 2) against the invariant above. It passes only while that rule
// cannot satisfy the invariant, which is the D1 gap. If the owner adds an
// authenticated issue time (D1), the implementation moves to rule 0 and this test
// stays valid as a statement that the certificate-start rule alone is insufficient.
test BrokenAuthoritySessionCertStartOnly [main = AuthoritySessionCertStartOnlyMain]:
    assert AuthoritySessionMonitor in AuthoritySessionCertStartOnlyModule;
