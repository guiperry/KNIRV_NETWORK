# Implementing the Updater Protocol

Given the decentralized nature of KNIRVROOT, its reliance on a private DHT, and the `agent://` URI scheme, a custom update mechanism leveraging the DHT combined with a library like `go-update` for applying the binary patch would be the most synergistic and robust approach.

**Hybrid Approach: DHT for Discovery/Fetch + `go-update` for Application**

This approach gives us the best of both worlds: decentralized discovery and robust binary patching.

**1. Update Manifest (The "What and Where"):**

*   Create a cryptographically signed JSON or YAML file (the "update manifest").
*   **Contents:**
    *   `version`: The new version string (e.g., `"v1.2.3"`).
    *   `releaseNotesURL` (optional): A `agent://` URI or HTTP URL to detailed release notes.
    *   `timestamp`: Release timestamp.
    *   `artifacts`: An array or map, where each entry describes an update package for a specific platform/architecture (e.g., `linux/amd64`, `windows/amd64`).
        *   `platform`: e.g., `"linux/amd64"`
        *   `url`: A `agent://<contentID>.content/<binary_name_version.tar.gz>` URI pointing to the actual update package (e.g., a compressed binary). This package would be announced and findable on your DHT like any other NRN resource.
        *   `hash`: SHA256 hash of the update package file.
        *   `signature`: Cryptographic signature of the hash (or the entire artifact entry) by a trusted update signing key.
*   The entire manifest itself must also be signed by a master update signing key.

**2. Update Announcement (Publishing to the DHT):**

*   The entity responsible for releases (e.g., KNIRVROOT Root) would:
    *   Build the new version binaries.
    *   Package them (e.g., `tar.gz`).
    *   Calculate their hashes.
    *   Host these packages on one or more stable KNIRVROOT Bootnodes (or even IPFS, then reference the IPFS CID via a `agent://<ipfs_cid>.chain/update` URI). These hosting Bootnode would announce the `agent://` URIs of the packages on the DHT.
    *   Create the update manifest, sign the artifact entries, and sign the manifest itself.
    *   Announce the signed update manifest on the DHT. This could be done by:
        *   Storing the manifest under a well-known, fixed `agent://` URI, e.g., `agent://KNIRVROOT-updates.chain/latest-manifest`. Nodes would resolve this URI to find devs serving the latest manifest.
        *   Or, announcing the manifest's content hash (CID) on the DHT under a well-known key, and nodes query for providers of that manifest CID.

**3. Update Discovery (Client Node Logic):**

*   Each KNIRVROOT node would periodically (e.g., on startup and every few hours):
    *   Attempt to resolve the well-known URI for the latest update manifest (e.g., `agent://KNIRVROOT-updates.chain/latest-manifest`) using its `DiscoveryManager`.
    *   Download the manifest from a discovered dev.
    *   Verify the signature of the manifest itself using a pre-configured trusted public key (the master update public key).
    *   If the manifest's version is newer than its current running version, proceed to fetch the update.

**4. Update Fetching & Verification (Client Node Logic):**

*   From the verified manifest, select the artifact entry matching the node's platform/architecture.
*   Resolve the `agent://...` URL for the update package using the `DiscoveryManager` to find devs hosting it.
*   Download the update package (e.g., `binary_v1.2.3.tar.gz`).
*   Verify the downloaded package's SHA256 hash against the hash specified in the manifest.
*   Verify the signature of the artifact (from the manifest) using the trusted update signing public key.

**5. Applying the Update (Client Node Logic - Using `go-update`):**

*   If all verifications pass, use the `go-update` library (or a similar one like `github.com/sanbornm/go-selfupdate` which is a fork of `go-update` with some enhancements).
*   The `go-update` library can take an `io.Reader` (your downloaded and verified package, possibly after decompression) and apply it to the current executable. It handles:
    *   Permissions.
    *   Replacing the current binary safely (e.g., writing to a temporary file, then renaming).
    *   Checksum verification (though you've already done a more robust hash and signature check).
*   After the update is applied, the node would need to restart itself to run the new version. This restart should be handled gracefully.

```go
// Conceptual Go snippet for applying update using go-update
// Assume 'updatePackageStream' is an io.Reader for the downloaded & verified binary
// and 'targetExecutablePath' is the path to the current running executable.

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/inconshreveable/go-update" // Or a fork like sanbornm/go-selfupdate
)

func applyUpdate(updatePackageStream io.Reader, targetExecutablePath string) error {
	err := update.Apply(updatePackageStream, update.Options{
		TargetPath: targetExecutablePath,
		// You can also provide checksums here if go-update supports your hash type,
		// but you've already verified it more strongly with signatures.
	})
	if err != nil {
		if rerr := update.RollbackError(err); rerr != nil {
			log.Printf("Failed to rollback from failed update: %v", rerr)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}
	log.Printf("Update applied successfully. Please restart the application.")
	// Trigger graceful shutdown and restart logic here
	return nil
}
```

**Security is Paramount:**

*   **Signing Keys:** The master update manifest signing key and the artifact signing key(s) are extremely sensitive. They must be rigorously protected. Consider HSMs for production.
*   **Trusted Public Keys:** Nodes must have the corresponding public keys pre-configured (e.g., embedded in the initial software distribution) to verify signatures. Updating these trusted public keys themselves is a very high-security operation, possibly requiring manual intervention or a multi-signature governance process.
*   **HTTPS for Initial Manifest (Optional):** While the DHT provides decentralization, for the very first discovery of the "latest manifest URI" or the master public key, a highly secured HTTPS endpoint could serve as an initial pointer, with subsequent updates and verifications happening via the DHT.
*   **Rollback Plan:** Ensure your update process allows for easy rollback if a new version introduces critical bugs. `go-update` has some rollback capabilities if an update fails mid-process.

**Why this is better for KNIRVROOT:**

*   **Maintains Decentralization:** Update discovery and fetching are not reliant on a single server.
*   **Uses Your Strengths:** Leverages your existing DHT and `agent://` URI infrastructure.
*   **Robust Application:** `go-update` handles the tricky OS-level details of replacing a running executable.

This approach requires more setup than just pointing `go-update` at an HTTP URL, but it aligns much better with the decentralized and secure ethos of a blockchain project.
```