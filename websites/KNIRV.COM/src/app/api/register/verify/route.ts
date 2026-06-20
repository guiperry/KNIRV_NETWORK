import { NextRequest, NextResponse } from 'next/server';
import { sessions } from '../init/route';

// GET /api/register/verify?sessionId=<id>  — polling endpoint for the page
// POST /api/register/verify                — callback from KNIRVCONTROLLER after QR scan

export async function GET(req: NextRequest) {
  const sessionId = req.nextUrl.searchParams.get('sessionId');
  if (!sessionId) {
    return NextResponse.json({ error: 'Missing sessionId' }, { status: 400 });
  }

  for (const session of sessions.values()) {
    if (session.sessionId === sessionId) {
      if (session.completed) {
        return NextResponse.json({
          completed: true,
          address: session.address,
          role: session.role ?? 'General',
          token: session.token,
        });
      }
      return NextResponse.json({ completed: false });
    }
  }

  return NextResponse.json({ error: 'Session not found' }, { status: 404 });
}

export async function POST(req: NextRequest) {
  let body: { nonce?: string; signature?: string; pubkey?: string; address?: string; role?: string };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: 'Invalid JSON' }, { status: 400 });
  }

  const { nonce, signature, pubkey, address, role } = body;
  if (!nonce || !signature || !pubkey || !address) {
    return NextResponse.json({ error: 'Missing fields' }, { status: 400 });
  }

  const session = sessions.get(nonce);
  if (!session) {
    return NextResponse.json({ error: 'Invalid or expired nonce' }, { status: 404 });
  }
  if (session.completed) {
    return NextResponse.json({ error: 'Nonce already used' }, { status: 409 });
  }

  // TODO: verify signature against pubkey and nonce using the KNIRV crypto primitives.
  // For now we accept any well-formed submission and flag it complete.

  const token = Buffer.from(`${address}:${Date.now()}`).toString('base64');

  session.completed = true;
  session.address = address;
  session.pubkey = pubkey;
  session.role = role ?? 'General';
  session.token = token;

  return NextResponse.json({ ok: true, token, role: session.role });
}
