//! ML-DSA-65 (FIPS 204) verification, key generation from a seed, and signing.
//!
//! The Rust core is the source of truth for ML-DSA-65; the Go and TypeScript
//! bindings call these functions through the `mldsa.*` operations and hold no
//! cryptography of their own. The algorithm is the pure-Rust `ml-dsa` crate,
//! pinned exactly in `Cargo.toml` (currently 0.1.2). Only the pure (non-prehash)
//! external interface is exposed: `ctx` is the FIPS 204 context string, at most
//! 255 bytes.
//!
//! Every length is checked against FIPS 204 Table 2 before the crate sees the
//! bytes, so malformed input is a `false` (or an error), never a panic.

use ml_dsa::{
    EncodedSignature, EncodedVerifyingKey, Keypair, MlDsa65, Seed, Signature, SigningKey,
};

/// ML-DSA-65 public key length in bytes (FIPS 204 Table 2).
pub const PUBLIC_KEY_LEN: usize = 1952;
/// ML-DSA-65 signature length in bytes (FIPS 204 Table 2).
pub const SIGNATURE_LEN: usize = 3309;
/// ML-DSA key generation seed length in bytes (FIPS 204 `xi`).
pub const SEED_LEN: usize = 32;
/// Largest context string FIPS 204 allows.
pub const MAX_CONTEXT_LEN: usize = 255;

/// Why an ML-DSA-65 input was refused. `code` is the stable wire name the
/// bindings return.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MlDsaFailure {
    PublicKeyLength,
    SignatureLength,
    SeedLength,
    ContextLength,
    SignatureInvalid,
    SignatureMalformed,
}

impl MlDsaFailure {
    pub fn code(self) -> &'static str {
        match self {
            Self::PublicKeyLength => "public_key_length",
            Self::SignatureLength => "signature_length",
            Self::SeedLength => "seed_length",
            Self::ContextLength => "context_length",
            Self::SignatureInvalid => "signature_invalid",
            Self::SignatureMalformed => "signature_malformed",
        }
    }

    pub fn message(self) -> &'static str {
        match self {
            Self::PublicKeyLength => "ml-dsa-65 public key must be 1952 bytes",
            Self::SignatureLength => "ml-dsa-65 signature must be 3309 bytes",
            Self::SeedLength => "ml-dsa-65 seed must be 32 bytes",
            Self::ContextLength => "ml-dsa context must be at most 255 bytes",
            Self::SignatureInvalid => "signature verification failed",
            Self::SignatureMalformed => "malformed ml-dsa-65 signature encoding",
        }
    }
}

fn check_context(ctx: &[u8]) -> Result<(), MlDsaFailure> {
    if ctx.len() > MAX_CONTEXT_LEN {
        Err(MlDsaFailure::ContextLength)
    } else {
        Ok(())
    }
}

fn parse_seed(seed: &[u8]) -> Result<Seed, MlDsaFailure> {
    let bytes: [u8; SEED_LEN] = seed.try_into().map_err(|_| MlDsaFailure::SeedLength)?;
    Ok(Seed::from(bytes))
}

/// Verifies an ML-DSA-65 signature and reports why it was refused. The
/// verification rule is FIPS 204 Algorithm 3 (`ML-DSA.Verify`, pure, with `ctx`).
pub fn ml_dsa65_verify_detail(
    public_key: &[u8],
    message: &[u8],
    ctx: &[u8],
    signature: &[u8],
) -> Result<(), MlDsaFailure> {
    let pk_bytes: &[u8; PUBLIC_KEY_LEN] = public_key
        .try_into()
        .map_err(|_| MlDsaFailure::PublicKeyLength)?;
    let sig_bytes: &[u8; SIGNATURE_LEN] = signature
        .try_into()
        .map_err(|_| MlDsaFailure::SignatureLength)?;
    check_context(ctx)?;
    let pk =
        ml_dsa::VerifyingKey::<MlDsa65>::decode(&EncodedVerifyingKey::<MlDsa65>::from(*pk_bytes));
    let enc = EncodedSignature::<MlDsa65>::from(*sig_bytes);
    let sig = Signature::<MlDsa65>::decode(&enc).ok_or(MlDsaFailure::SignatureMalformed)?;
    if pk.verify_with_context(message, ctx, &sig) {
        Ok(())
    } else {
        Err(MlDsaFailure::SignatureInvalid)
    }
}

/// Verifies an ML-DSA-65 signature. Returns `false` for any wrong length, any
/// context longer than 255 bytes, and any failed verification. Never panics.
pub fn ml_dsa65_verify(public_key: &[u8], message: &[u8], ctx: &[u8], signature: &[u8]) -> bool {
    ml_dsa65_verify_detail(public_key, message, ctx, signature).is_ok()
}

/// The 1952-byte ML-DSA-65 public key for a 32-byte seed (FIPS 204 `ML-DSA.KeyGen`).
pub fn ml_dsa65_public_key(seed: &[u8]) -> Result<Vec<u8>, MlDsaFailure> {
    let sk = SigningKey::<MlDsa65>::from_seed(&parse_seed(seed)?);
    Ok(sk.verifying_key().encode().as_slice().to_vec())
}

/// Signs `message` under the key derived from a 32-byte seed, with the
/// deterministic variant of FIPS 204 Algorithm 2 (`rnd` is all zeros), so the
/// same inputs always give the same 3309-byte signature.
pub fn ml_dsa65_sign(seed: &[u8], message: &[u8], ctx: &[u8]) -> Result<Vec<u8>, MlDsaFailure> {
    check_context(ctx)?;
    let sk = SigningKey::<MlDsa65>::from_seed(&parse_seed(seed)?);
    let sig = sk
        .expanded_key()
        .sign_deterministic(message, ctx)
        .map_err(|_| MlDsaFailure::SignatureMalformed)?;
    Ok(sig.encode().as_slice().to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
    use serde_json::Value;

    const ACVP_V1: &str = include_str!("../../testvectors/mldsa/ml_dsa65_acvp_v1.json");
    const FRAME_V1: &str = include_str!("../../testvectors/mldsa/ml_dsa65_frame_signing_v1.json");

    fn hex_bytes(s: &str) -> Vec<u8> {
        hex::decode(s).expect("vector hex")
    }

    fn b64(v: &Value, field: &str) -> Vec<u8> {
        BASE64
            .decode(v[field].as_str().expect("vector field"))
            .expect("vector base64")
    }

    #[test]
    fn acvp_sig_ver_known_answers() {
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        let cases = doc["sig_ver"].as_array().unwrap();
        let mut seen_pass = 0;
        let mut seen_fail = 0;
        for c in cases {
            let pk = hex_bytes(c["pk"].as_str().unwrap());
            let msg = hex_bytes(c["message"].as_str().unwrap());
            let ctx = hex_bytes(c["context"].as_str().unwrap());
            let sig = hex_bytes(c["signature"].as_str().unwrap());
            let want = c["test_passed"].as_bool().unwrap();
            let got = ml_dsa65_verify(&pk, &msg, &ctx, &sig);
            assert_eq!(got, want, "ACVP sigVer tcId {}", c["tc_id"]);
            if want {
                seen_pass += 1
            } else {
                seen_fail += 1
            }
        }
        assert!(
            seen_pass > 0 && seen_fail > 0,
            "vector set must include pass and fail cases"
        );
    }

    #[test]
    fn acvp_key_gen_known_answers() {
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        for c in doc["key_gen"].as_array().unwrap() {
            let seed = hex_bytes(c["seed"].as_str().unwrap());
            let want = hex_bytes(c["pk"].as_str().unwrap());
            assert_eq!(
                ml_dsa65_public_key(&seed).unwrap(),
                want,
                "ACVP keyGen tcId {}",
                c["tc_id"]
            );
        }
    }

    #[test]
    fn sign_round_trips_and_is_deterministic() {
        // ACVP sigGen vectors give expanded secret keys, not seeds, so signing is
        // checked by round trip against the public key from the keyGen vectors.
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        let c = &doc["key_gen"].as_array().unwrap()[0];
        let seed = hex_bytes(c["seed"].as_str().unwrap());
        let pk = hex_bytes(c["pk"].as_str().unwrap());
        let msg = b"knirv sign round trip";
        let ctx = b"ctx";
        let sig = ml_dsa65_sign(&seed, msg, ctx).unwrap();
        assert_eq!(sig.len(), SIGNATURE_LEN);
        assert!(ml_dsa65_verify(&pk, msg, ctx, &sig));
        assert!(
            !ml_dsa65_verify(&pk, msg, b"other", &sig),
            "wrong ctx must fail"
        );
        assert_eq!(
            ml_dsa65_sign(&seed, msg, ctx).unwrap(),
            sig,
            "deterministic variant"
        );
    }

    #[test]
    fn wrong_lengths_are_refused_without_panic() {
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        let c = &doc["sig_ver"].as_array().unwrap()[0];
        let pk = hex_bytes(c["pk"].as_str().unwrap());
        let msg = hex_bytes(c["message"].as_str().unwrap());
        let ctx = hex_bytes(c["context"].as_str().unwrap());
        let sig = hex_bytes(c["signature"].as_str().unwrap());
        assert!(!ml_dsa65_verify(
            &pk[..PUBLIC_KEY_LEN - 1],
            &msg,
            &ctx,
            &sig
        ));
        assert!(!ml_dsa65_verify(&[], &msg, &ctx, &sig));
        assert!(!ml_dsa65_verify(&pk, &msg, &ctx, &sig[..SIGNATURE_LEN - 1]));
        assert!(!ml_dsa65_verify(&pk, &msg, &ctx, &[0u8; SIGNATURE_LEN + 1]));
        assert_eq!(
            ml_dsa65_verify_detail(&pk, &msg, &ctx, &sig[..10]),
            Err(MlDsaFailure::SignatureLength)
        );
        assert_eq!(
            ml_dsa65_public_key(&[0u8; 31]),
            Err(MlDsaFailure::SeedLength)
        );
        assert_eq!(
            ml_dsa65_sign(&[0u8; 32], b"m", &[0u8; 256]),
            Err(MlDsaFailure::ContextLength)
        );
    }

    #[test]
    fn verify_refuses_wrong_public_key_length_with_detail() {
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        let c = &doc["sig_ver"].as_array().unwrap()[0];
        let pk = hex_bytes(c["pk"].as_str().unwrap());
        let msg = hex_bytes(c["message"].as_str().unwrap());
        let ctx = hex_bytes(c["context"].as_str().unwrap());
        let sig = hex_bytes(c["signature"].as_str().unwrap());
        assert_eq!(pk.len(), PUBLIC_KEY_LEN);
        // A zero-padded or truncated key must be refused, not repaired and verified.
        let mut long = pk.clone();
        long.push(0);
        assert_eq!(
            ml_dsa65_verify_detail(&pk[..PUBLIC_KEY_LEN - 1], &msg, &ctx, &sig),
            Err(MlDsaFailure::PublicKeyLength)
        );
        assert_eq!(
            ml_dsa65_verify_detail(&long, &msg, &ctx, &sig),
            Err(MlDsaFailure::PublicKeyLength)
        );
        assert_eq!(
            ml_dsa65_verify_detail(&[], &msg, &ctx, &sig),
            Err(MlDsaFailure::PublicKeyLength)
        );
    }

    #[test]
    fn verify_refuses_context_longer_than_255_with_detail() {
        let doc: Value = serde_json::from_str(ACVP_V1).unwrap();
        let c = &doc["sig_ver"].as_array().unwrap()[0];
        let pk = hex_bytes(c["pk"].as_str().unwrap());
        let msg = hex_bytes(c["message"].as_str().unwrap());
        let sig = hex_bytes(c["signature"].as_str().unwrap());
        let ctx_ok = vec![0u8; MAX_CONTEXT_LEN];
        let ctx_long = vec![0u8; MAX_CONTEXT_LEN + 1];
        // 255 bytes is the largest legal context: it is not refused for length.
        assert_ne!(
            ml_dsa65_verify_detail(&pk, &msg, &ctx_ok, &sig),
            Err(MlDsaFailure::ContextLength)
        );
        assert_eq!(
            ml_dsa65_verify_detail(&pk, &msg, &ctx_long, &sig),
            Err(MlDsaFailure::ContextLength)
        );
    }

    #[test]
    fn frame_signing_message_cross_check_from_knirvbase_circl() {
        // Vector generated by KNIRVBASE go/pkg/nrv (circl ML-DSA-65) over
        // nrv.FrameSigningMessage; the key is a test-only seed. See the vector file.
        let doc: Value = serde_json::from_str(FRAME_V1).unwrap();
        let pk = b64(&doc, "public_key_base64");
        let msg = b64(&doc, "signed_message_base64");
        let sig = b64(&doc, "signature_base64");
        assert!(
            ml_dsa65_verify(&pk, &msg, &[], &sig),
            "circl frame signature must verify in the core"
        );
        let mut tampered = msg.clone();
        let last = tampered.len() - 1;
        tampered[last] ^= 1;
        assert!(
            !ml_dsa65_verify(&pk, &tampered, &[], &sig),
            "tampered payload byte must fail"
        );
    }
}
