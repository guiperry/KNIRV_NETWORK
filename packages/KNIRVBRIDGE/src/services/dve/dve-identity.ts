export interface DVEIdentity {
  nodeID: string;
  dveURI: string;
  walletAddress: string;
  extensionID: string;
  browserVersion: string;
}

/**
 * Derive a DVEIdentity from a wallet address.
 * Uses crypto.subtle.digest('SHA-256', ...) available in extension service worker context.
 * nodeID = sha256(walletAddress + extensionID)
 * dveURI = "knirv://dve/{walletAddress}/browser"
 */
export async function deriveDVEIdentity(walletAddress: string): Promise<DVEIdentity> {
  const extensionID = chrome.runtime.id;

  let browserVersion = '';
  try {
    const { navigator: nav } = globalThis;
    browserVersion = nav.userAgent || '';
  } catch {
    browserVersion = 'unknown';
  }

  const nodeID = await generateNodeID(walletAddress, extensionID);
  const dveURI = formatDVEURI(walletAddress);

  return {
    nodeID,
    dveURI,
    walletAddress,
    extensionID,
    browserVersion,
  };
}

/**
 * Format a knirv:// DVE URI from a wallet address.
 */
export function formatDVEURI(walletAddress: string): string {
  return `knirv://dve/${walletAddress}/browser`;
}

/**
 * Generate a deterministic node ID by SHA-256 hashing the wallet address
 * concatenated with the extension ID.
 */
export async function generateNodeID(
  walletAddress: string,
  extensionID: string,
): Promise<string> {
  const data = new TextEncoder().encode(walletAddress + extensionID);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return 'dve-' + hashArray.map((b) => b.toString(16).padStart(2, '0')).join('').substring(0, 16);
}
