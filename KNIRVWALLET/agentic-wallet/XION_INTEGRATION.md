# XION Meta Accounts Integration

This document describes the XION Meta Accounts integration implemented in Month 3 of the KNIRV D-TEN implementation plan.

## Overview

The XION integration provides:
- Meta account creation and management
- NRN token operations (transfer, burn for skills)
- Gasless transaction support
- Faucet integration for testnet
- Secure wallet storage

## Components

### 1. XionMetaAccount Class (`src/xion-meta-accounts.ts`)
- Handles XION blockchain interactions
- Manages wallet creation and restoration
- Provides NRN token operations
- Supports gasless transactions

### 2. WalletManager Class (`src/xion-meta-accounts.ts`)
- Manages multiple wallets
- Provides secure storage (encrypted)
- Handles wallet import/export

### 3. MetaAccountDashboard Component (`src/components/MetaAccountDashboard.tsx`)
- React Native UI for XION meta accounts
- Wallet creation and switching
- Balance display and refresh
- Transfer and skill invocation forms

### 4. Configuration (`src/config/xion-config.ts`)
- Network configurations (testnet/mainnet)
- Contract addresses
- RPC endpoints

## Setup

1. Copy `.env.example` to `.env` and configure contract addresses:
```bash
cp .env.example .env
```

2. Update the contract addresses in `.env`:
```
EXPO_PUBLIC_NRN_CONTRACT=xion1your_nrn_contract_address
EXPO_PUBLIC_FAUCET_CONTRACT=xion1your_faucet_contract_address
```

## Usage

### Creating a Wallet
```typescript
import { WalletManager, getXionConfig } from './src/xion-meta-accounts';

const config = getXionConfig('testnet');
const walletManager = new WalletManager(config);
const wallet = await walletManager.createWallet('my-wallet');
```

### Getting NRN Balance
```typescript
const balance = await wallet.getNRNBalance();
console.log(`Balance: ${balance} NRN`);
```

### Transferring NRN
```typescript
const txHash = await wallet.transferNRN(recipientAddress, amount);
console.log(`Transfer TX: ${txHash}`);
```

### Burning NRN for Skills
```typescript
const txHash = await wallet.burnNRNForSkill(skillId, amount);
console.log(`Skill invocation TX: ${txHash}`);
```

## UI Integration

The XION meta account functionality is integrated into the existing wallet tab as a new "XION Meta" tab. Users can:

1. Create new XION meta accounts
2. View and refresh NRN balances
3. Request NRN from the testnet faucet
4. Transfer NRN to other addresses
5. Burn NRN for skill invocations
6. Enable gasless transactions

## Security

- Mnemonics are encrypted before storage
- Private keys never leave the device
- All transactions are signed locally
- Secure storage implementation (placeholder for production)

## Dependencies

- `@cosmjs/proto-signing`: Wallet and signing functionality
- `@cosmjs/cosmwasm-stargate`: CosmWasm contract interactions
- `@cosmjs/stargate`: Cosmos SDK interactions

## Future Enhancements

- Hardware wallet support
- Multi-signature accounts
- Advanced account abstraction features
- Cross-chain bridging
- Enhanced security with biometric authentication
