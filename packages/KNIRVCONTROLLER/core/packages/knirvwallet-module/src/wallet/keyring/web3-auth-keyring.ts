import type { Provider, Tx } from '../wallet';
import { v4 as uuidv4 } from 'uuid';
import { Secp256k1Wallet } from '@cosmjs/amino';

import { Document } from '../..';
import { hexToArray } from '../../utils/data';
import { documentToTx } from '../../utils/messages';
import { Keyring, KeyringData, KeyringType } from './keyring';
import { broadcastCommit, broadcastSync } from './broadcast';
import { SimpleKNIRVWallet } from './keyring-util';

export class Web3AuthKeyring implements Keyring {
  public readonly id: string;
  public readonly type: KeyringType = 'WEB3_AUTH';
  public readonly publicKey: Uint8Array;
  public readonly privateKey: Uint8Array;

  constructor({ id, publicKey, privateKey }: KeyringData) {
    if (!publicKey || !privateKey) {
      throw new Error('Invalid parameter values');
    }
    this.id = id || uuidv4();
    this.publicKey = Uint8Array.from(publicKey);
    this.privateKey = Uint8Array.from(privateKey);
  }

  toData() {
    return {
      id: this.id,
      type: this.type,
      publicKey: Array.from(this.publicKey),
      privateKey: Array.from(this.privateKey),
    };
  }

  async sign(provider: Provider, document: Document, _hdPath?: number) {
    const wallet = await SimpleKNIRVWallet.fromPrivateKey(this.privateKey);
    wallet.connect(provider);
    const canonicalDocument = { ...document, chain_id: provider.chainId };
    const signed = await wallet.signTransaction(documentToTx(canonicalDocument), canonicalDocument);
    return {
      signed,
      signature: signed.signatures.map((signature) => ({
        pub_key: { key: (signed as any).public_key as string },
        signature,
      })),
    };
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx) {
    return broadcastSync(provider, signedTx);
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx) {
    return broadcastCommit(provider, signedTx);
  }

  public static async fromPrivateKey(privateKey: Uint8Array) {
    const wallet = await Secp256k1Wallet.fromKey(privateKey);
    const accounts = await wallet.getAccounts();
    const publicKey = accounts[0].pubkey;
    
    return new Web3AuthKeyring({
      publicKey: Array.from(publicKey),
      privateKey: Array.from(privateKey)
    });
  }

  public static async fromPrivateKeyStr(privateKeyStr: string) {
    const privateKey = hexToArray(privateKeyStr);
    const wallet = await Secp256k1Wallet.fromKey(privateKey);
    const accounts = await wallet.getAccounts();
    const publicKey = accounts[0].pubkey;
    return new Web3AuthKeyring({
      publicKey: Array.from(publicKey),
      privateKey: Array.from(privateKey)
    });
  }
}
