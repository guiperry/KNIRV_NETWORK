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

import { Document } from '../..';
import { fromBech32 } from '../../encoding';
import { Keyring, KeyringData, KeyringType } from './keyring';
import { broadcastCommit, broadcastSync } from './broadcast';

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
    return broadcastSync(provider, signedTx);
  }

  async broadcastTxCommit(provider: Provider, signedTx: Tx) {
    return broadcastCommit(provider, signedTx);
  }

  public static async fromAddress(address: string) {
    const { data: addressBytes } = fromBech32(address);
    return new AddressKeyring({ addressBytes: [...addressBytes] });
  }
}
