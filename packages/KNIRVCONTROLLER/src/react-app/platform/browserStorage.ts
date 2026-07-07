import { Capacitor } from '@capacitor/core';
import { nativeStorage } from './nativeStorage';

export type StorageScope = 'local' | 'session';

function getStorage(scope: StorageScope): Storage | null {
  if (typeof window === 'undefined') { return null; }
  try {
    return scope === 'local' ? window.localStorage : window.sessionStorage;
  } catch { return null; }
}

const sessionMap = new Map<string, string>();

export function getSessionMap(): Map<string, string> {
  return sessionMap;
}

export function readStorageItem(key: string, scope: StorageScope = 'local') {
  return getStorage(scope)?.getItem(key) ?? null;
}

export function writeStorageItem(key: string, value: string, scope: StorageScope = 'local') {
  getStorage(scope)?.setItem(key, value);
}

export function removeStorageItem(key: string, scope: StorageScope = 'local') {
  getStorage(scope)?.removeItem(key);
}

export function readBooleanStorage(key: string, defaultValue: boolean, scope: StorageScope = 'local') {
  const stored = readStorageItem(key, scope);
  return stored === null ? defaultValue : stored === 'true';
}

export async function readStorageItemAsync(key: string, scope: StorageScope = 'local', useNative = Capacitor.isNativePlatform()): Promise<string | null> {
  if (useNative) {
    if (scope === 'session') return sessionMap.get(key) ?? null;
    return nativeStorage.get(key);
  }
  return readStorageItem(key, scope);
}

export async function writeStorageItemAsync(key: string, value: string, scope: StorageScope = 'local', useNative = Capacitor.isNativePlatform()): Promise<void> {
  if (useNative) {
    if (scope === 'session') { sessionMap.set(key, value); return; }
    await nativeStorage.set(key, value);
  } else {
    writeStorageItem(key, value, scope);
  }
}

export async function removeStorageItemAsync(key: string, scope: StorageScope = 'local', useNative = Capacitor.isNativePlatform()): Promise<void> {
  if (useNative) {
    if (scope === 'session') { sessionMap.delete(key); return; }
    await nativeStorage.remove(key);
  } else {
    removeStorageItem(key, scope);
  }
}

export function canUseBrowserApis() {
  return typeof window !== 'undefined' && typeof navigator !== 'undefined';
}

export function hasSecureContext() {
  return canUseBrowserApis() && window.isSecureContext;
}

export function hasCameraSupport() {
  return canUseBrowserApis() && !!navigator.mediaDevices?.getUserMedia;
}
