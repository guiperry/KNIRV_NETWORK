import type {
  Provider,
  Tx,
  TxSignature,
} from '../wallet';

// Utility function to replace uint8ArrayToBase64
export function uint8ArrayToBase64(uint8Array: Uint8Array): string {
  return btoa(String.fromCharCode(...uint8Array));
}
import { v4 as uuidv4 } from 'uuid';

import { Document, fromBech32 } from '../..';
import { Keyring, KeyringData, KeyringType } from './keyring';

export class AddressKeyring implements Keyring {
  public readonly id: string;
  public readonly type: KeyringType = 'ADDRESS';
  public readonly addressBytes: Uint8Array;

  constructor({ id, addressBytes }: KeyringData) {
    if (!addressBytes) {
      throw new Error('Invalid parameter values');
    }
    this.id = id || uuidv4();
    this.addressBytes = Uint8Array.from(addressBytes);
  }

  toData() {
    return {
      id: this.id,
      type: this.type,
      addressBytes: Array.from(this.addressBytes),
    };
  }

  async sign(
    _provider: Provider,
    _document: Document,
  ): Promise<{
    signed: Tx;
    signature: TxSignature[];
  }> {
    throw new Error('Not support transaction sign');
  }

  async broadcastTxSync(provider: Provider, signedTx: Tx) {
    // For KNIRV, we'll use the transaction SDK to submit transactions
    // This is a placeholder implementation - in practice, you'd use the KNIRV transaction client
    return {
      hash: 'placeholder-hash',
      code: 1,
      log: 'Transaction broadcasting not implemented for airgap accounts',
    };
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx) {
    // For KNIRV, we'll use the transaction SDK to submit transactions
    // This is a placeholder implementation - in practice, you'd use the KNIRV transaction client
    return {
      hash: 'placeholder-hash',
      height: 0,
      code: 1,
      log: 'Transaction broadcasting not implemented for airgap accounts',
      gasUsed: 0,
      gasWanted: 0,
    };
  }

  public static async fromAddress(address: string) {
    const { data: addressBytes } = fromBech32(address);
    return new AddressKeyring({ addressBytes: [...addressBytes] });
  }
}
