/** One-time migration of legacy game challenges into backend-owned postings.
 * Requires an admin bearer token; it is safe to re-run because challenge IDs
 * are preserved as risk-class IDs and existing classes are skipped. */
import { CHALLENGES } from '../src/data/challenges.ts';

const baseURL = (process.env.KNIRV_ACTUARIAL_API || 'http://localhost:8082/api/v1/actuarial').replace(/\/$/, '');
const token = process.env.KNIRV_ADMIN_TOKEN;
const network = process.env.KNIRV_NETWORK_ID || 'knirv-testnet-1';
if (!token) throw new Error('KNIRV_ADMIN_TOKEN is required');
const headers = { authorization: `Bearer ${token}`, 'content-type': 'application/json', 'x-knirv-network-id': network };
const existingResponse = await fetch(`${baseURL}/risk-classes`, { headers });
if (!existingResponse.ok) throw new Error(`list risk classes failed: ${existingResponse.status}`);
const existing = new Set((await existingResponse.json()).map(item => item.id));
let created = 0;
for (const challenge of CHALLENGES) {
  if (existing.has(challenge.id)) continue;
  const response = await fetch(`${baseURL}/risk-classes`, { method: 'POST', headers: { ...headers, 'idempotency-key': `arena-challenge-${challenge.id}` }, body: JSON.stringify({
    id: challenge.id, display_name: challenge.title, description: challenge.description, taxonomy_version: 'arena-challenges-v1', domain: 'code_error', difficulty_tier: Math.max(1, Math.round(challenge.difficulty * 10)), limits: { max_payout_per_claim: Math.round(challenge.bounty * 100000000), max_aggregate_liability: Math.round(challenge.bounty * 100000000), max_blast_radius: 1 }, curated_challenge: { legacy_id: challenge.id, type: challenge.type, buggy_code: challenge.buggyCode, context: challenge.context, hints: challenge.hints },
  }) });
  if (!response.ok) throw new Error(`seed ${challenge.id} failed: ${await response.text()}`);
  created++;
}
console.log(`Actuarial challenge migration complete: ${created} created, ${CHALLENGES.length - created} already present.`);
