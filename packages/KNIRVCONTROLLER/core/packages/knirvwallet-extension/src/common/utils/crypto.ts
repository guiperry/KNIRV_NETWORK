import CryptoJS from 'crypto-js';

export const encryptAES = async (value: string, password: string) => {
  return CryptoJS.AES.encrypt(value, password).toString();
};

export const decryptAES = async (encryptedValue: string, password: string) => {
  const bytes = CryptoJS.AES.decrypt(encryptedValue, password);
  return bytes.toString(CryptoJS.enc.Utf8);
};
