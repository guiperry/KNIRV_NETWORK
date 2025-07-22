KNIRVCHAIN SDK Testing Guide
This guide will help you test the KNIRVCHAIN SDKs that have been implemented in Go, Python, and JavaScript/TypeScript. Each SDK provides similar functionality but with language-specific patterns and features.

Prerequisites
Before testing the SDKs, ensure you have the following installed:

Go (version 1.16 or later)
Python (version 3.7 or later)
Node.js (version 16 or later)
npm or yarn (for JavaScript/TypeScript SDK)
Testing the Go SDK
Setup
Navigate to the Go SDK directory:

cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/go
Install dependencies:

go mod tidy
Running Tests
Run the parser tests:

cd parser
go test -v
Run the client tests:

cd ../client
go test -v
Running the Example
Build and run the example:
cd ../example
go build -o knirv-example
./knirv-example knirv://mychain-alpha.chain/block?number=123
Testing the Python SDK
Setup
Navigate to the Python SDK directory:

cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/python
Create a virtual environment:

python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
Install the SDK in development mode:

pip install -e .
Running Tests
Install pytest:

pip install pytest
Run the tests:

pytest
Running the Example
Run the example script:
python examples/fetch_resource.py knirv://mychain-alpha.chain/block?number=123
Testing the JavaScript/TypeScript SDK
Setup
Navigate to the JavaScript SDK directory:

cd /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP/sdk/js
Install dependencies:

npm install
# or
yarn install
Build the SDK:

npm run build
# or
yarn build
Running Tests
Run the tests:
npm test
# or
yarn test
Running the Example
Run the TypeScript example:
npx ts-node examples/fetch-resource.ts knirv://mychain-alpha.chain/block?number=123
Testing with Real KNIRVCHAIN Network
To test with a real KNIRVCHAIN network, you'll need:

Bootstrap Peers: Replace the default bootstrap peers in the examples with actual KNIRVCHAIN network peers.

Valid URIs: Use valid knirv:// URIs that point to resources available on the network.

Example Configuration
For all SDKs, update the bootstrap peers to point to your KNIRVCHAIN network nodes:

// Go
config := client.ClientConfig{
    BootstrapPeers: []string{
        "/ip4/your-node-ip/tcp/port/p2p/peer-id",
        // Add more peers here
    },
    LogEnabled: true,
}

# Python
client = KnirvClient(
    bootstrap_peers=[
        "/ip4/your-node-ip/tcp/port/p2p/peer-id",
        # Add more peers here
    ],
    log_enabled=True
)

// JavaScript/TypeScript
const client = new KnirvClient({
    bootstrapPeers: [
        '/ip4/your-node-ip/tcp/port/p2p/peer-id',
        // Add more peers here
    ],
    logEnabled: true
});
Troubleshooting
Common Issues
Connection Failures: If you can't connect to bootstrap peers, check:

Network connectivity
Firewall settings
Correct peer multiaddresses
Resource Not Found: If resources can't be found, verify:

The URI is correctly formatted
The resource exists on the network
The provider for that resource is online
Build Errors:

Go: Ensure all dependencies are available (go mod tidy)
Python: Check virtual environment is activated
JavaScript: Verify all npm/yarn dependencies are installed
Debugging
All SDKs include a logEnabled option that provides detailed logging:

// Go
config.LogEnabled = true

# Python
client = KnirvClient(log_enabled=True)

// JavaScript/TypeScript
const client = new KnirvClient({ logEnabled: true });
Next Steps
After successfully testing the SDKs, you can:

Integrate with Your Application: Import the SDK into your application and use it to interact with the KNIRVCHAIN network.

Extend the SDK: Add additional functionality specific to your use case.

Contribute Improvements: If you find bugs or have enhancements, consider contributing them back to the project.

Support
If you encounter issues or have questions, please:

Check the documentation in each SDK's README file
Look for error messages in the logs (enable logging as shown above)
Contact the KNIRVCHAIN development team for assistance
Happy testing!