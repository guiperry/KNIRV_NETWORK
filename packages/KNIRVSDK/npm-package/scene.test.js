import { PURPOSE_SCENE_MANIFEST, SCENES_DOMAIN, verifyMessage } from './signing.js';
function fail(message) {
    throw new Error(message);
}
/** Verifies the shared knirv.scenes/scene-manifest golden vector. */
export async function runSceneVectors(vectors) {
    const envelope = vectors.envelope;
    if (envelope.domain !== SCENES_DOMAIN)
        fail(`domain = ${envelope.domain}`);
    if (envelope.purpose !== PURPOSE_SCENE_MANIFEST)
        fail(`purpose = ${envelope.purpose}`);
    const signed = {
        envelope: vectors.signed.envelope_base64,
        signature: vectors.signed.signature_base64,
        public_key: vectors.signed.public_key_base64,
        address: vectors.signed.address,
    };
    const at = new Date(vectors.verify_at_unix * 1000);
    await verifyMessage(signed, SCENES_DOMAIN, PURPOSE_SCENE_MANIFEST, envelope.chain_id, envelope.nonce, at);
    const expired = new Date((envelope.expires_at_unix + 1) * 1000);
    let rejected = false;
    try {
        await verifyMessage(signed, SCENES_DOMAIN, PURPOSE_SCENE_MANIFEST, envelope.chain_id, envelope.nonce, expired);
    }
    catch {
        rejected = true;
    }
    if (!rejected)
        fail('expired scene manifest verified');
    const bytes = Uint8Array.from(atob(signed.envelope), (c) => c.charCodeAt(0));
    if (bytes.length === 0)
        fail('empty envelope');
}
