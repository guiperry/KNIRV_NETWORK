import { Capacitor } from '@capacitor/core';
import { readStorageItem, removeStorageItem, writeStorageItem, getSessionMap } from './browserStorage';

export interface VaultHandoffPayload {
  origin: string;
  message: string;
  createdAt: number;
  autoOpenVault: boolean;
}

const VAULT_HANDOFF_KEY = 'knirv.vault.handoff';

function getInMemoryHandoff(): VaultHandoffPayload | null {
  const raw = getSessionMap().get(VAULT_HANDOFF_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as VaultHandoffPayload;
  } catch { return null; }
}

function setInMemoryHandoff(payload: VaultHandoffPayload) {
  getSessionMap().set(VAULT_HANDOFF_KEY, JSON.stringify(payload));
}

function removeInMemoryHandoff() {
  getSessionMap().delete(VAULT_HANDOFF_KEY);
}

export function saveVaultHandoff(payload: VaultHandoffPayload) {
  if (Capacitor.isNativePlatform()) {
    setInMemoryHandoff(payload);
  } else {
    writeStorageItem(VAULT_HANDOFF_KEY, JSON.stringify(payload), 'session');
  }
}

export function readVaultHandoff() {
  if (Capacitor.isNativePlatform()) {
    return getInMemoryHandoff();
  }
  const raw = readStorageItem(VAULT_HANDOFF_KEY, 'session');
  if (!raw) { return null; }
  try {
    return JSON.parse(raw) as VaultHandoffPayload;
  } catch { return null; }
}

export function clearVaultHandoff() {
  if (Capacitor.isNativePlatform()) {
    removeInMemoryHandoff();
  } else {
    removeStorageItem(VAULT_HANDOFF_KEY, 'session');
  }
}
