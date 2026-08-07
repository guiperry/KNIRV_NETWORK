import { Secp256k1, Slip10, Slip10Curve, Slip10RawIndex, HdPath } from '@cosmjs/crypto';

import type { Tx } from '../wallet';
import { decodeTxMessages, Document, documentToTx } from '../../utils/messages';
import { publicKeyToAddress } from '../../utils/address';
import { Bip39, EnglishMnemonic } from '../../crypto';
import { signDirectTransaction, type DirectSignRequest } from '@knirv/sdk/signing';

// KNIRV Wallet interface to replace Tm2Wallet
export interface KNIRVWallet {
  connect(provider: any): void;
  signTransaction(tx: Tx, document?: Document): Promise<Tx>;
  getPublicKey(): Promise<Uint8Array>;
  getAddress(): Promise<string>;
}

// Same HD path used elsewhere in this package (hd-wallet-keyring.ts,
// ledger-keyring.ts): m/44'/118'/0'/0/{accountIndex}.
function generateHDPath(accountIndex: number): HdPath {
  return [
    Slip10RawIndex.hardened(44),
    Slip10RawIndex.hardened(118),
    Slip10RawIndex.hardened(0),
    Slip10RawIndex.normal(0),
    Slip10RawIndex.normal(accountIndex),
  ];
}

// Simple KNIRV wallet implementation
export class SimpleKNIRVWallet implements KNIRVWallet {
  private privateKey: Uint8Array;

  constructor(privateKey: Uint8Array) {
    this.privateKey = privateKey;
  }

  connect(_provider: any): void {
  }

  async signTransaction(tx: Tx, document?: Document): Promise<Tx> {
    if (!document) throw new Error('Cosmos SIGN_MODE_DIRECT requires the original signing document');
    const address = await this.getAddress();
    const decoded = decodeTxMessages(tx.body.messages);
    const first = decoded[0] || {};
    const rawAmount = first.amount?.[0]?.amount ?? first.amount ?? first.value ?? 0;
    const numericAmount = /^\d+$/.test(String(rawAmount)) ? String(rawAmount) : '0';
    const request: DirectSignRequest = {
      action: {
        action: first['@type'] || 'knirv.transaction',
        sender: first.from_address || first.caller || first.creator || first.sender || address,
        recipient: first.to_address || first.recipient || first.contract || '',
        amount: numericAmount,
        payload: new TextEncoder().encode(JSON.stringify({ messages: decoded, memo: document.memo || '' })),
        timestampUnix: Math.floor(Date.now() / 1000),
      },
      chainId: document.chain_id,
      accountNumber: document.account_number,
      sequence: document.sequence,
      fee: {
        denom: document.fee.amount[0]?.denom,
        amount: document.fee.amount[0]?.amount,
        gasLimit: document.fee.gas,
        payer: document.fee.payer,
        granter: document.fee.granter,
      },
    };
    const signed = await signDirectTransaction(this.privateKey, request);
    return {
      ...tx,
      ...signed,
      signatures: signed.signatures,
    } as Tx;
  }

  async getPublicKey(): Promise<Uint8Array> {
    const { pubkey } = await Secp256k1.makeKeypair(this.privateKey);
    return Secp256k1.compressPubkey(pubkey);
  }

  async getAddress(): Promise<string> {
    const publicKey = await this.getPublicKey();
    return publicKeyToAddress(publicKey);
  }

  static async fromPrivateKey(privateKey: Uint8Array): Promise<KNIRVWallet> {
    return new SimpleKNIRVWallet(privateKey);
  }

  static async fromMnemonic(mnemonic: string, options?: { accountIndex?: number }): Promise<KNIRVWallet> {
    const englishMnemonic = new EnglishMnemonic(mnemonic);
    const seed = await Bip39.mnemonicToSeed(englishMnemonic);
    const { privkey } = Slip10.derivePath(
      Slip10Curve.Secp256k1,
      seed,
      generateHDPath(options?.accountIndex ?? 0),
    );
    return new SimpleKNIRVWallet(privkey);
  }

}
import { AddressKeyring } from './address-keyring';
import { HDWalletKeyring } from './hd-wallet-keyring';
import { Keyring } from './keyring';
import { LedgerKeyring } from './ledger-keyring';
import { PrivateKeyKeyring } from './private-key-keyring';
import { Web3AuthKeyring } from './web3-auth-keyring';

export function isHDWalletKeyring(keyring: Keyring): keyring is HDWalletKeyring {
  return keyring.type === 'HD';
}

export function isLedgerKeyring(keyring: Keyring): keyring is LedgerKeyring {
  return keyring.type === 'LEDGER';
}

export function isPrivateKeyKeyring(keyring: Keyring): keyring is PrivateKeyKeyring {
  return keyring.type === 'PRIVATE_KEY';
}

export function isWeb3AuthKeyring(keyring: Keyring): keyring is Web3AuthKeyring {
  return keyring.type === 'WEB3_AUTH';
}

export function isAddressKeyring(keyring: Keyring): keyring is AddressKeyring {
  return keyring.type === 'ADDRESS';
}

export function hasPrivateKey(
  keyring: Keyring,
): keyring is HDWalletKeyring | PrivateKeyKeyring | Web3AuthKeyring {
  if (isHDWalletKeyring(keyring)) {
    return true;
  }
  if (isPrivateKeyKeyring(keyring)) {
    return true;
  }
  if (isWeb3AuthKeyring(keyring)) {
    return true;
  }
  return false;
}

export function useTm2Wallet(_document: Document): typeof SimpleKNIRVWallet {
  return SimpleKNIRVWallet;
}

export function makeSignedTx(wallet: KNIRVWallet, document: Document): Promise<Tx> {
  const tx = documentToTx(document);
  return wallet.signTransaction(tx, document);
}
