import { DVEIdentity } from './dve-identity';
import { DVE_CONSTANTS } from '@common/constants/dve.constant';

export interface RegisterDVEPayload {
  wallet_address: string;
  tee_type: 'browser-extension';
  extension_id: string;
  browser_version: string;
  capabilities: string[];
  badge_nft_ids: string[];
  is_remote: boolean;
  node_id: string;
  dve_uri: string;
}

export interface DVENodeResponse {
  id: string;
  node_id: string;
  status: 'active' | 'inactive' | 'suspended';
  wallet_address: string;
  dve_uri: string;
  capabilities: string[];
  badge_nft_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface DVEHeartbeatPayload {
  node_id: string;
  timestamp: number;
  capabilities: string[];
  badge_nft_ids: string[];
}

export class DVERegistryService {
  private serverURL: string;
  private authToken: string;

  constructor(serverURL: string, authToken: string) {
    this.serverURL = serverURL.replace(/\/+$/, '');
    this.authToken = authToken;
  }

  /**
   * Register a new DVE node with the server.
   * POST /api/v1/dve/nodes
   * Returns the assigned server-side node ID.
   */
  async register(
    identity: DVEIdentity,
    capabilities: string[],
    badgeNFTIDs: string[],
  ): Promise<string> {
    const payload: RegisterDVEPayload = {
      wallet_address: identity.walletAddress,
      tee_type: 'browser-extension',
      extension_id: identity.extensionID,
      browser_version: identity.browserVersion,
      capabilities,
      badge_nft_ids: badgeNFTIDs,
      is_remote: false,
      node_id: identity.nodeID,
      dve_uri: identity.dveURI,
    };

    const response = await fetch(`${this.serverURL}/api/v1/dve/nodes`, {
      method: 'POST',
      headers: this.buildHeaders(),
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new Error(`DVE registration failed (${response.status}): ${errorText}`);
    }

    const data: DVENodeResponse = await response.json();
    return data.id;
  }

  /**
   * Send a heartbeat to keep the node alive.
   * PUT /api/v1/dve/nodes/{id}/heartbeat
   */
  async heartbeat(nodeID: string): Promise<void> {
    const payload: DVEHeartbeatPayload = {
      node_id: nodeID,
      timestamp: Date.now(),
      capabilities: [],
      badge_nft_ids: [],
    };

    const response = await fetch(
      `${this.serverURL}/api/v1/dve/nodes/${encodeURIComponent(nodeID)}/heartbeat`,
      {
        method: 'PUT',
        headers: this.buildHeaders(),
        body: JSON.stringify(payload),
      },
    );

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new Error(`DVE heartbeat failed (${response.status}): ${errorText}`);
    }
  }

  /**
   * Deregister a DVE node from the server.
   * PUT /api/v1/dve/nodes/{id}/status
   */
  async deregister(nodeID: string): Promise<void> {
    const response = await fetch(
      `${this.serverURL}/api/v1/dve/nodes/${encodeURIComponent(nodeID)}/status`,
      {
        method: 'PUT',
        headers: this.buildHeaders(),
        body: JSON.stringify({ node_id: nodeID, status: 'inactive' }),
      },
    );

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new Error(`DVE deregistration failed (${response.status}): ${errorText}`);
    }
  }

  /**
   * Sync the capabilities of an existing node.
   * PUT /api/v1/dve/nodes/{id}/status
   */
  async syncCapabilities(nodeID: string, capabilities: string[]): Promise<void> {
    const response = await fetch(
      `${this.serverURL}/api/v1/dve/nodes/${encodeURIComponent(nodeID)}/status`,
      {
        method: 'PUT',
        headers: this.buildHeaders(),
        body: JSON.stringify({ node_id: nodeID, capabilities }),
      },
    );

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new Error(`DVE capability sync failed (${response.status}): ${errorText}`);
    }
  }

  /**
   * Sync the badge NFT IDs for an existing node.
   * PUT /api/v1/dve/nodes/{id}/status
   */
  async syncBadges(nodeID: string, badgeNFTIDs: string[]): Promise<void> {
    const response = await fetch(
      `${this.serverURL}/api/v1/dve/nodes/${encodeURIComponent(nodeID)}/status`,
      {
        method: 'PUT',
        headers: this.buildHeaders(),
        body: JSON.stringify({ node_id: nodeID, badge_nft_ids: badgeNFTIDs }),
      },
    );

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error');
      throw new Error(`DVE badge sync failed (${response.status}): ${errorText}`);
    }
  }

  private buildHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }
    return headers;
  }
}
