import { LedgerConnector } from '@cosmjs/ledger-amino';
import { Slip10RawIndex, HdPath } from '@cosmjs/crypto';
import type { Provider, Tx } from '../wallet';

// Utility function to replace generateHDPath
function generateHDPath(accountIndex: number): HdPath {
  return [
    Slip10RawIndex.hardened(44),
    Slip10RawIndex.hardened(118),
    Slip10RawIndex.hardened(0),
    Slip10RawIndex.normal(0),
    Slip10RawIndex.normal(accountIndex),
  ];
}
import { v4 as uuidv4 } from 'uuid';

import { toBase64 } from '../../encoding';
import { documentToTx } from '../../utils/messages';
import type { Document } from '../../utils/messages';
import { SimpleKNIRVWallet, KNIRVWallet } from './keyring-util';
import { Keyring, KeyringData, KeyringType } from './keyring';

export class LedgerKeyring implements Keyring {
  public readonly id: string;
  public readonly type: KeyringType = 'LEDGER';
  private connector: LedgerConnector | null;

  constructor({ id }: KeyringData) {
    this.id = id || uuidv4();
    this.connector = null;
  }

  setConnector(connector: LedgerConnector) {
    this.connector = connector;
  }

  getPublicKey(hdPath: number) {
    if (!this.connector) {
      throw new Error('Ledger connector does not found');
    }
    const gnoHdPath = generateHDPath(hdPath);
    return this.connector.getPubkey(gnoHdPath);
  }

  toData() {
    return {
      id: this.id,
      type: this.type,
    };
  }

  async sign(provider: Provider, document: Document, hdPath: number = 0) {
    if (!this.connector) {
      throw new Error('Ledger connector does not found');
    }
    const wallet = await SimpleKNIRVWallet.fromLedger(this.connector as any, {
      accountIndex: hdPath,
    });
    wallet.connect(provider);
    return this.signByWallet(wallet, document);
  }

  private async signByWallet(wallet: KNIRVWallet, document: Document) {
    const signedTx = await wallet.signTransaction(documentToTx(document));
    const pubKeyBase64 = toBase64(await wallet.getPublicKey());
    const signatures =
      (signedTx.signatures || []).length > 0
        ? signedTx.signatures.map((sig) => ({
            pub_key: {
              key: pubKeyBase64,
            },
            signature: sig,
          }))
        : [
            {
              pub_key: {
                key: pubKeyBase64,
              },
              signature: '',
            },
          ];
    return {
      signed: signedTx,
      signature: signatures,
    };
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx, hdPath: number = 0) {
    // For KNIRV, we'll use the transaction SDK to submit transactions
    // This is a placeholder implementation
    return {
      hash: 'placeholder-hash',
      code: 0,
      log: 'Transaction broadcasting not implemented yet - use KNIRV transaction SDK',
    };
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx, hdPath: number = 0) {
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

  public static async fromLedger(connector: LedgerConnector) {
    const keyring = new LedgerKeyring({});
    keyring.setConnector(connector);
    return keyring;
  }
}
