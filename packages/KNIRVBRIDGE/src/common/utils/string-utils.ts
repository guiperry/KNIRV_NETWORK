import { formatAddress } from './client-utils';

// Type-safe Bech32 implementation
interface Bech32Result {
  prefix: string;
  data: Uint8Array;
}

interface Bech32Decoded {
  prefix: string;
  words: number[];
}

/**
 * Type-safe Bech32 decoder implementation
 * Replaces adena-module's fromBech32 function
 */
export function fromBech32(address: string): Bech32Result {
  if (!address || typeof address !== 'string') {
    throw new Error('Invalid address format');
  }

  // Bech32 character set
  const CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
  const GENERATOR = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

  // Remove the '1' separator and split prefix from data
  const parts = address.split('1');
  if (parts.length !== 2) {
    throw new Error('Invalid Bech32 address format');
  }

  const prefix = parts[0].toLowerCase();
  const encoded = parts[1].toLowerCase();

  // Verify checksum
  if (!verifyChecksum(prefix, encoded)) {
    throw new Error('Invalid Bech32 checksum');
  }

  // Decode the data part
  const data = new Uint8Array(encoded.length * 5 / 8);
  let bits = 0;
  let index = 0;

  for (let i = 0; i < encoded.length; i++) {
    const value = CHARSET.indexOf(encoded[i]);
    if (value === -1) {
      throw new Error('Invalid Bech32 character');
    }

    bits = (bits << 5) | value;
    if (i % 8 === 7) {
      data[index++] = bits >> 4;
      bits = 0;
    }
  }

  // Handle remaining bits
  if (bits > 0) {
    data[index] = bits >> 4;
  }

  return { prefix, data };
}

/**
 * Verify Bech32 checksum
 */
function verifyChecksum(prefix: string, encoded: string): boolean {
  const CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
  const GENERATOR = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

  let checksum = 0;
  const combined = prefix + '1' + encoded;

  for (let i = 0; i < combined.length; i++) {
    const value = CHARSET.indexOf(combined[i]);
    if (value === -1) return false;

    const top = checksum >> 25;
    checksum = ((checksum & 0x1ffffff) << 5) ^ value;

    for (let j = 0; j < 5; j++) {
      if ((top >> j) & 1) {
        checksum ^= GENERATOR[j];
      }
    }
  }

  return checksum === 0;
}

export const convertTextToAmount = (text: string): { value: string; denom: string } | null => {
  try {
    const balance = text
      .trim()
       
      .replace('"', '')
      .match(/[a-zA-Z]+|[0-9]+(?:\.[0-9]+|)/g);

    if (!balance || balance.length < 2) {
      throw new Error('Parse error');
    }

    const value = balance.length > 0 ? balance[0] : '0';
    const denom = balance.length > 1 ? balance[1] : '';
    return { value, denom };
  } catch {
    return null;
  }
};

export const makeQueryString = (parameters: { [key in string]: string }): string => {
  return Object.entries(parameters)
    .map((entry) => `${entry[0]}=${entry[1]}`)
    .join('&');
};

export const makeDisplayPackagePath = (packagePath: string): string => {
  const items = packagePath.split('/');
  return items.map((item) => (isBech32Address(item) ? formatAddress(item, 4) : item)).join('/');
};

export const isBech32Address = (str: string): boolean => {
  try {
    const { prefix } = fromBech32(str);
    return !!prefix;
  } catch {
    return false;
  }
};

export function calculateByteSize(str: string): number {
  const encoder = new TextEncoder();
  const encodedStr = encoder.encode(str);
  return encodedStr.length;
}

export function reverseString(str: string): string {
  return str.split('').reverse().join('');
}
