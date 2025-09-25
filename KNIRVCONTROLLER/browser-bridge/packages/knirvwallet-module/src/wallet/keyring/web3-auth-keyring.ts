import type { Provider, Tx, TxSignature } from '../wallet';
import { v4 as uuidv4 } from 'uuid';
import { Secp256k1Wallet, StdFee, StdSignDoc } from '@cosmjs/amino';
import { fromBase64 } from '@cosmjs/encoding';

import { Document } from '../..';
import { hexToArray, arrayToHex } from '../../utils/data';
import { Keyring, KeyringData, KeyringType } from './keyring';

// Type assertion to handle KNIRV-specific types
interface KNIRVPubKey {
  key: string;
}

interface KNIRVSignerInfo {
  public_key: KNIRVPubKey;
  mode_info: { single: { mode: number } };
  sequence: string;
}

function convertToStdSignDoc(doc: Document): StdSignDoc {
  return {
    chain_id: doc.chain_id,
    account_number: doc.account_number.toString(),
    sequence: doc.sequence.toString(),
    fee: {
      amount: [...doc.fee.amount.map(coin => ({
        denom: coin.denom,
        amount: coin.amount.toString()
      }))],
      gas: doc.fee.gas.toString()
    },
    msgs: [...doc.msgs],
    memo: doc.memo || ''
  };
}

async function convertToTx(privateKey: Uint8Array, signed: StdSignDoc, signature: Uint8Array): Promise<Tx> {
  // Cast to KNIRV-specific types
  const wallet = await Secp256k1Wallet.fromKey(privateKey);
  const accounts = await wallet.getAccounts();
  
  const signer_info: KNIRVSignerInfo = {
    public_key: {
      key: arrayToHex(accounts[0].pubkey)
    },
    mode_info: { single: { mode: 1 } }, // SIGN_MODE_DIRECT = 1
    sequence: signed.sequence // Already a string from StdSignDoc
  };
  // Convert AminoMsg to Any type
  const messages = signed.msgs.map(msg => ({
    type_url: msg.type,
    value: msg.value
  }));

  return {
    body: {
      messages,
      memo: signed.memo,
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: []
    },
    auth_info: {
      signer_infos: [signer_info],
      fee: {
        amount: [...signed.fee.amount],
        gas: signed.fee.gas,
        granter: '',
        payer: ''
      }
    },
    signatures: [arrayToHex(signature)]
  };
}

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

  async sign(provider: Provider, document: Document, hdPath?: number) {
    const wallet = await Secp256k1Wallet.fromKey(this.privateKey);
    const accounts = await wallet.getAccounts();
    
    const stdSignDoc = convertToStdSignDoc({
      ...document,
      chain_id: provider.chainId
    });

    const { signature } = await wallet.signAmino(accounts[0].address, stdSignDoc);
    const tx = await convertToTx(this.privateKey, stdSignDoc, fromBase64(signature.signature));

    return {
      signed: tx,
      signature: [{
        pub_key: {
          key: arrayToHex(accounts[0].pubkey)
        },
        signature: signature.signature
      }]
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
