

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/1_Private_DHT_Implementation.md

```markdown
## Implementation Plan: Private KNIRVCHAIN DHT

This plan outlines the steps to establish a private DHT for your KNIRVCHAIN network, isolating it from the public IPFS DHT.

**1. Establish Dedicated Bootstrap Nodes:**

*   **Objective:** Create a small set of stable, publicly accessible KNIRVCHAIN nodes that will serve as the entry points for your private network.
*   **Action:**
    *   Identify 2-3 servers/machines with static IP addresses (or reliable DNS) where you can run the KNIRVCHAIN node software continuously.
    *   Ensure the P2P port (e.g., 4001 or a custom one for bootstrap nodes) is open and accessible on these machines.
    *   Run the KNIRVCHAIN application on these machines. They can run in standard mode (`go run .`) or potentially a dedicated bootstrap mode if you add one later. They must *not* be client-only.
    *   Once running, record the full multiaddress of each bootstrap node. This looks like `/ip4/<IP_ADDRESS>/tcp/<P2P_PORT>/p2p/<PEER_ID>`. You can find this in the startup logs.

**2. Configure Custom Bootstrap Peers:**

*   **Objective:** Tell all KNIRVCHAIN nodes (root, reflection, devs) to use *only* your dedicated bootstrap nodes for initial DHT connection, instead of the public IPFS defaults.
*   **Action:**

    *   **(Option A - Hardcoding - Simpler Start):**
        *   Open `discovery_manager.go`.
        *   Create a new variable, perhaps `KNIRVCHAINBootstrapPeers`, and populate it with the multiaddresses recorded in Step 1.

        ```go
        // discovery_manager.go
        var KNIRVCHAINBootstrapPeers = []string{
            "/ip4/YOUR_BOOTSTRAP_IP_1/tcp/YOUR_P2P_PORT_1/p2p/YOUR_BOOTSTRAP_PEERID_1",
            "/ip4/YOUR_BOOTSTRAP_IP_2/tcp/YOUR_P2P_PORT_2/p2p/YOUR_BOOTSTRAP_PEERID_2",
            // Add more...
        }
        ```

    *   **(Option B - Configuration File - More Flexible):**
        *   Add a `bootstrap_devs` field (a string slice) to `config.Config` in `config/config.go` and to your `config.json`.
        *   Populate `config.json` with the bootstrap multiaddresses.
        *   Ensure `config.LoadConfig` reads this field.
        *   Pass this list from `main.go` into `NewDiscoveryManager`.

**3. Modify NewDiscoveryManager:**

*   **Objective:** Update the DHT creation logic to use the custom bootstrap dev list.
*   **Action:**
    *   In `discovery_manager.go`, locate the `dual.New` call within `NewDiscoveryManager`.
    *   Modify the `dht.BootstrapPeers` option to use your new list (either `KNIRVCHAINBootstrapPeers` or the list loaded from config) instead of `DefaultBootstrapPeers`. Make sure to use `convertToAddrInfo` to convert the strings.

        ```go
        // discovery_manager.go -> NewDiscoveryManager

        // Use the custom list here (replace DefaultBootstrapPeers)
        bootstrapPeersToUse := KNIRVCHAINBootstrapPeers // Or the list from config

        idht, err := dual.New(ctx, h,
            dual.DHTOption(dht.Mode(dht.ModeServer)),
            dual.DHTOption(dht.BootstrapPeers(convertToAddrInfo(bootstrapPeersToUse)...)), // Use your list
        )
        // ... rest of the function
        ```

**4. AutoRelay (If Used):**

*   If you are using `libp2p.EnableAutoRelay` (typically for non-client-only nodes), also update the `autorelay.WithPeerSource` function to provide your custom bootstrap devs as potential relay candidates, similar to how it uses `DefaultBootstrapPeers` now.

**5. Testing:**

*   **Objective:** Verify that nodes connect only within the private network and can still discover each other based on KNIRVCHAIN resource IDs.
*   **Action:**
    *   Start your dedicated bootstrap nodes.
    *   Start a root node (`go run . -network`) using the modified code. Check its logs to ensure it connects only to your bootstrap devs initially and successfully bootstraps the DHT.
    *   Run the dev installation (`go run . -dev`). Ensure it connects to the root node (which is part of the private DHT) for URI generation.
    *   After installation, restart the dev node (`go run . -dev`). Verify it connects to the bootstrap nodes and successfully announces its resource (`myPeer.chain` or similar) within the private DHT.
    *   **Modify `dev_lifecycle_test.go`:**
        *   Update the `startTestNodeProcess` helper or the test setup to ensure the test root node and the test dev node are also configured to use the custom `KNIRVCHAINBootstrapPeers` list (you might need to pass this via config or an environment variable during the test).
    *   Run the test. It should pass, confirming installation and DHT announcement within the private network.
    *   **(Optional)** Start a node using the old code (with public IPFS bootstrap devs). Verify it cannot find the KNIRVCHAIN root or dev nodes via DHT lookups for `agent-root-xxxx.chain` or `myPeer.chain`.

**6. Deployment:**

*   **Objective:** Roll out the private network configuration.
*   **Action:**
    *   Ensure the dedicated bootstrap nodes are running reliably.
    *   Deploy the updated KNIRVCHAIN application (root, reflection, dev nodes) configured to use the custom bootstrap dev list.

This plan establishes an isolated DHT overlay for KNIRVCHAIN, allowing your nodes to discover each other efficiently without relying on or interacting with the public IPFS DHT, while still using your `agent://` derived identifiers for resource discovery within your network.
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
