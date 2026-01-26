import type { Provider, Tx } from '../wallet';
import { v4 as uuidv4 } from 'uuid';

import { Document, makeSignedTx, useTm2Wallet, KNIRVWallet } from '../..';
import { Keyring, KeyringData, KeyringType } from './keyring';

export class PrivateKeyKeyring implements Keyring {
  public readonly id: string;
  public readonly type: KeyringType = 'PRIVATE_KEY';
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

  async sign(provider: Provider, document: Document) {
    const wallet = await useTm2Wallet(document).fromPrivateKey(this.privateKey);
    wallet.connect(provider);
    return this.signByWallet(wallet, document);
  }

  private async signByWallet(wallet: KNIRVWallet, document: Document) {
    const signedTx = await makeSignedTx(wallet, document);
    // Convert string signatures to TxSignature format
    const signatures = (signedTx.signatures || []).map(sig => ({
      pub_key: {
        key: '', // Placeholder - would be derived from wallet
      },
      signature: sig,
    }));
    return {
      signed: signedTx,
      signature: signatures,
    };
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx) {
    // For KNIRV, we'll use the transaction SDK to submit transactions
    // This is a placeholder implementation
    return {
      hash: 'placeholder-hash',
      code: 0,
      log: 'Transaction broadcasting not implemented yet - use KNIRV transaction SDK',
    };
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx) {
    // For KNIRV, we'll use the transaction SDK to submit transactions
    // This is a placeholder implementation
    return {
      hash: 'placeholder-hash',
      height: 0,
      code: 0,
      log: 'Transaction broadcasting not implemented yet - use KNIRV transaction SDK',
      gasUsed: 0,
      gasWanted: 0,
    };
  }

  public static async fromPrivateKeyStr(privateKeyStr: string) {
    const adjustPrivateKeyStr = privateKeyStr.replace('0x', '');
    const privateKey = Uint8Array.from(Buffer.from(adjustPrivateKeyStr, 'hex'));
    const wallet = await useTm2Wallet({} as Document).fromPrivateKey(privateKey);
    const publicKey = await wallet.getPublicKey();
    return new PrivateKeyKeyring({
      publicKey: Array.from(publicKey),
      privateKey: Array.from(privateKey),
    });
  }
}
