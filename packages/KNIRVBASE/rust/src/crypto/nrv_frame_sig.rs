//! `.nrv` frame signatures: ML-DSA-65 (FIPS 204, pure, empty context).
//!
//! The signed bytes are byte-for-byte the ones the Go implementation signs,
//! `FrameSigningMessage` in `KNIRVBASE/go/pkg/nrv/signature.go`:
//!   - a frame without a payload section is signed over its bracket binary alone
//!     (the legacy form, KNIRVBASE_Upgrade.md §3.3);
//!   - a frame with a payload section is signed over
//!     `"KNIRV-NRV-FRAME-SIG/v1\0" || u64le(len(brackets)) || brackets || u64le(len(payload)) || payload`.
//!
//! The ML-DSA-65 operations themselves come from the KNIRVSDK Rust core
//! (`knirv_sdk::mldsa`); nothing here re-implements the algorithm.

use knirv_sdk::mldsa::{self, MlDsaFailure};

/// PQCManifest algorithm name for frame signatures. Matches Go's
/// `SignatureAlgorithmMLDSA65`.
pub const SIGNATURE_ALGORITHM: &str = "ML-DSA-65";

/// Domain tag for the framed signing message. Matches Go's `frameSigDomain`.
const FRAME_SIG_DOMAIN: &[u8] = b"KNIRV-NRV-FRAME-SIG/v1\0";

/// The exact bytes a frame signature covers. `payload` is `None` when the frame
/// has no payload section; `Some` (even empty) selects the framed form.
pub fn frame_signing_message(brackets: &[u8], payload: Option<&[u8]>) -> Vec<u8> {
    let payload = match payload {
        None => return brackets.to_vec(),
        Some(p) => p,
    };
    let mut out = Vec::with_capacity(FRAME_SIG_DOMAIN.len() + 16 + brackets.len() + payload.len());
    out.extend_from_slice(FRAME_SIG_DOMAIN);
    out.extend_from_slice(&(brackets.len() as u64).to_le_bytes());
    out.extend_from_slice(brackets);
    out.extend_from_slice(&(payload.len() as u64).to_le_bytes());
    out.extend_from_slice(payload);
    out
}

/// Signs a frame with the ML-DSA-65 key derived from a 32-byte seed.
/// Deterministic: the same seed and frame give the same 3309 bytes.
pub fn sign_frame(
    seed: &[u8],
    brackets: &[u8],
    payload: Option<&[u8]>,
) -> Result<Vec<u8>, MlDsaFailure> {
    mldsa::ml_dsa65_sign(seed, &frame_signing_message(brackets, payload), &[])
}

/// Verifies a frame signature. Returns `false` for any malformed input or failed check.
pub fn verify_frame(
    public_key: &[u8],
    brackets: &[u8],
    payload: Option<&[u8]>,
    signature: &[u8],
) -> bool {
    mldsa::ml_dsa65_verify(
        public_key,
        &frame_signing_message(brackets, payload),
        &[],
        signature,
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{engine::general_purpose::STANDARD as B64, Engine as _};

    const SEED: [u8; 32] = [0x5a; 32];

    fn pk() -> Vec<u8> {
        mldsa::ml_dsa65_public_key(&SEED).unwrap()
    }

    #[test]
    fn legacy_form_is_brackets_only() {
        assert_eq!(frame_signing_message(b"abc", None), b"abc".to_vec());
    }

    #[test]
    fn framed_layout_matches_go_encoding() {
        let msg = frame_signing_message(b"BR", Some(b"PAY"));
        let mut want = Vec::new();
        want.extend_from_slice(b"KNIRV-NRV-FRAME-SIG/v1\0");
        want.extend_from_slice(&2u64.to_le_bytes());
        want.extend_from_slice(b"BR");
        want.extend_from_slice(&3u64.to_le_bytes());
        want.extend_from_slice(b"PAY");
        assert_eq!(msg, want);
    }

    #[test]
    fn empty_payload_is_still_framed() {
        assert_ne!(
            frame_signing_message(b"BR", Some(b"")),
            frame_signing_message(b"BR", None)
        );
    }

    #[test]
    fn sign_verify_roundtrip_and_mutants() {
        let sig = sign_frame(&SEED, b"brackets", Some(b"payload")).unwrap();
        assert_eq!(sig.len(), mldsa::SIGNATURE_LEN);
        let pk = pk();
        assert!(verify_frame(&pk, b"brackets", Some(b"payload"), &sig));
        // tampered payload, tampered brackets, legacy form, wrong key, truncated signature
        assert!(!verify_frame(&pk, b"brackets", Some(b"payloaD"), &sig));
        assert!(!verify_frame(&pk, b"brackes", Some(b"payload"), &sig));
        assert!(!verify_frame(&pk, b"brackets", None, &sig));
        let other = mldsa::ml_dsa65_public_key(&[0x01; 32]).unwrap();
        assert!(!verify_frame(&other, b"brackets", Some(b"payload"), &sig));
        assert!(!verify_frame(
            &pk,
            b"brackets",
            Some(b"payload"),
            &sig[..sig.len() - 1]
        ));
    }

    #[test]
    fn go_signed_vector_verifies_in_rust() {
        let v: serde_json::Value =
            serde_json::from_str(include_str!("../../testdata/mldsa65/go_frame_vector.json"))
                .unwrap();
        let dec = |k: &str| B64.decode(v[k].as_str().unwrap()).unwrap();
        let (pk, brackets, payload, sig) = (
            dec("public_key"),
            dec("brackets"),
            dec("payload"),
            dec("signature"),
        );
        assert!(verify_frame(&pk, &brackets, Some(&payload), &sig));
        assert!(!verify_frame(&pk, &brackets, Some(&payload[1..]), &sig));
    }
}
