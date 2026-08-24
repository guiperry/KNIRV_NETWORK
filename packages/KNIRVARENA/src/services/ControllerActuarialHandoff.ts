import QRCode from 'qrcode';
import type { MutationAuthorizer } from './ActuarialSyndicateService';

function baseURL() { return import.meta.env.VITE_KNIRVSERVER_URL ? new URL(import.meta.env.VITE_KNIRVSERVER_URL).origin : window.location.origin; }
function canonical(value: Record<string, unknown>): string {
  const sort = (v: unknown): unknown => Array.isArray(v) ? v.map(sort) : v && typeof v === 'object' ? Object.fromEntries(Object.entries(v as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b)).map(([k, x]) => [k, sort(x)])) : v;
  return JSON.stringify(sort(value));
}
function showQR(uri: string) {
  const dialog = document.createElement('div');
  dialog.style.cssText = 'position:fixed;inset:0;z-index:2147483647;display:grid;place-items:center;background:#000c;color:#fff;padding:24px;text-align:center';
  dialog.innerHTML = '<div style="max-width:360px;background:#172033;padding:24px;border-radius:12px"><h2>Approve in KNIRVCONTROLLER</h2><p>Scan with the mobile Controller. This expires in five minutes.</p><img alt="Controller approval QR code" style="width:280px;height:280px;background:white"/></div>';
  QRCode.toDataURL(uri, { width: 280, margin: 1 }).then(url => { (dialog.querySelector('img') as HTMLImageElement).src = url; });
  document.body.appendChild(dialog); return () => dialog.remove();
}

/** QR/mobile approval; desktop Arena never receives Controller credentials. */
export function controllerActuarialAuthorizer(): MutationAuthorizer {
  return async payload => {
    const relay = payload.__knirv_relay as { path?: string; body?: Record<string, unknown> } | undefined;
    if (!relay?.path || !relay.body) throw new Error('Missing actuarial relay request');
    const clean = { ...payload }; delete clean.__knirv_relay;
    const now = Math.floor(Date.now() / 1000); const network = import.meta.env.VITE_KNIRV_NETWORK_ID || 'testnet';
    const response = await fetch(`${baseURL()}/api/controller/signing/requests`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({
      kind: 'message', chain_id: network,
      envelope: { schemaVersion: 'knirv.message.v1', domain: 'knirv.controller', purpose: 'user-approval', chainId: network, nonce: crypto.randomUUID(), issuedAtUnix: now, expiresAtUnix: now + 300, payload: Array.from(new TextEncoder().encode(canonical(clean))) },
      actuarial: { path: relay.path, body: relay.body, idempotency_key: clean.idempotency_key },
    }) });
    const created = await response.json() as { request_id?: string; approval_uri?: string; error?: string };
    if (!response.ok || !created.request_id || !created.approval_uri) throw new Error(created.error || 'Unable to create Controller approval request');
    const close = showQR(created.approval_uri);
    try {
      for (let i = 0; i < 150; i++) {
        await new Promise(resolve => window.setTimeout(resolve, 2000));
        const status = await (await fetch(`${baseURL()}/api/controller/signing/requests/${encodeURIComponent(created.request_id)}`, { cache: 'no-store' })).json() as { status: string; result?: { actuarial_result?: unknown }; reason?: string };
        if (status.status === 'approved' && status.result?.actuarial_result !== undefined) return { signedIntent: { nonce: '', canonical_payload: '', signature: '' }, headers: {}, result: status.result.actuarial_result };
        if (status.status === 'rejected' || status.status === 'expired') throw new Error(status.reason || `Controller request ${status.status}`);
      }
      throw new Error('Controller approval timed out');
    } finally { close(); }
  };
}
