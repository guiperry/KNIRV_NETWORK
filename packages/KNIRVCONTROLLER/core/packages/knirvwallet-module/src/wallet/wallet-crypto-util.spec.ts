import {
  decryptAES,
  encryptAES,
  encryptSha256,
  executeKdf,
  makeCryptKey,
} from './wallet-crypto-util';
import { toHex } from '../encoding';

describe('execute kdf', () => {
  it('success', async () => {
    const salt = 'TESTTESTTESTTEST';
    const password = 'PASSWORD';
    const kdfConfiguration = {
      algorithm: 'argon2id',
      params: {
        outputLength: 32,
        opsLimit: 24,
        memLimitKib: 12 * 1024,
      },
    };
    const result = await executeKdf(salt, password, kdfConfiguration);
    const hexResult = toHex(result);

    expect(hexResult).toBe('74765557f5a3b6296852ea1a159eaa3e638022bbc4e507b44a69b014f3f8b36b');
  });
});

describe('make cryptkey with kdf', () => {
  it('success', async () => {
    const password = 'PASSWORD';
    const result = await makeCryptKey(password);

    expect(result).toBe('74765557f5a3b6296852ea1a159eaa3e638022bbc4e507b44a69b014f3f8b36b');
  });
});

describe('encrypt SHA256', () => {
  it('success', async () => {
    const password = 'PASSWORD';
    const result = await encryptSha256(password);

    expect(result).toBe('06b96931254927a2cd9a12c366fbd0d7717a92ec7fc41515044b1969fd9f2330');
  });
});

describe('encrypt/decrypt AES', () => {
  it('encrypt success', async () => {
    const value = 'CURRENT_VALUE';
    const password = 'PASSWORD';
    const result = await encryptAES(value, password);

    expect(result).toBeTruthy();
  });

  it('decrypt success', async () => {
    const value = 'CURRENT_VALUE';
    const password = 'PASSWORD';
    const encryptedValue = await encryptAES(value, password);
    const result = await decryptAES(encryptedValue, password);

    expect(result).toBe(value);
  });

  it('encrypt with decrypt success', async () => {
    const value = 'CURRENT_VALUE';
    const password = 'PASSWORD';
    const encryptedValue = await encryptAES(value, password);
    const result = await decryptAES(encryptedValue, password);

    expect(result).toBe('CURRENT_VALUE');
  });
});
