// KNIRV Network transaction types and utilities
import { fromBase64 } from '../encoding';

// KNIRV Message Types
export const KNIRVMessageTypes = {
  MSG_SEND: '/knirv.transaction.v1.MsgSend',
  MSG_CALL: '/knirv.transaction.v1.MsgCall',
  MSG_ADD_PACKAGE: '/knirv.transaction.v1.MsgAddPackage',
  MSG_RUN: '/knirv.transaction.v1.MsgRun',
  MSG_MCP_INVOKE: '/knirv.mcp.v1.MsgInvoke',
  MSG_MCP_REGISTER: '/knirv.mcp.v1.MsgRegister',
} as const;

// KNIRV Transaction Interfaces
export interface KNIRVTransaction {
  id: string;
  fee: string;
  from: string;
  public_key: string;
  signature: string;
  timestamp: number;
  type: string;
  version: number;
  data?: Record<string, unknown>;
  status?: string;
  to?: string | null;
  transaction_hash?: string;
  value?: string;
}

export interface KNIRVMessage {
  '@type': string;
  [key: string]: any;
}

export interface Any {
  type_url: string;
  value: Uint8Array;
}

export interface PubKeySecp256k1 {
  key: string;
}

export interface TxFee {
  amount: Array<{ denom: string; amount: string }>;
  gas: string;
  granter?: string;
  payer?: string;
}

export interface TxSignature {
  pub_key: PubKeySecp256k1;
  signature: string;
}

export interface Tx {
  body: {
    messages: Any[];
    memo: string;
    timeout_height: string;
    extension_options: Any[];
    non_critical_extension_options: Any[];
  };
  auth_info: {
    signer_infos: Array<{
      public_key: PubKeySecp256k1;
      mode_info: {
        single: {
          mode: number;
        };
      };
      sequence: string;
    }>;
    fee: TxFee;
  };
  signatures: string[];
}

export interface Document {
  chain_id: string;
  account_number: string;
  sequence: string;
  fee: {
    amount: {
      denom: string;
      amount: string;
    }[];
    gas: string;
    granter?: string;
    payer?: string;
  };
  msgs: {
    type: string;
    value: any;
  }[];
  memo: string;
}

export const decodeTxMessages = (messages: Any[]): any[] => {
  return messages.map((m: Any) => {
    // For KNIRV, we assume the value is already JSON-encoded or can be parsed directly
    try {
      const valueStr = new TextDecoder().decode(m.value);
      const parsedValue = JSON.parse(valueStr);

      return {
        '@type': m.type_url,
        ...parsedValue,
      };
    } catch (error) {
      // If parsing fails, return the raw value
      return {
        '@type': m.type_url,
        value: m.value,
      };
    }
  });
};

// KNIRV Package structure
export interface KNIRVMemFile {
  name: string;
  body: string;
}

export interface KNIRVMemPackage {
  name: string;
  path: string;
  files: KNIRVMemFile[];
}

function createMemPackage(memPackage: RawMemPackage): KNIRVMemPackage {
  return {
    name: memPackage.name,
    path: memPackage.path,
    files: memPackage.files.map((file: any) => ({
      name: file.name,
      body: file.body,
    })),
  };
}

function encodeMessageValue(message: { type: string; value: any }): Any {
  // For KNIRV, we encode the message value as JSON
  const jsonValue = JSON.stringify(message.value);
  const encodedValue = new TextEncoder().encode(jsonValue);

  return {
    type_url: message.type,
    value: encodedValue,
  };
}

export function documentToTx(document: Document): Tx {
  const messages: Any[] = document.msgs.map(encodeMessageValue);
  return {
    body: {
      messages,
      memo: document.memo,
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: {
      signer_infos: [],
      fee: {
        amount: document.fee.amount,
        gas: document.fee.gas,
        granter: document.fee.granter,
        payer: document.fee.payer,
      },
    },
    signatures: [],
  };
}

export function documentToDefaultTx(document: Document): Tx {
  const messages: Any[] = document.msgs.map(encodeMessageValue);
  return {
    body: {
      messages,
      memo: document.memo,
      timeout_height: '0',
      extension_options: [],
      non_critical_extension_options: [],
    },
    auth_info: {
      signer_infos: [
        {
          public_key: {
            key: '',
          },
          mode_info: {
            single: {
              mode: 1,
            },
          },
          sequence: '0',
        },
      ],
      fee: {
        amount: document.fee.amount,
        gas: document.fee.gas,
        granter: document.fee.granter,
        payer: document.fee.payer,
      },
    },
    signatures: [''],
  };
}

export function txToDocument(tx: Tx): Document {
  return {
    chain_id: '',
    account_number: '0',
    sequence: '0',
    fee: tx.auth_info.fee,
    msgs: tx.body.messages.map((msg) => ({
      type: msg.type_url,
      value: JSON.parse(new TextDecoder().decode(msg.value)),
    })),
    memo: tx.body.memo,
  };
}

export interface RawBankSendMessage {
  '@type': string;
  from_address: string;
  to_address: string;
  amount: string;
}

export interface RawVmCallMessage {
  '@type': string;
  caller: string;
  func: string;
  send: string;
  pkg_path: string;
  args: string[];
}

export interface RawVmAddPackageMessage {
  '@type': string;
  creator: string;
  deposit: string;
  package: RawMemPackage;
}

export interface RawVmRunMessage {
  '@type': string;
  caller: string;
  send: string;
  package: RawMemPackage;
}

export interface RawMemPackage {
  name: string;
  path: string;
  files: {
    name: string;
    body: string;
  }[];
}

export type RawTxMessageType =
  | RawBankSendMessage
  | RawVmCallMessage
  | RawVmAddPackageMessage
  | RawVmRunMessage;

export interface RawTx {
  msg: RawTxMessageType[];
  fee: { gas_wanted: string; gas_fee: string };
  signatures: {
    pub_key: {
      '@type': string;
      value: string;
    };
    signature: string;
  }[];
  memo: string;
}

/**
 * Change transaction json string to a Signed Tx.
 *
 * @param str
 * @returns Tx | null
 */
export const strToSignedTx = (str: string): Tx | null => {
  let rawTx = null;
  try {
    rawTx = JSON.parse(str);
  } catch (e) {
    console.error(e);
  }

  if (rawTx === null) return null;

  try {
    const document = rawTx as RawTx;
    const messages: Any[] = document.msg
      .map((msg) => ({
        type: msg['@type'],
        value: { ...msg },
      }))
      .map(encodeMessageValue);

    return {
      body: {
        messages,
        memo: document.memo,
        timeout_height: '0',
        extension_options: [],
        non_critical_extension_options: [],
      },
      auth_info: {
        signer_infos: document.signatures.map((signature) => {
          const publicKeyBytes = fromBase64(signature?.pub_key?.value || '');
          return {
            public_key: {
              key: btoa(String.fromCharCode(...publicKeyBytes)),
            },
            mode_info: {
              single: {
                mode: 1,
              },
            },
            sequence: '0',
          };
        }),
        fee: {
          amount: [],
          gas: document.fee.gas_wanted || '0',
          granter: undefined,
          payer: undefined,
        },
      },
      signatures: document.signatures.map((signature) =>
        signature?.signature || ''
      ),
    };
  } catch (e) {
    console.error(e);
    return null;
  }
};
