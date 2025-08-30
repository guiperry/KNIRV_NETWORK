// Mock implementation for @knirvsdk/crypto
export const encryptAES = jest.fn((data: string, key: string) => {
  return `encrypted_${data}_with_${key}`;
});

export const decryptAES = jest.fn((encryptedData: string, key: string) => {
  return encryptedData.replace(`encrypted_`, '').replace(`_with_${key}`, '');
});

export const makeCryptKey = jest.fn((password: string, salt?: string) => {
  return `key_${password}_${salt || 'default_salt'}`;
});

export const sha256 = jest.fn((data: string) => {
  return `sha256_${data}`;
});

export default {
  encryptAES,
  decryptAES,
  makeCryptKey,
  sha256
};
