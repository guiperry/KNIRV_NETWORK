import { Secp256k1, Slip10, Slip10Curve, Slip10RawIndex, HdPath } from '@cosmjs/crypto';

import type { Tx } from '../wallet';
import { decodeTxMessages, Document, documentToTx } from '../../utils/messages';
import { toBase64 } from '../../encoding';
import { sha256 } from '../../crypto';
import { publicKeyToAddress } from '../../utils/address';
import { Bip39, EnglishMnemonic } from '../../crypto';

// KNIRV Wallet interface to replace Tm2Wallet
export interface KNIRVWallet {
  connect(provider: any): void;
  signTransaction(tx: Tx, decodeFn?: any): Promise<Tx>;
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
  private provider: any;

  constructor(privateKey: Uint8Array) {
    this.privateKey = privateKey;
  }

  connect(provider: any): void {
    this.provider = provider;
  }

  // Signs a deterministic JSON serialization of `tx` with secp256k1
  // (RFC 6979 deterministic, low-S, Cosmos-native 64-byte r|s encoding).
  //
  // Note: this does not yet implement Cosmos/Gno SIGN_MODE_DIRECT or
  // LEGACY_AMINO_JSON canonical encoding, so a chain node expecting one of
  // those wire formats will not accept this signature as-is. No canonical
  // encoding for KNIRV's Tx shape is defined anywhere in this codebase yet -
  // that is tracked separately (see docs/Bootnode_Failover_Implementation_Plan.md
  // Phase 0.4 in KNIRV_NETWORK). What this fixes is the mock itself: the
  // previous implementation returned `tx` completely unsigned.
  async signTransaction(tx: Tx, decodeFn?: any): Promise<Tx> {
    const { pubkey } = await Secp256k1.makeKeypair(this.privateKey);
    const compressedPubkey = Secp256k1.compressPubkey(pubkey);
    const messageHash = sha256(new TextEncoder().encode(JSON.stringify(tx)));
    const signature = await Secp256k1.createSignature(messageHash, this.privateKey);
    const signatureBase64 = toBase64(signature.toFixedLength());

    return {
      ...tx,
      auth_info: {
        ...tx.auth_info,
        signer_infos: [
          {
            public_key: { key: toBase64(compressedPubkey) },
            mode_info: { single: { mode: 1 } },
            sequence: tx.auth_info?.signer_infos?.[0]?.sequence ?? '0',
          },
        ],
      },
      signatures: [signatureBase64],
    };
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

  static async fromLedger(connector: any, options?: { accountIndex?: number }): Promise<KNIRVWallet> {
    // Ledger signing requires delegating the signature operation to the
    // connected hardware device (it never exposes a private key to sign
    // locally with, unlike fromMnemonic/fromPrivateKey above). That
    // integration isn't wired up yet, so this throws rather than silently
    // returning a wallet backed by a zero-filled key, which would look valid
    // but sign nothing meaningful.
    throw new Error('SimpleKNIRVWallet.fromLedger is not implemented yet; Ledger signing requires direct connector integration.');
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

export function useTm2Wallet(document: Document): typeof SimpleKNIRVWallet {
  return SimpleKNIRVWallet;
}

export function makeSignedTx(wallet: KNIRVWallet, document: Document): Promise<Tx> {
  const tx = documentToTx(document);
  const decodeTxMessageFunction = decodeTxMessages;

  return wallet.signTransaction(tx, decodeTxMessageFunction);
}
