import { useCallback, useState } from 'react';
import {
  DEFAULT_NETWORK_ID,
  getNetworkOption,
  getStoredDevServerUrl,
  getStoredNetworkId,
  setStoredNetwork,
  type NetworkId,
} from '@/react-app/platform/networkConfig';

export function useNetworkConfig() {
  const [storedNetworkId, setStoredNetworkId] = useState<NetworkId | null>(() => getStoredNetworkId());
  const [devServerUrl, setDevServerUrl] = useState<string>(() => getStoredDevServerUrl());

  const isConfigured = storedNetworkId !== null;
  const networkId = storedNetworkId ?? DEFAULT_NETWORK_ID;
  const serverUrl = networkId === 'dev' ? devServerUrl : getNetworkOption(networkId).url;

  const selectNetwork = useCallback((id: NetworkId, customDevServerUrl?: string) => {
    const trimmedDevUrl = customDevServerUrl?.trim();
    setStoredNetwork(id, trimmedDevUrl || devServerUrl);
    setStoredNetworkId(id);
    if (id === 'dev' && trimmedDevUrl) {
      setDevServerUrl(trimmedDevUrl);
    }
  }, [devServerUrl]);

  return {
    networkId,
    isConfigured,
    serverUrl,
    devServerUrl,
    selectNetwork,
  };
}
