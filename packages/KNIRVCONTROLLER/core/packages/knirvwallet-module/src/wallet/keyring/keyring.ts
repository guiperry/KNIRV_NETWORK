// Import KNIRV-compatible types from wallet
import type {
  BroadcastTxCommitResult,
  BroadcastTxSyncResult,
  Provider,
  Tx,
  TxSignature,
} from '../wallet';

import { Document } from '../..';
import { AddressKeyring } from './address-keyring';
import { HDWalletKeyring } from './hd-wallet-keyring';
import { LedgerKeyring } from './ledger-keyring';
import { PrivateKeyKeyring } from './private-key-keyring';
import { Web3AuthKeyring } from './web3-auth-keyring';

export type KeyringType = 'HD' | 'PRIVATE_KEY' | 'LEDGER' | 'WEB3_AUTH' | 'ADDRESS';

export interface Keyring {
  id: string;
  type: KeyringType;
  toData: () => KeyringData;
  sign: (
    provider: Provider,
    document: Document,
    hdPath?: number,
  ) => Promise<{
    signed: Tx;
    signature: TxSignature[];
  }>;
  broadcastTxSync: (
    provider: Provider,
    signedTx: Tx,
    hdPath?: number,
  ) => Promise<BroadcastTxSyncResult>;
  broadcastTxCommit: (
    provider: Provider,
    signedTx: Tx,
    hdPath?: number,
  ) => Promise<BroadcastTxCommitResult>;
}

export interface KeyringData {
  id?: string;
  type?: KeyringType;
  publicKey?: number[];
  privateKey?: number[];
  seed?: number[];
  mnemonicEntropy?: number[];
  addressBytes?: number[];
}

export function makeKeyring(keyringData: KeyringData) {
  switch (keyringData.type) {
    case 'HD':
      return new HDWalletKeyring(keyringData);
    case 'LEDGER':
      return new LedgerKeyring(keyringData);
    case 'PRIVATE_KEY':
      return new PrivateKeyKeyring(keyringData);
    case 'WEB3_AUTH':
      return new Web3AuthKeyring(keyringData);
    case 'ADDRESS':
      return new AddressKeyring(keyringData);
    default:
      throw new Error('Invalid Account type');
  }
}
