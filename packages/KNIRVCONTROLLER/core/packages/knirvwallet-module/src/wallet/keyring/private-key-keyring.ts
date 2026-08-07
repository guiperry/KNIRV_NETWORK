import type { Provider, Tx } from '../wallet';
import { v4 as uuidv4 } from 'uuid';

import { documentToTx } from '../../utils/messages';
import type { Document } from '../../utils/messages';
import { SimpleKNIRVWallet, KNIRVWallet } from './keyring-util';
import { Keyring, KeyringData, KeyringType } from './keyring';
import { broadcastCommit, broadcastSync } from './broadcast';
import { toBase64 } from '../../encoding';

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
    const wallet = await SimpleKNIRVWallet.fromPrivateKey(this.privateKey);
    wallet.connect(provider);
    return this.signByWallet(wallet, document);
  }

  private async signByWallet(wallet: KNIRVWallet, document: Document) {
    const signedTx = await wallet.signTransaction(documentToTx(document), document);
		if (!signedTx.signatures?.length) {
			throw new Error('Canonical signer returned no signature');
		}
		const publicKey = toBase64(await wallet.getPublicKey());
		const signatures = signedTx.signatures.map((sig) => ({
					pub_key: {
						key: publicKey,
					},
					signature: sig,
				}));
    return {
      signed: signedTx,
      signature: signatures,
    };
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx) {
    return broadcastSync(provider, signedTx);
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx) {
    return broadcastCommit(provider, signedTx);
  }

  public static async fromPrivateKeyStr(privateKeyStr: string) {
    const adjustPrivateKeyStr = privateKeyStr.replace('0x', '');
    const privateKey = Uint8Array.from(Buffer.from(adjustPrivateKeyStr, 'hex'));
    const wallet = await SimpleKNIRVWallet.fromPrivateKey(privateKey);
    const publicKey = await wallet.getPublicKey();
    return new PrivateKeyKeyring({
      publicKey: Array.from(publicKey),
      privateKey: Array.from(privateKey),
    });
  }
}
