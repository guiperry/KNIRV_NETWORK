# Agentic Wallet: README

[TOC]

## Overview

The Agentic Wallet is a secure, multi-chain wallet application. This document details the integration of XION Meta Accounts, a key feature implemented in Month 3 of the KNIRV D-TEN project.  The XION integration provides meta account creation and management, NRN token operations (transfer, burn for skills), gasless transaction support, testnet faucet integration, and secure wallet storage.

## Features

- **XION Meta Account Management:** Create, import, and manage XION meta accounts.
- **NRN Token Operations:** Transfer and burn NRN tokens.
- **Gasless Transactions:** Execute transactions without paying gas fees.
- **Testnet Faucet Integration:** Easily acquire NRN tokens for testing on the testnet.
- **Secure Wallet Storage:** Encrypted storage of mnemonics and private keys.
- **Multi-Wallet Management:** Manage multiple wallets simultaneously.
- **User-Friendly Interface:** Intuitive React Native UI for managing XION meta accounts.


## XION Meta Accounts Integration

### Components

| Component                     | Description                                                                        | Location                                     |
|---------------------------------|------------------------------------------------------------------------------------|-------------------------------------------------|
| `XionMetaAccount` Class        | Handles XION blockchain interactions, wallet management, and NRN token operations. | `src/xion-meta-accounts.ts`                   |
| `WalletManager` Class          | Manages multiple wallets and provides secure, encrypted storage.                 | `src/xion-meta-accounts.ts`                   |
| `MetaAccountDashboard` Component | React Native UI for interacting with XION meta accounts.                           | `src/components/MetaAccountDashboard.tsx`     |
| Configuration (`xion-config.ts`) | Network configurations, contract addresses, and RPC endpoints.                    | `src/config/xion-config.ts`                   |


### Setup

1. Copy the example environment file and configure contract addresses:
   ```bash
   cp .env.example .env
   ```
2. Update the contract addresses in `.env`:
   ```
   EXPO_PUBLIC_NRN_CONTRACT=xion1your_nrn_contract_address
   EXPO_PUBLIC_FAUCET_CONTRACT=xion1your_faucet_contract_address
   ```

### Usage

#### Creating a Wallet

```typescript
import { WalletManager, getXionConfig } from './src/xion-meta-accounts';

const config = getXionConfig('testnet');
const walletManager = new WalletManager(config);
const wallet = await walletManager.createWallet('my-wallet');
```

#### Getting NRN Balance

```typescript
const balance = await wallet.getNRNBalance();
console.log(`Balance: ${balance} NRN`);
```

#### Transferring NRN

```typescript
const txHash = await wallet.transferNRN(recipientAddress, amount);
console.log(`Transfer TX: ${txHash}`);
```

#### Burning NRN for Skills

```typescript
const txHash = await wallet.burnNRNForSkill(skillId, amount);
console.log(`Skill invocation TX: ${txHash}`);
```

### UI Integration

The XION meta account functionality is integrated into the wallet application as a new "XION Meta" tab.  Users can create new XION meta accounts, view and refresh NRN balances, request NRN from the testnet faucet, transfer NRN, burn NRN for skill invocations, and enable gasless transactions.


## Security

- Mnemonics are encrypted before storage.
- Private keys never leave the device.
- All transactions are signed locally.
- Secure storage implementation (placeholder for production).  Future enhancements include biometric authentication.


## Dependencies

- `@cosmjs/proto-signing`: Wallet and signing functionality.
- `@cosmjs/cosmwasm-stargate`: CosmWasm contract interactions.
- `@cosmjs/stargate`: Cosmos SDK interactions.


## Future Enhancements

- Hardware wallet support
- Multi-signature accounts
- Advanced account abstraction features
- Cross-chain bridging
- Enhanced security with biometric authentication

