**Introduction**

Welcome to the KNIRV SDK User Guide. This guide provides a comprehensive overview of the KNIRV SDK, including installation, usage, and troubleshooting. The KNIRV SDK is a set of tools for building applications on the KNIRV Network.

**Available SDKs**

The KNIRV SDK offers three types of SDKs:

* **Gateway SDKs:** These provide complete integration with KNIRVGATEWAY, offering access to economic services, API gateway functionality, and PoAu-D consensus management.
* **Core KNIRV SDKs:** These provide fundamental network functionality, such as transaction management and network communication.
* **Unified SDKs (In Development):** These will combine access to all KNIRV services into a single SDK.

### Gateway SDKs

The Gateway SDKs provide complete integration with KNIRVGATEWAY, offering access to economic services, API gateway functionality, and PoAu-D consensus management.

#### Key Features

* **Economics Service:** Skill invocation, LLM registration, fee calculation, metrics, and transaction management.
* **API Gateway:** Service routing, health monitoring, and status management.
* **Integration Management:** Component connectivity, cross-service communication, and real-time monitoring.
* **PoAu-D Consensus Management:** Control over the PoAu-D consensus mechanism, management of Network Author Peers (NAPs), and transaction delegation.

### Core KNIRV SDKs

The Core KNIRV SDKs provide fundamental network functionality, such as transaction management and network communication.

#### Key Features

* Transaction Management
* Network Communication
* URI Resolution (`knirv://` URIs)
* Peer Discovery
* Resource Fetching
* Error Handling

### Unified SDKs (In Development)

The Unified SDKs will combine access to all KNIRV services into a single SDK.

### KNIRV URI Structure

KNIRV URIs follow this structure: `knirv://<ID>.<ResourceType>/<OptionalSubPath>?param1=value1&param2=value2`

* **Scheme:** `knirv`
* **Authority:** `<ID>.<ResourceType>` (e.g., `mychain.chain`, `content123.nrn`)
* **Path:** `/<OptionalSubPath>` (e.g., `/block`, `/content`)
* **Query:** `?param1=value1...`

### Installation and Setup

Installation instructions vary depending on the SDK and programming language. Please refer to the individual SDK READMEs linked above for detailed instructions.

### Quick Start Examples

The following examples demonstrate basic usage of the Gateway and Core SDKs.

#### Gateway SDK (Go)

```go
import "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
// ... (rest of the Go example from the original README)
```

#### Gateway SDK (TypeScript)

```typescript
import { KNIRVGatewayClient } from '@knirv/gateway-sdk';
// ... (rest of the TypeScript example from the original README)
```

#### PoAu-D Consensus Management (Go)

```go
package main
import (
    "context"
    "fmt"
    "log"
    "github.com/cloud-equities/KNIRVGATEWAY/sdk/go/gateway"
)
// ... (rest of the Go PoAu-D example from the original README)
```

#### PoAu-D Consensus Management (TypeScript)

```typescript
import { PoAuDClient } from '@knirv/gateway-sdk';
// ... (rest of the TypeScript PoAu-D example from the original README)
```

### Troubleshooting

* **Connection Issues:** Ensure your network is configured correctly and that you have the necessary permissions. Check the KNIRV Network status for outages.
* **Error Handling:**  Each SDK provides robust error handling.  Examine error messages carefully for clues.
* **Version Compatibility:** Ensure you are using compatible versions of the SDKs and the KNIRV Network.

### Further Information

For more detailed information, refer to the individual SDK READMEs and the documentation linked within the original README.

Improvements Needed:

* Add more detailed examples for each SDK.
* Provide a more comprehensive troubleshooting guide.
* Consider adding a FAQ section.
* Update the content to reflect the latest changes in the KNIRV Network and SDKs.
* Consider adding a section on best practices for using the KNIRV SDK.
* Improve the overall organization and structure of the guide.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
