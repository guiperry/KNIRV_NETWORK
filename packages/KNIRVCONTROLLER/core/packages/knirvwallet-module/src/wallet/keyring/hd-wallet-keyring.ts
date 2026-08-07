import { Slip10, Slip10Curve, Slip10RawIndex, HdPath, Secp256k1 } from '@cosmjs/crypto';
import type {
  Provider,
  Tx,
  TxSignature,
} from '../wallet';
import { broadcastCommit, broadcastSync } from './broadcast';

// Standard Cosmos/Gno HD path: m/44'/118'/0'/0/{accountIndex}
// (matches ledger-keyring.ts's generateHDPath so software and hardware
// wallets derive the same account at the same index).
function generateHDPath(accountIndex: number): HdPath {
  return [
    Slip10RawIndex.hardened(44),
    Slip10RawIndex.hardened(118),
    Slip10RawIndex.hardened(0),
    Slip10RawIndex.normal(0),
    Slip10RawIndex.normal(accountIndex),
  ];
}

// Derives a real secp256k1 keypair from a BIP-39 seed via SLIP-10.
// Replaces a placeholder that returned zero-filled keys regardless of input.
async function generateKeyPair(seed: Uint8Array, hdPath: number): Promise<{ privateKey: Uint8Array; publicKey: Uint8Array }> {
  const { privkey } = Slip10.derivePath(Slip10Curve.Secp256k1, seed, generateHDPath(hdPath));
  const { pubkey } = await Secp256k1.makeKeypair(privkey);
  return {
    privateKey: privkey,
    publicKey: Secp256k1.compressPubkey(pubkey),
  };
}
import { v4 as uuidv4 } from 'uuid';

import { Bip39, EnglishMnemonic, entropyToMnemonic, mnemonicToEntropy } from '../../crypto';
import { toBase64 } from '../../encoding';
import { documentToTx } from '../../utils/messages';
import type { Document } from '../../utils/messages';
import { SimpleKNIRVWallet, KNIRVWallet } from './keyring-util';
import { Keyring, KeyringData, KeyringType } from './keyring';

export class HDWalletKeyring implements Keyring {
  public readonly id: string;
  public readonly type: KeyringType = 'HD';
  public readonly seed: Uint8Array;
  public readonly mnemonicEntropy: Uint8Array;

  constructor({ id, mnemonicEntropy, seed }: KeyringData) {
    if (!mnemonicEntropy || !seed) {
      throw new Error('Invalid parameter values');
    }
    this.id = id || uuidv4();
    this.mnemonicEntropy = Uint8Array.from(mnemonicEntropy);
    this.seed = Uint8Array.from(seed);
  }

  getMnemonic() {
    return entropyToMnemonic(this.mnemonicEntropy);
  }

  async getKeypair(hdPath: number) {
    const { privateKey, publicKey } = await generateKeyPair(this.seed, hdPath);
    return { privateKey, publicKey: publicKey };
  }

  async getPrivateKey(hdPath: number) {
    const { privateKey } = await this.getKeypair(hdPath);
    return privateKey;
  }

  async getPublicKey(hdPath: number) {
    const { publicKey } = await this.getKeypair(hdPath);
    return publicKey;
  }

  toData() {
    return {
      id: this.id,
      type: this.type,
      seed: Array.from(this.seed),
      mnemonicEntropy: Array.from(this.mnemonicEntropy),
    };
  }

  async sign(
    provider: Provider,
    document: Document,
    hdPath: number = 0,
  ): Promise<{
    signed: Tx;
    signature: TxSignature[];
  }> {
    const wallet = await SimpleKNIRVWallet.fromMnemonic(this.getMnemonic(), {
      accountIndex: hdPath,
    });
    wallet.connect(provider);
    return this.signByWallet(wallet, document);
  }

  private async signByWallet(wallet: KNIRVWallet, document: Document) {
    const signedTx = await wallet.signTransaction(documentToTx(document), document);
    const pubKeyBase64 = toBase64(await wallet.getPublicKey());
		if (!signedTx.signatures?.length) {
			throw new Error('Canonical signer returned no signature');
		}
		const signatures = signedTx.signatures.map((sig) => ({
					pub_key: {
						key: pubKeyBase64,
					},
					signature: sig,
				}));
    return {
      signed: signedTx,
      signature: signatures,
    };
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx, _hdPath: number = 0) {
    return broadcastSync(provider, signedTx);
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx, _hdPath: number = 0) {
    return broadcastCommit(provider, signedTx);
  }

  public static async fromMnemonic(mnemonic: string) {
    const englishMnemonic = new EnglishMnemonic(mnemonic);
    const seed = await Bip39.mnemonicToSeed(englishMnemonic);
    const mnemonicEntropy = await mnemonicToEntropy(englishMnemonic.toString());
    return new HDWalletKeyring({
      mnemonicEntropy: Array.from(mnemonicEntropy),
      seed: Array.from(seed),
    });
  }
}
