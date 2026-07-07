import { Capacitor } from '@capacitor/core';
import { BarcodeScanner } from 'capacitor-barcode-scanner';

export async function scanNativeQrCode() {
  if (!Capacitor.isNativePlatform()) {
    throw new Error('Native QR scanning is only available inside the Android or iOS shell.');
  }

  const result = await BarcodeScanner.scan();
  if (!result.result || !result.code?.trim()) {
    throw new Error('No QR payload detected.');
  }

  return result.code.trim();
}
