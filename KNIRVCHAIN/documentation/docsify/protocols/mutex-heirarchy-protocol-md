# AGENTCHAIN Mutex Hierarchy Protocol

## 1. Introduction

This document defines the mutex locking hierarchy for the AGENTCHAIN application. The purpose of this protocol is to prevent deadlocks by establishing a strict order in which mutexes must be acquired when multiple locks are needed by a single goroutine. Adherence to this protocol is critical for the stability and correctness of the concurrent operations within the system.

**General Rule:** If a goroutine needs to acquire multiple mutexes, it must acquire them in the order specified in this document (from lower numerical order/higher level to higher numerical order/lower level). A goroutine holding a lock `L_i` may acquire lock `L_j` only if `i < j` according to the defined hierarchy. A goroutine must never attempt to acquire a lock `L_k` if it already holds a lock `L_m` where `k < m`.

## 2. Mutex Definitions and Hierarchy

The following mutexes are identified within the AGENTCHAIN system. They are listed in the order they should be acquired (i.e., you can acquire `M1` then `M2`, but not `M2` then `M1`).

---

### Level 1: Global Application State / Configuration Mutexes

*   **`config.ConfigMutex` (Hypothetical - if config can be dynamically reloaded/written)**
    *   **File:** `config/config.go` (if implemented)
    *   **Purpose:** Protects access to the global application configuration if it's mutable at runtime.
    *   **Scope:** Global application configuration.
    *   **Notes:** Currently, your config is loaded at startup. If it becomes dynamically updatable, this lock would be paramount.

*   **`server_utils.portMutex`**
    *   **File:** `server_utils.go`
    *   **Purpose:** Protects global `serverPort` variable.
    *   **Scope:** Global server port state.

---

### Level 2: Core Service Manager Mutexes

These mutexes protect the state of major, independent services.

*   **`DiscoveryManager.mu` / `DiscoveryManager.closeMu`**
    *   **File:** `discovery_manager.go`
    *   **Purpose:** `mu` protects internal state like `clients`, `bootstrapped`. `closeMu` (with `closeOnce`) protects the `Close()` operation.
    *   **Scope:** State of the `DiscoveryManager` instance.
    *   **Notes:** `closeOnce` ensures `Close()` logic runs only once. `closeMu` protects the `closed` flag if used.

*   **`ConsensusManager.mu` (Legacy HTTP-based)**
    *   **File:** `self_consensus_manager.go`
    *   **Purpose:** Protects `LongestChain`, `UpdateRequired`, `MiningLocked`, `stop`, `reflectionURLs`.
    *   **Scope:** State of the `ConsensusManager` instance.

*   **`P2PConsensusManager.mu`**
    *   **File:** `p2p_consensus.go`
    *   **Purpose:** Protects `miningLocked`, `isSyncing`.
    *   **Scope:** State of the `P2PConsensusManager` instance.

*   **`ReflectionManager.mu`**
    *   **File:** `reflection_manager.go`
    *   **Purpose:** Protects the `reflections` slice.
    *   **Scope:** Global list of reflection nodes.
    *   **Notes:** `reflectionManagerOnce` ensures singleton instantiation.

---

### Level 3: Blockchain Structure Mutex

This is a critical, central lock.

*   **`BlockchainStruct.mu`**
    *   **File:** `blockchain_struct.go`
    *   **Purpose:** Protects the core blockchain data: `TransactionPool`, `Blocks`, `ChainAddress`, `Reflections` (within `BlockchainStruct`), `MiningLocked` (within `BlockchainStruct`), `isActivelyMining`, `ChainID`, `Accounts` (implicitly, as balance checks and updates happen under this lock).
    *   **Scope:** The entire state of a `BlockchainStruct` instance.
    *   **Criticality:** This lock is frequently acquired. Operations holding this lock should be as short as possible. Avoid calling functions that might acquire Level 1 or Level 2 locks while holding this lock.

---

### Level 4: GUI State Mutex

*   **`guiState.mu`**
    *   **File:** `gui.go`
    *   **Purpose:** Protects non-data-bound fields or complex updates to lists within the `guiState` that are not inherently thread-safe through Fyne's data binding alone (e.g., updating `reflectionsList` from the `blockchain.Reflections` map).
    *   **Scope:** Internal state of the `guiState` object.
    *   **Notes:** Fyne's data binding mechanisms are generally thread-safe for individual bound items. This mutex is for more complex, multi-part updates or access to shared slices/maps within `guiState` that are populated from other goroutines.

---

### Level 5: Database Instance Mutex (Implicit)

*   **`LevelDB.Client` (Internal Mutexes within `goleveldb`)**
    *   **File:** `leveldb.go`, `leveldb_mcp.go`
    *   **Purpose:** The `goleveldb` library handles its own internal locking to ensure thread-safe access to the database files.
    *   **Scope:** Operations on a specific LevelDB instance.
    *   **Notes:** You do not explicitly acquire/release these. However, database operations can be blocking. Avoid holding higher-level locks (Levels 1-4) for extended periods if a database operation within that critical section might be slow. Prefer releasing higher-level locks before long DB operations if possible, or ensure DB operations are quick.

---

### Level 6: Wallet Object Mutex (Hypothetical - if Wallet becomes concurrent)

*   **`Wallet.mu` (Hypothetical)**
    *   **File:** `wallet.go`
    *   **Purpose:** If a single `Wallet` object instance were to be used concurrently (e.g., for signing multiple transactions in parallel by different goroutines using the *same* wallet instance), its internal state (especially if it involved counters or non-atomic operations related to the private key) would need a mutex.
    *   **Scope:** A single `Wallet` instance.
    *   **Notes:** Currently, `Wallet` methods like `SignTransaction` or `GetSignedTxn` appear to operate on the wallet's data without explicit internal locking, implying they are intended to be used by one goroutine at a time for a given wallet instance, or the `ecdsa.SignASN1` is inherently safe for concurrent use with the same key (which it generally is, as it doesn't modify the key). If a `Wallet` instance's state *could* be modified by signing (e.g., a nonce counter if you were implementing that within the wallet), a lock would be needed. For now, this is considered hypothetical unless specific concurrent use cases for a single wallet object arise.

---

## 3. Locking Order Rules and Examples

1.  **Acquire locks in ascending order of their Level.**
    *   **Correct:** Acquire Level 2 lock, then Level 3 lock.
    *   **Incorrect (Potential Deadlock):** Acquire Level 3 lock, then attempt to acquire Level 2 lock.

2.  **Within the same Level, if an explicit order is needed, it should be documented.**
    *   For Level 2, there's no obvious required order between `DiscoveryManager.mu`, `ConsensusManager.mu`, and `P2PConsensusManager.mu` *if they are independent*. However, if one manager's method calls another's, the calling order dictates the lock acquisition order.
    *   **Example:** If `P2PConsensusManager` needs to call a method on `DiscoveryManager` that acquires `DiscoveryManager.mu`, then `P2PConsensusManager.mu` should be acquired *before* `DiscoveryManager.mu` if both are needed by the same goroutine.

3.  **Release locks in the reverse order of acquisition.** (Standard practice).

4.  **Minimize Lock Hold Time:** Hold locks for the shortest possible duration, especially higher-level locks like `BlockchainStruct.mu`.
    *   Perform non-critical computations or I/O operations *before* acquiring the lock or *after* releasing it.
    *   **Example (from `BlockchainStruct.AddBlock`):**
        ```go
        // bc.Lock() // Acquire lock
        // ...
        // bcDataToSave, err := json.Marshal(bc) // Marshal data WHILE locked
        // ...
        // bc.Unlock() // Release lock BEFORE database I/O
        // err = db.PutBytes(chainAddr, bcDataToSave)
        ```
        Your `AddBlock` was refactored to release the lock before DB write, which is good. Ensure this pattern is followed.

5.  **Avoid Calling External/Unknown Code While Holding Locks:** This includes network calls, complex library functions that might have their own locking or blocking behavior, or callbacks into user-provided plugin code.

## 4. Specific Scenarios and Lock Interactions

*   **Transaction Processing (`BlockchainStruct.AddTransactionToTransactionPool`):**
    1.  Acquires `BlockchainStruct.mu` (Level 3) for pool and block checks.
    2.  Releases `BlockchainStruct.mu`.
    3.  Performs `transaction.VerifyTxn()` (which involves crypto, but no AGENTCHAIN mutexes).
    4.  Performs `bc.validateMCPTransaction()` (reads blockchain state, might need `BlockchainStruct.mu` if not already re-acquired carefully, or if it queries DB directly).
    5.  Performs `bc.simulatedBalanceCheck()` (acquires `BlockchainStruct.mu`).
    6.  Calls `bc.addVerifiedTxnToPoolAndSignal()` (acquires `BlockchainStruct.mu`).
    7.  Calls `bc.BroadcastTransaction()`:
        *   If `bc.p2pConsensusMgr` is used, `BroadcastTransaction` on `P2PConsensusManager` might acquire `P2PConsensusManager.mu` (Level 2). This is a **potential hierarchy violation** if `BlockchainStruct.mu` (Level 3) is still held.
        *   **Correction:** `BroadcastTransaction` should be called *after* releasing `BlockchainStruct.mu` or the `P2PConsensusManager` methods should not acquire their own lock if called from a context where `BlockchainStruct.mu` is already held (less ideal). The safest is to release the higher-level lock.
        *   Your current `AddTransactionToTransactionPool` releases the lock before verification and then re-acquires it for `addVerifiedTxnToPoolAndSignal`. The broadcast happens *after* this, which is good if the lock is released before broadcasting.

*   **Block Mining / Proposal (e.g., `BlockchainStruct.ProofOfWorkMining` or new PoAu-D `ProposeBlock`):**
    1.  Acquires `ConsensusManager.mu` (Level 2) via `cons.getMiningLockState()`.
    2.  Acquires `BlockchainStruct.mu` (Level 3) to get `TransactionPool`.
    3.  Releases `BlockchainStruct.mu`.
    4.  (PoW: Iterates nonces. PoAu-D: Prepares block).
    5.  Calls `bc.AddBlock()`:
        *   `AddBlock` acquires `BlockchainStruct.mu` (Level 3).
        *   `AddBlock` calls `bc.processMCPTransactions()` which calls `bc.db.SaveCapability()` etc. (Level 5 - DB internal locks). This is fine.
        *   `AddBlock` calls `bc.BroadcastBlock()`:
            *   Similar to transaction broadcasting, ensure `BlockchainStruct.mu` is released before `P2PConsensusManager.BroadcastBlock()` is called if it acquires `P2PConsensusManager.mu`.

*   **P2P Consensus (`P2PConsensusManager` methods):**
    *   `handleBlocks`, `handleTransactions`: These receive messages from the network. When they call `processReceivedBlock` or `processReceivedTransaction`:
        *   `processReceivedBlock` calls `pcm.lockMining()` (acquires `P2PConsensusManager.mu` - Level 2).
        *   Then calls `pcm.blockchain.AddBlock()` (which acquires `BlockchainStruct.mu` - Level 3). This order is correct (Level 2 then Level 3).
    *   `requestChainFromPeers`:
        *   Acquires `P2PConsensusManager.mu` (Level 2) for `isSyncing`.
        *   Calls `pcm.discoveryManager.FindResource()` (which might acquire `DiscoveryManager.mu` - Level 2). If these are independent, it's okay. If `FindResource` could somehow call back into `P2PConsensusManager` and try to acquire `P2PConsensusManager.mu`, that would be a problem (re-entrancy). Assume `DiscoveryManager` is independent here.
        *   Opens libp2p streams (network I/O).
        *   Calls `pcm.switchToChain()`:
            *   `switchToChain` calls `pcm.lockMining()` (acquires `P2PConsensusManager.mu` - Level 2).
            *   Then calls `pcm.blockchain.SetBlocks()` (which acquires `BlockchainStruct.mu` - Level 3). Correct order.
            *   Then calls `pcm.db.PutIntoDb()` (Level 5). Correct order.
            *   Then calls `pcm.updateTransactionPool()` (which acquires `BlockchainStruct.mu` - Level 3). Correct order.

*   **GUI Updates (`guiState.refreshData`):**
    *   Acquires `g.state.blockchain.Lock()` (`BlockchainStruct.mu` - Level 3).
    *   Accesses `g.state.discoveryMgr.host.Network().Peers()` (libp2p internal locking, likely fine).
    *   Calls `g.state.UpdateReflectionsList()` which acquires `g.state.mu` (`guiState.mu` - Level 4). This is correct (Level 3 then Level 4).

## 5. Review and Maintenance

*   This document should be reviewed whenever new mutexes are introduced or when existing concurrent workflows are significantly modified.
*   Code reviews should explicitly check for adherence to this locking hierarchy.
*   Use Go's race detector (`go test -race`) regularly.
*   For complex interactions, consider using `go tool pprof` to analyze goroutine states during testing if deadlocks are suspected.

## 6. Conclusion

By strictly following this Mutex Hierarchy Protocol, AGENTCHAIN aims to minimize the risk of deadlocks and ensure robust concurrent operation. Developers must be diligent in understanding and applying these rules.

*   **Key Takeaways and How to Use This Document:**

*   *Identify Your Locks:* Make sure all significant mutexes are listed.
*   *Establish Order:* The most crucial part is defining the order. The levels I've proposed are a starting point. You might need to refine them or add sub-levels.
*   *Document Rationale:* For complex ordering decisions, briefly note why that order was chosen.
*   *Code Reviews:* Make checking for adherence to this protocol a standard part of your code review process.
*   *Short Lock Durations:* Emphasize keeping critical sections (code executed while a lock is held) as short and fast as possible.
*   *Avoid Calls "Up" the Hierarchy:* A function holding a lock at Level N should generally not call another function that will try to acquire a lock at Level M where M < N.
*   *Database Operations:* While LevelDB handles its own locking, be mindful that DB operations can be I/O bound and potentially slow. Try to release higher-level application locks before performing extensive DB writes if the data needed for the write has already been prepared. Your BlockchainStruct.AddBlock has a good pattern for this by marshaling data while locked, then unlocking before the db.PutBytes call.

This document is a living one. As you implement PoAu-D and other features, you'll likely discover more subtle interactions and may need to update the hierarchy or add specific rules for new components like the Transaction Delegator. The Transaction Delegator Logic (TDL) especially will need careful consideration regarding its interaction with BlockchainStruct.mu and potentially P2PConsensusManager.mu.