import { readStorageItem, writeStorageItem } from '@/react-app/platform/browserStorage';

export type NetworkId = 'mainnet' | 'testnet' | 'dev';

export interface NetworkOption {
  id: NetworkId;
  name: string;
  description: string;
  url: string;
  editable: boolean;
  disabled: boolean;
  badge?: string;
}

export const NETWORK_OPTIONS: NetworkOption[] = [
  {
    id: 'mainnet',
    name: 'Mainnet',
    description: 'The production KNIRV network.',
    url: 'https://gateway.knirv.network',
    editable: false,
    disabled: true,
    badge: 'Coming Soon',
  },
  {
    id: 'testnet',
    name: 'Testnet',
    description: 'The public KNIRV test network.',
    url: 'https://testnet-gateway.knirv.network',
    editable: false,
    disabled: false,
    badge: 'Default',
  },
  {
    id: 'dev',
    name: 'Dev',
    description: 'Connect to a local development server.',
    url: 'http://localhost:8080',
    editable: true,
    disabled: false,
  },
];

export const DEFAULT_NETWORK_ID: NetworkId = 'testnet';

const DEFAULT_DEV_SERVER_URL = NETWORK_OPTIONS.find((option) => option.id === 'dev')!.url;

const STORAGE_KEYS = {
  networkId: 'knirv.settings.networkId',
  devServerUrl: 'knirv.settings.devServerUrl',
} as const;

export function getNetworkOption(id: NetworkId): NetworkOption {
  return NETWORK_OPTIONS.find((option) => option.id === id)!;
}

export function getStoredNetworkId(): NetworkId | null {
  const stored = readStorageItem(STORAGE_KEYS.networkId);
  return stored === 'mainnet' || stored === 'testnet' || stored === 'dev' ? stored : null;
}

export function getStoredDevServerUrl(): string {
  return readStorageItem(STORAGE_KEYS.devServerUrl) || DEFAULT_DEV_SERVER_URL;
}

export function setStoredNetwork(id: NetworkId, devServerUrl?: string): void {
  writeStorageItem(STORAGE_KEYS.networkId, id);
  if (id === 'dev' && devServerUrl) {
    writeStorageItem(STORAGE_KEYS.devServerUrl, devServerUrl);
  }
}

export function getActiveServerUrl(): string {
  const id = getStoredNetworkId() ?? DEFAULT_NETWORK_ID;
  return id === 'dev' ? getStoredDevServerUrl() : getNetworkOption(id).url;
}
