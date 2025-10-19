# KNIRV URI Handler Documentation

## Overview

The KNIRV application supports custom URI schemes to enable deep linking and integration with other applications. This document describes the implementation of URI handling in the KNIRV application.

## Supported URI Schemes

The application supports two custom URI schemes:

1. **`chain://...`**: For KNIRVCHAIN Verifiers
   - Identify specific verifier nodes
   - Access verifier APIs (e.g., for checking asset validity, querying metadata)
   - Establish peer-to-peer connections between verifiers
   - Manage node reputation or staking
   - Example: `chain://verifier1.example.com/status?nodeID=xyz123&region=US`

2. **`nrn://...`**: For privately shared NRN Assets
   - Identify and locate 3D assets
   - Specify asset metadata (author, version, license)
   - Facilitate access to the asset content
   - Example: `nrn://asset123?author=creator&version=2&license=MIT&network=mainnet&relay=relay.example.com`

## Implementation

The implementation is contained in the `uri_handler.go` file, which provides a complete solution for:

1. **Detecting Launch from URI**: The application detects when it's launched via a URI and extracts the URI from command-line arguments.
2. **Parsing URIs**: The application parses both `chain://` and `nrn://` URIs to extract relevant information.
3. **Handling Content**: Based on the URI type and content, the application performs appropriate actions.

### Key Components

- **`AssetMetadata` struct**: Stores metadata extracted from NRN URIs
- **`parseChainURI` function**: Parses chain:// URIs and extracts parameters
- **`parseNRNURI` function**: Parses nrn:// URIs and extracts asset metadata
- **`handleURI` function**: Determines the URI type and processes it accordingly
- **`main` function**: Entry point that checks for URI launch and handles normal startup

## Usage

When the application is launched with a URI (e.g., by clicking a link in a browser or another application), the operating system passes the URI as a command-line argument. The application detects this and processes the URI accordingly.

### Example URIs

- Chain URI: `chain://verifier1.example.com/status?nodeID=xyz123&region=US`
- NRN URI: `nrn://asset123?author=creator&version=2&license=MIT&network=mainnet&relay=relay.example.com`

## Important Considerations

### Security

- **URI Validation**: The implementation thoroughly validates the structure and contents of URIs to prevent malicious URIs from being used to attack the application.
- **Access Tokens**: If access tokens are included in URIs, they should be protected from interception. Consider using short-lived tokens and HTTPS.

### User Experience

- **Clear Error Messages**: The implementation provides clear error messages if a URI is invalid or if there are problems retrieving content.
- **Fallback**: If the application is not installed or cannot handle a URI, a fallback mechanism should be provided (e.g., redirect to a web page).

### Platform Differences

URI scheme registration is platform-specific. You'll need to handle the differences between Windows, macOS, and Linux:

- **Windows**: Registry entries
- **macOS**: Info.plist file
- **Linux**: Desktop entry files

### Updating Handlers

Make sure that after updating the handler of the URI, the system gets updated as well.

## Testing

The implementation includes a `testURIParsers` function that demonstrates the URI parsing functionality with example URIs. This function is called when the application is launched without a URI, for demonstration purposes.