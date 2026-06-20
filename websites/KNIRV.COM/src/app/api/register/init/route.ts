import { NextResponse } from 'next/server';
import crypto from 'crypto';

// In-memory store: nonce → { sessionId, createdAt, completed, address, role }
// In production this should be Redis or a DB with TTLs.
const sessions = new Map<string, {
  sessionId: string;
  nonce: string;
  createdAt: number;
  completed: boolean;
  address?: string;
  pubkey?: string;
  role?: string;
  token?: string;
}>();

// Expose sessions map for the verify route (same process, same module cache).
export { sessions };

export async function POST() {
  const nonce = crypto.randomBytes(24).toString('hex');
  const sessionId = crypto.randomBytes(16).toString('hex');

  sessions.set(nonce, {
    sessionId,
    nonce,
    createdAt: Date.now(),
    completed: false,
  });

  // Clean sessions older than 10 minutes
  const cutoff = Date.now() - 10 * 60 * 1000;
  for (const [k, v] of sessions) {
    if (v.createdAt < cutoff) sessions.delete(k);
  }

  return NextResponse.json({ nonce, sessionId });
}
