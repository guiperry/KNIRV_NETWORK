# KNIRV Wallet Browser Extension

A comprehensive browser extension for the KNIRV Network ecosystem, providing secure wallet management, transaction signing, and dApp integration directly in your browser.

## Overview

The KNIRV Wallet Extension is a browser-based wallet that enables users to:
- Manage KNIRV Network (NRN) tokens and other supported cryptocurrencies
- Interact with decentralized applications (dApps) on the KNIRV Network
- Sign transactions securely without exposing private keys
- Sync with the native KNIRV Wallet mobile app via QR codes
- Access DeFi protocols and services on the KNIRV ecosystem

## Features

### 🔐 Security
- **Hardware Wallet Support**: Integration with Ledger devices
- **Multi-Keyring Architecture**: Support for HD wallets, private keys, and Web3Auth
- **Secure Storage**: Encrypted private key storage in browser extension storage
- **Transaction Signing**: Local transaction signing without key exposure
- **Permission Management**: Granular permissions for dApp connections

### 💼 Wallet Management
- **Multiple Wallet Types**:
  - HD (Hierarchical Deterministic) wallets with BIP39 mnemonic support
  - Private key import wallets
  - Ledger hardware wallets
  - Web3Auth social login wallets
  - Address-only (watch-only) wallets

### 🌐 dApp Integration
- **Web3 Provider**: Standard Web3 provider interface for dApp compatibility
- **Transaction Requests**: Handle and approve transaction requests from dApps
- **Message Signing**: Sign arbitrary messages for dApp authentication
- **Network Management**: Automatic network detection and switching

### 📱 Cross-Platform Sync
- **QR Code Connectivity**: Connect and sync with KNIRV Wallet mobile app
- **Real-time Synchronization**: Live sync of wallet data and transactions
- **Multi-Device Support**: Access your wallets across multiple devices

## Installation

### From Chrome Web Store
1. Visit the Chrome Web Store (link coming soon)
2. Click "Add to Chrome"
3. Follow the installation prompts

### Development Installation
```bash
# Clone the repository
git clone https://github.com/your-org/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVWALLET/browser-wallet/packages/knirvwallet-extension

# Install dependencies
npm install

# Build the extension
npm run build

# Load in Chrome
# 1. Open Chrome and go to chrome://extensions/
# 2. Enable "Developer mode"
# 3. Click "Load unpacked" and select the dist/ folder
```

## Usage

### Initial Setup
1. **Install Extension**: Add the extension to your browser
2. **Create Wallet**: Choose from:
   - Generate new HD wallet with mnemonic
   - Import existing wallet with mnemonic or private key
   - Connect Ledger hardware wallet
   - Sign in with Web3Auth (Google, Facebook, etc.)

3. **Secure Your Wallet**: Set a strong password for encryption

### Wallet Operations

#### Creating a New Wallet
```javascript
// The extension automatically handles wallet creation through UI
// No direct API calls needed for basic users
```

#### Connecting to dApps
```javascript
// dApps can request connection
if (window.knirv) {
  const accounts = await window.knirv.request({
    method: 'knirv_requestAccounts'
  });
  console.log('Connected accounts:', accounts);
}
```

#### Sending Transactions
```javascript
// dApps can request transactions
const txHash = await window.knirv.request({
  method: 'knirv_sendTransaction',
  params: [{
    from: '0x...',
    to: '0x...',
    value: '1000000', // Amount in smallest unit
    data: '0x...'     // Optional transaction data
  }]
});
```

### QR Code Sync with Mobile App

#### Connecting Wallets
1. Open KNIRV Wallet mobile app
2. Go to "Connect Browser Wallet"
3. In browser extension, click "Connect Mobile App"
4. Scan the QR code displayed in the extension
5. Approve the connection on mobile

#### Syncing Data
- **Automatic Sync**: Wallets sync automatically when connected
- **Manual Sync**: Use the sync button in extension settings
- **Real-time Updates**: Transaction updates appear instantly across devices

## API Reference

### Window Object Integration
The extension injects a `knirv` object into web pages:

```javascript
window.knirv = {
  // Check if KNIRV Wallet is installed
  isKnirvWallet: true,
  
  // Request method for all operations
  request: async (args) => { /* ... */ },
  
  // Event listeners
  on: (event, callback) => { /* ... */ },
  removeListener: (event, callback) => { /* ... */ }
}
```

### Supported Methods

#### Account Management
- `knirv_requestAccounts`: Request access to user accounts
- `knirv_accounts`: Get currently connected accounts
- `knirv_getBalance`: Get account balance

#### Transaction Methods
- `knirv_sendTransaction`: Send a transaction
- `knirv_signTransaction`: Sign a transaction without sending
- `knirv_signMessage`: Sign an arbitrary message

#### Network Methods
- `knirv_chainId`: Get current chain ID
- `knirv_switchChain`: Request chain switch

### Events
- `accountsChanged`: Fired when active account changes
- `chainChanged`: Fired when active chain changes
- `connect`: Fired when wallet connects to dApp
- `disconnect`: Fired when wallet disconnects from dApp

## Configuration

### Network Settings
```json
{
  "networks": {
    "knirv-mainnet": {
      "chainId": "knirv-1",
      "rpcUrl": "https://rpc.knirv.network",
      "name": "KNIRV Network"
    },
    "knirv-testnet": {
      "chainId": "knirv-testnet-1",
      "rpcUrl": "https://testnet-rpc.knirv.network",
      "name": "KNIRV Testnet"
    }
  }
}
```

### Security Settings
- **Auto-lock Timer**: Automatically lock wallet after inactivity
- **Transaction Confirmations**: Require confirmations for large transactions
- **dApp Permissions**: Manage which dApps can access your wallet

## Development

### Building from Source
```bash
# Install dependencies
npm install

# Development build with hot reload
npm run dev

# Production build
npm run build

# Run tests
npm test

# Lint code
npm run lint
```

### Project Structure
```
src/
├── background/          # Background script
├── content/            # Content scripts
├── popup/              # Extension popup UI
├── options/            # Options page
├── inject/             # Injected scripts
└── shared/             # Shared utilities
```

## Security Considerations

### Best Practices
- **Never share your mnemonic**: Keep your recovery phrase secure
- **Verify transactions**: Always review transaction details before signing
- **Use hardware wallets**: For large amounts, consider Ledger integration
- **Regular backups**: Export and securely store wallet backups

### Security Features
- **Encrypted Storage**: All sensitive data is encrypted
- **Secure Communication**: All network requests use HTTPS
- **Permission System**: Granular control over dApp access
- **Auto-lock**: Automatic wallet locking for security

## Troubleshooting

### Common Issues

#### Extension Not Loading
- Check if developer mode is enabled
- Verify all files are present in the extension directory
- Check browser console for errors

#### Connection Issues
- Ensure you're on a supported network
- Check if the dApp is compatible with KNIRV Wallet
- Try refreshing the page and reconnecting

#### Sync Problems
- Verify both devices are connected to the internet
- Check if QR code scanning permissions are granted
- Ensure both apps are updated to the latest version

### Support
- **Documentation**: [docs.knirv.network](https://docs.knirv.network)
- **Community**: [Discord](https://discord.gg/knirv)
- **Issues**: [GitHub Issues](https://github.com/your-org/KNIRV_NETWORK/issues)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

---

**Note**: This extension is part of the KNIRV Network ecosystem. For the complete wallet experience, also check out the [KNIRV Wallet mobile app](../agentic-wallet/README.md).
