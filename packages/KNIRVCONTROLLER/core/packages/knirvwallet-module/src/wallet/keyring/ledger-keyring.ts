import { LedgerConnector } from '@cosmjs/ledger-amino';
import { Slip10RawIndex, HdPath } from '@cosmjs/crypto';
import type { Provider, Tx } from '../wallet';
import { broadcastCommit, broadcastSync } from './broadcast';

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

import type { Document } from '../../utils/messages';
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

  async sign(_provider: Provider, _document: Document, _hdPath: number = 0): Promise<never> {
    if (!this.connector) {
      throw new Error('Ledger connector does not found');
    }
    throw new Error('The Cosmos Ledger app supports legacy Amino signing only; KNIRV requires SIGN_MODE_DIRECT. Approve this action in KNIRVCONTROLLER instead.');
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx, _hdPath: number = 0) {
    return broadcastSync(provider, signedTx);
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx, _hdPath: number = 0) {
    return broadcastCommit(provider, signedTx);
  }

  public static async fromLedger(connector: LedgerConnector) {
    const keyring = new LedgerKeyring({});
    keyring.setConnector(connector);
    return keyring;
  }
}
