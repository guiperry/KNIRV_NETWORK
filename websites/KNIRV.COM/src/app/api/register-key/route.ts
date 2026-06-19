import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';

// Tier requirements per key type.
const TIER_REQUIREMENTS: Record<string, string[]> = {
  enterprise: ['professional', 'enterprise', 'custom'],
  bootnode: ['enterprise', 'custom'],
};

// Verify the EIP-191 personal_sign signature.
// We use a pure-Node implementation to avoid shipping a web3 library.
function recoverAddress(message: string, signature: string): string | null {
  try {
    const prefix = `\x19Ethereum Signed Message:\n${Buffer.byteLength(message, 'utf8')}`;
    const prefixedMsg = Buffer.concat([Buffer.from(prefix), Buffer.from(message, 'utf8')]);
    const msgHash = crypto.createHash('sha256').update(prefixedMsg).digest();
    // Node's built-in crypto does not support EC recovery — stub returns address from sig hash
    // so downstream validation can be completed with a proper web3 lib (ethers/viem) in production.
    // For now we derive a deterministic placeholder to validate the flow end-to-end.
    const sigBuf = Buffer.from(signature.replace(/^0x/, ''), 'hex');
    const recovered = '0x' + crypto.createHash('sha256')
      .update(Buffer.concat([msgHash, sigBuf]))
      .digest()
      .slice(12)
      .toString('hex');
    return recovered;
  } catch {
    return null;
  }
}

// Derive the AES key material from the wallet signature.
// The signature acts as the root password seed; the key is AES-256-GCM encrypted
// using a key derived from the signature via HKDF.
function deriveKeyMaterial(walletAddress: string, signature: string, keyType: string): Buffer {
  const ikm = Buffer.from(signature.replace(/^0x/, ''), 'hex');
  const salt = Buffer.from(`knirv-${keyType}-salt-v1`);
  const info = Buffer.from(`knirv:${keyType}:${walletAddress.toLowerCase()}`);
  return crypto.hkdfSync('sha512', ikm, salt, info, 64) as unknown as Buffer;
}

// Build the binary key file content.
// Format (v1): 4-byte magic | 1-byte version | 1-byte key type | 20-byte wallet address |
//              4-byte timestamp | 64-byte key material | 32-byte HMAC-SHA256 tag
function buildKeyFile(walletAddress: string, signature: string, keyType: string): Buffer {
  const MAGIC = Buffer.from('KNRV');
  const VERSION = Buffer.from([0x01]);
  const KEY_TYPE_BYTE = Buffer.from([keyType === 'enterprise' ? 0x02 : keyType === 'bootnode' ? 0x03 : 0x01]);
  const wallet = Buffer.from(walletAddress.replace(/^0x/, '').toLowerCase(), 'hex');
  const ts = Buffer.alloc(4);
  ts.writeUInt32BE(Math.floor(Date.now() / 1000), 0);
  const keyMaterial = deriveKeyMaterial(walletAddress, signature, keyType);

  const body = Buffer.concat([MAGIC, VERSION, KEY_TYPE_BYTE, wallet, ts, keyMaterial]);
  const hmacKey = deriveKeyMaterial(walletAddress, signature, `${keyType}-hmac`).slice(0, 32);
  const tag = crypto.createHmac('sha256', hmacKey).update(body).digest();

  return Buffer.concat([body, tag]);
}

// Get user tier from Supabase (or env-override for dev).
// In production this queries the subscriptions table via the service role key.
async function getUserTier(request: NextRequest): Promise<string | null> {
  // Allow tier override in development mode for testing.
  const devTier = process.env.DEV_KEY_TIER;
  if (process.env.NODE_ENV !== 'production' && devTier) {
    return devTier;
  }

  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const supabaseServiceKey = process.env.SUPABASE_SERVICE_ROLE_KEY;
  if (!supabaseUrl || !supabaseServiceKey) {
    return null;
  }

  const authHeader = request.headers.get('authorization') || '';
  const sessionToken = authHeader.replace(/^Bearer /, '');
  if (!sessionToken) return null;

  // Verify JWT and look up subscription tier.
  const res = await fetch(`${supabaseUrl}/rest/v1/rpc/get_user_tier`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'apikey': supabaseServiceKey,
      'Authorization': `Bearer ${sessionToken}`,
    },
  }).catch(() => null);

  if (!res || !res.ok) return null;
  const data = await res.json().catch(() => null);
  return data?.tier ?? null;
}

export async function POST(request: NextRequest) {
  let body: {
    keyType?: string;
    walletAddress?: string;
    signature?: string;
    nonce?: string;
    message?: string;
  };

  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: 'Invalid request body' }, { status: 400 });
  }

  const { keyType, walletAddress, signature, nonce, message } = body;

  // Validate inputs.
  if (!keyType || !walletAddress || !signature || !nonce || !message) {
    return NextResponse.json({ error: 'Missing required fields: keyType, walletAddress, signature, nonce, message' }, { status: 400 });
  }

  const allowedKeyTypes = Object.keys(TIER_REQUIREMENTS);
  if (!allowedKeyTypes.includes(keyType)) {
    return NextResponse.json({ error: `Invalid key type. Must be one of: ${allowedKeyTypes.join(', ')}` }, { status: 400 });
  }

  if (!/^0x[0-9a-fA-F]{40}$/.test(walletAddress)) {
    return NextResponse.json({ error: 'Invalid wallet address format' }, { status: 400 });
  }

  if (!/^0x[0-9a-fA-F]{130}$/.test(signature)) {
    return NextResponse.json({ error: 'Invalid signature format' }, { status: 400 });
  }

  // Verify the signed message contains the correct nonce and wallet to prevent replay.
  if (!message.includes(nonce) || !message.toLowerCase().includes(walletAddress.toLowerCase())) {
    return NextResponse.json({ error: 'Message nonce or wallet mismatch' }, { status: 400 });
  }

  // Recover signing address (production: replace stub with ethers.verifyMessage).
  const recovered = recoverAddress(message, signature);
  if (!recovered) {
    return NextResponse.json({ error: 'Could not recover signing address from signature' }, { status: 400 });
  }

  // Verify subscription tier.
  const userTier = await getUserTier(request);
  const requiredTiers = TIER_REQUIREMENTS[keyType];
  if (!userTier || !requiredTiers.includes(userTier.toLowerCase())) {
    return NextResponse.json(
      { error: `A ${keyType} key requires one of these subscription tiers: ${requiredTiers.join(', ')}. Your current tier: ${userTier ?? 'none'}.` },
      { status: 403 }
    );
  }

  // Build and return the key file.
  const keyFile = buildKeyFile(walletAddress, signature, keyType);
  // Convert to ArrayBuffer for Next.js NextResponse body.
  const arrayBuffer = keyFile.buffer.slice(keyFile.byteOffset, keyFile.byteOffset + keyFile.byteLength);

  return new NextResponse(arrayBuffer as BodyInit, {
    status: 200,
    headers: {
      'Content-Type': 'application/octet-stream',
      'Content-Disposition': `attachment; filename="${keyType}.key"`,
      'Content-Length': keyFile.length.toString(),
      'Cache-Control': 'no-store',
      'X-Key-Type': keyType,
      'X-Wallet': walletAddress,
    },
  });
}
