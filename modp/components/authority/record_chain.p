// record_chain.p - per-operator authority record chain (AUTHORITY_SPEC.md).
// Every accepted record names the operator's current chain head as its
// prev_record_hash, so one operator can never have two accepted records that
// share a prev_record_hash (no fork) and cannot skip a link (no gap).
//
// Hashes are small ints; 0 is the genesis head.

event eAuthChainAppend: (op: int, recordHash: int, prevHash: int);
event eAuthChainAccepted: (op: int, recordHash: int, prevHash: int);
event eAuthChainRejected: (op: int, recordHash: int, prevHash: int);

machine AuthorityChainMachine {
    var heads: map[int, int];
    var allowFork: bool;

    start state Init {
        // allowFork=true is the deliberately broken variant: it skips the
        // chain-head check. Only the negative test constructs it.
        entry (cfg: (allowFork: bool)) {
            heads = default(map[int, int]);
            allowFork = cfg.allowFork;
            goto Active;
        }
    }

    state Active {
        on eAuthChainAppend do (r: (op: int, recordHash: int, prevHash: int)) {
            var head: int;
            head = 0;
            if (r.op in heads) {
                head = heads[r.op];
            }
            if (r.prevHash != head && !allowFork) {
                announce eAuthChainRejected, r;
                return;
            }
            heads[r.op] = r.recordHash;
            announce eAuthChainAccepted, r;
        }
    }
}

// The monitor does not trust the machine: it rebuilds each operator's head from
// accepted records alone.
spec AuthorityChainMonitor observes eAuthChainAccepted {
    var heads: map[int, int];

    start state Monitoring {
        entry {
            heads = default(map[int, int]);
        }

        on eAuthChainAccepted do (r: (op: int, recordHash: int, prevHash: int)) {
            var head: int;
            head = 0;
            if (r.op in heads) {
                head = heads[r.op];
            }
            assert r.prevHash == head,
                "an accepted record must extend the operator's chain head (no fork, no gap)";
            heads[r.op] = r.recordHash;
        }
    }
}

machine AuthorityChainWorkload {
    var chain: machine;
    var i: int;
    var nextHash: int;
    var who: int;
    var prev: int;

    start state Init {
        entry (cfg: (allowFork: bool)) {
            chain = new AuthorityChainMachine((allowFork = cfg.allowFork,));
            nextHash = 1;
            i = 0;
            while (i < 6) {
                who = choose(2);
                prev = choose(nextHash);
                send chain, eAuthChainAppend, (op = who, recordHash = nextHash, prevHash = prev);
                nextHash = nextHash + 1;
                i = i + 1;
            }
        }
    }
}

machine AuthorityChainMain {
    start state Init {
        entry {
            new AuthorityChainWorkload((allowFork = false,));
        }
    }
}

machine AuthorityChainForkBugMain {
    start state Init {
        entry {
            new AuthorityChainWorkload((allowFork = true,));
        }
    }
}

module AuthorityChainModule = { AuthorityChainMain, AuthorityChainWorkload, AuthorityChainMachine };
module AuthorityChainForkBugModule = { AuthorityChainForkBugMain, AuthorityChainWorkload, AuthorityChainMachine };

test AuthorityRecordChain [main = AuthorityChainMain]:
    assert AuthorityChainMonitor in AuthorityChainModule;

// Expected to FAIL (named Broken* so a prefix match on the real test never runs it): proves AuthorityChainMonitor can detect a fork.
test BrokenAuthorityRecordChainFork [main = AuthorityChainForkBugMain]:
    assert AuthorityChainMonitor in AuthorityChainForkBugModule;
