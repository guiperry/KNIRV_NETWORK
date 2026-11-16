//! Uniform open payload every `.nrv` frame may carry. This is the source of
//! truth for the schema; `go/pkg/nrv/payload.go` and
//! `ts/src/components/storage/nrv/payload.ts` mirror these widths, range checks
//! and absent-vs-empty rules, and all three read `testdata/framepayload/cases.json`.
//!
//! A frame indexes its encoded payload under [`PAYLOAD_MODALITY`] in
//! `FrameEntry.modalities`.
//!
//! # Integer widths
//!
//! | Field                                   | Width | Valid range            | Why                                                              |
//! |-----------------------------------------|-------|------------------------|------------------------------------------------------------------|
//! | `token_window.tokens[]`                 | `u32` | `0..=u32::MAX`         | Vocabulary indices, never negative; no vocab approaches 2^32; exact in a JS number. |
//! | `token_window.start`                    | `u32` | `0..=u32::MAX`         | Offset of the window in one frame's token stream.                |
//! | `token_window.brackets`                 | `u16` | `0..=MAX_BRACKETS_PER_FRAME` (1024) | Brackets per frame are capped by the producer (`MaxBracketsPerFrame`). |
//! | `supervision_episode.tool_trace[].tier` | `u8`  | `0..=MAX_TOOL_TIER` (3) | Tool tier set, owner-decided: 0 execute, 1 ask, 2 deny, 3 escalate. |
//!
//! Boxes use normalised coordinates: `x`, `y`, `w`, `h` are finite and each lies in
//! `0..=1` of the frame (top-left origin, `w`/`h` are extents). A box must stay
//! inside the frame, so `x + w <= 1` and `y + h <= 1`, with a [`BOX_EPSILON`]
//! tolerance for float rounding. Embedding values are finite.
//!
//! # Absent means empty
//!
//! A string or array that is empty is omitted on encode and read back as the
//! zero value. An optional sub-payload (`Option<_>`) whose fields are all empty
//! or zero is treated as absent and is dropped, so `{}`, `null` and
//! `{"perceptual":{"boxes":[]}}` all canonicalise to `{}`.

use serde::{Deserialize, Serialize};
use std::fmt;

/// Key under which a frame's encoded `FramePayload` is indexed.
pub const PAYLOAD_MODALITY: &str = "payload";

/// Highest tool tier. The tier set is 0..=3: 0 execute, 1 ask, 2 deny, 3 escalate.
pub const MAX_TOOL_TIER: u8 = 3;

/// Tolerance for `x + w <= 1` and `y + h <= 1` on normalised boxes, for float rounding.
pub const BOX_EPSILON: f64 = 1e-9;

/// Producer cap on brackets in one frame (`go/pkg/nrv/convert_nrv3.go`
/// `MaxBracketsPerFrame`). A token window cannot hold more brackets than its frame.
pub const MAX_BRACKETS_PER_FRAME: u16 = 1024;

/// Errors from decoding, validating or encoding a payload.
#[derive(Debug, Clone, PartialEq)]
pub enum PayloadError {
    /// Bytes are not a JSON object, or a value has the wrong JSON type or does
    /// not fit its integer width (serde's message names the type, not the field).
    Decode(String),
    /// A value is well-typed but outside the range its schema allows.
    Range { field: String, detail: String },
}

impl fmt::Display for PayloadError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            PayloadError::Decode(msg) => write!(f, "payload decode: {msg}"),
            PayloadError::Range { field, detail } => write!(f, "payload field {field}: {detail}"),
        }
    }
}

impl std::error::Error for PayloadError {}

impl From<serde_json::Error> for PayloadError {
    fn from(e: serde_json::Error) -> Self {
        PayloadError::Decode(e.to_string())
    }
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct FramePayload {
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub token_window: Option<TokenWindowPayload>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub error_test_case: Option<ErrorTestCasePayload>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub trl_sample: Option<TRLSamplePayload>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub supervision_episode: Option<SupervisionEpisodePayload>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub perceptual: Option<PerceptualPayload>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct TokenWindowPayload {
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub tokens: Vec<u32>,
    #[serde(default)]
    pub start: u32,
    #[serde(default)]
    pub brackets: u16,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct ErrorTestCasePayload {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub input: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub expected: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct TRLSamplePayload {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub prompt: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub completion: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub chosen: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub rejected: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct ToolCallPayload {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool: String,
    #[serde(default)]
    pub tier: u8,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub outcome: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct SupervisionEpisodePayload {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub state: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub choices: Vec<String>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub decision: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub tool_trace: Vec<ToolCallPayload>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub outcome: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct BoxPayload {
    #[serde(default)]
    pub x: f64,
    #[serde(default)]
    pub y: f64,
    #[serde(default)]
    pub w: f64,
    #[serde(default)]
    pub h: f64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub label: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct PerceptualPayload {
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub image_hashes: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub boxes: Vec<BoxPayload>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub ocr: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub embedding: Vec<f32>,
}

impl TokenWindowPayload {
    fn is_empty(&self) -> bool {
        self.tokens.is_empty() && self.start == 0 && self.brackets == 0
    }
}

impl ErrorTestCasePayload {
    fn is_empty(&self) -> bool {
        self.id.is_empty() && self.input.is_empty() && self.expected.is_empty()
    }
}

impl TRLSamplePayload {
    fn is_empty(&self) -> bool {
        self.prompt.is_empty() && self.completion.is_empty() && self.chosen.is_empty() && self.rejected.is_empty()
    }
}

impl SupervisionEpisodePayload {
    fn is_empty(&self) -> bool {
        self.state.is_empty()
            && self.choices.is_empty()
            && self.decision.is_empty()
            && self.tool_trace.is_empty()
            && self.outcome.is_empty()
    }
}

impl PerceptualPayload {
    fn is_empty(&self) -> bool {
        self.image_hashes.is_empty() && self.boxes.is_empty() && self.ocr.is_empty() && self.embedding.is_empty()
    }
}

impl FramePayload {
    /// The payload as a test-taker may see it: the expected answer of an
    /// `ErrorTestCase` is removed. The receiver is not modified.
    pub fn public_view(&self) -> FramePayload {
        let mut out = self.clone();
        if let Some(case) = out.error_test_case.as_mut() {
            case.expected.clear();
        }
        out
    }

    /// Returns the payload with every empty optional sub-payload set to `None`.
    pub fn normalized(&self) -> FramePayload {
        let mut out = self.clone();
        if out.token_window.as_ref().is_some_and(|v| v.is_empty()) {
            out.token_window = None;
        }
        if out.error_test_case.as_ref().is_some_and(|v| v.is_empty()) {
            out.error_test_case = None;
        }
        if out.trl_sample.as_ref().is_some_and(|v| v.is_empty()) {
            out.trl_sample = None;
        }
        if out.supervision_episode.as_ref().is_some_and(|v| v.is_empty()) {
            out.supervision_episode = None;
        }
        if out.perceptual.as_ref().is_some_and(|v| v.is_empty()) {
            out.perceptual = None;
        }
        out
    }

    /// Checks the range rules that the integer widths do not already enforce.
    /// Run on the normalized payload.
    pub fn validate(&self) -> Result<(), PayloadError> {
        if let Some(tw) = &self.token_window {
            if tw.brackets > MAX_BRACKETS_PER_FRAME {
                return Err(PayloadError::Range {
                    field: "token_window.brackets".into(),
                    detail: format!("{} exceeds the per-frame cap of {}", tw.brackets, MAX_BRACKETS_PER_FRAME),
                });
            }
        }
        if let Some(se) = &self.supervision_episode {
            for (i, t) in se.tool_trace.iter().enumerate() {
                if t.tier > MAX_TOOL_TIER {
                    return Err(PayloadError::Range {
                        field: format!("supervision_episode.tool_trace[{i}].tier"),
                        detail: format!("{} is outside the tool tier set 0..={}", t.tier, MAX_TOOL_TIER),
                    });
                }
            }
        }
        if let Some(p) = &self.perceptual {
            for (i, b) in p.boxes.iter().enumerate() {
                for (name, v) in [("x", b.x), ("y", b.y), ("w", b.w), ("h", b.h)] {
                    if !v.is_finite() {
                        return Err(PayloadError::Range {
                            field: format!("perceptual.boxes[{i}].{name}"),
                            detail: format!("{v} is not finite"),
                        });
                    }
                    if !(0.0..=1.0).contains(&v) {
                        return Err(PayloadError::Range {
                            field: format!("perceptual.boxes[{i}].{name}"),
                            detail: format!("{v} is outside the normalised range 0..=1"),
                        });
                    }
                }
                if b.x + b.w > 1.0 + BOX_EPSILON {
                    return Err(PayloadError::Range {
                        field: format!("perceptual.boxes[{i}].x"),
                        detail: format!("x + w = {} extends past the frame (1)", b.x + b.w),
                    });
                }
                if b.y + b.h > 1.0 + BOX_EPSILON {
                    return Err(PayloadError::Range {
                        field: format!("perceptual.boxes[{i}].y"),
                        detail: format!("y + h = {} extends past the frame (1)", b.y + b.h),
                    });
                }
            }
            for (i, e) in p.embedding.iter().enumerate() {
                if !e.is_finite() {
                    return Err(PayloadError::Range {
                        field: format!("perceptual.embedding[{i}]"),
                        detail: format!("{e} is not finite"),
                    });
                }
            }
        }
        Ok(())
    }

    /// Serialises the normalized, validated payload. Invalid payloads are refused
    /// rather than written.
    pub fn encode(&self) -> Result<Vec<u8>, PayloadError> {
        let n = self.normalized();
        n.validate()?;
        Ok(serde_json::to_vec(&n)?)
    }

    /// Parses payload bytes, then normalizes and validates them.
    pub fn decode(bytes: &[u8]) -> Result<FramePayload, PayloadError> {
        let trimmed = bytes.iter().position(|b| !b.is_ascii_whitespace()).map(|i| bytes[i]);
        if trimmed != Some(b'{') {
            return Err(PayloadError::Decode("payload must be a JSON object".into()));
        }
        let p: FramePayload = serde_json::from_slice(bytes)?;
        let n = p.normalized();
        n.validate()?;
        Ok(n)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const CASES: &str = include_str!("../../testdata/framepayload/cases.json");

    fn full() -> FramePayload {
        FramePayload {
            token_window: Some(TokenWindowPayload { tokens: vec![1, 2, 3], start: 0, brackets: 2 }),
            error_test_case: Some(ErrorTestCasePayload { id: "t1".into(), input: "2+2".into(), expected: "4".into() }),
            trl_sample: Some(TRLSamplePayload {
                prompt: "p".into(),
                completion: String::new(),
                chosen: "good".into(),
                rejected: "bad".into(),
            }),
            supervision_episode: Some(SupervisionEpisodePayload {
                state: "node healthy".into(),
                choices: vec!["no-op".into(), "drain".into()],
                decision: "no-op".into(),
                tool_trace: vec![ToolCallPayload { tool: "manifest.get".into(), tier: 0, outcome: "execute".into() }],
                outcome: "ok".into(),
            }),
            perceptual: Some(PerceptualPayload {
                image_hashes: vec!["ab".into()],
                boxes: vec![BoxPayload { x: 0.1, y: 0.2, w: 0.3, h: 0.4, label: "cat".into() }],
                ocr: "hi".into(),
                embedding: vec![0.5],
            }),
        }
    }

    fn one_box(w: f64, h: f64) -> FramePayload {
        FramePayload {
            perceptual: Some(PerceptualPayload {
                boxes: vec![BoxPayload { x: 0.0, y: 0.0, w, h, label: String::new() }],
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    fn one_window(tokens: Vec<u32>, start: u32, brackets: u16) -> FramePayload {
        FramePayload { token_window: Some(TokenWindowPayload { tokens, start, brackets }), ..Default::default() }
    }

    #[test]
    fn round_trip_keeps_all_five_fields() {
        let p = full();
        let back = FramePayload::decode(&p.encode().unwrap()).unwrap();
        assert_eq!(p, back);
        assert!(back.token_window.is_some() && back.error_test_case.is_some());
        assert!(back.trl_sample.is_some() && back.supervision_episode.is_some() && back.perceptual.is_some());
    }

    #[test]
    fn public_view_strips_only_expected() {
        let p = full();
        let pv = p.public_view();
        assert_eq!(pv.error_test_case.as_ref().unwrap().expected, "");
        assert_eq!(p.error_test_case.as_ref().unwrap().expected, "4");
        assert_eq!(pv.error_test_case.as_ref().unwrap().input, "2+2");
        assert!(pv.trl_sample.is_some());
    }

    #[test]
    fn empty_payload_encodes_as_empty_object() {
        assert_eq!(FramePayload::default().encode().unwrap(), b"{}".to_vec());
    }

    #[test]
    fn matches_go_wire_names() {
        let json = String::from_utf8(full().encode().unwrap()).unwrap();
        for key in ["token_window", "error_test_case", "trl_sample", "supervision_episode", "perceptual", "tool_trace", "image_hashes"] {
            assert!(json.contains(&format!("\"{}\"", key)), "missing wire key {}", key);
        }
    }

    #[test]
    fn token_id_and_start_bounds() {
        assert!(one_window(vec![0, u32::MAX], u32::MAX, 0).encode().is_ok());
        let over = br#"{"token_window":{"tokens":[4294967296],"start":0,"brackets":0}}"#;
        assert!(matches!(FramePayload::decode(over), Err(PayloadError::Decode(_))));
        let neg = br#"{"token_window":{"tokens":[-1],"start":0,"brackets":0}}"#;
        assert!(matches!(FramePayload::decode(neg), Err(PayloadError::Decode(_))));
        let start_over = br#"{"token_window":{"tokens":[1],"start":4294967296,"brackets":0}}"#;
        assert!(matches!(FramePayload::decode(start_over), Err(PayloadError::Decode(_))));
    }

    #[test]
    fn brackets_frame_cap_is_named() {
        assert!(one_window(vec![1], 0, MAX_BRACKETS_PER_FRAME).encode().is_ok());
        match one_window(vec![1], 0, MAX_BRACKETS_PER_FRAME + 1).encode() {
            Err(PayloadError::Range { field, .. }) => assert_eq!(field, "token_window.brackets"),
            other => panic!("expected range error, got {other:?}"),
        }
    }

    fn one_tier(tier: u8) -> FramePayload {
        FramePayload {
            supervision_episode: Some(SupervisionEpisodePayload {
                state: "s".into(),
                tool_trace: vec![ToolCallPayload { tool: "t".into(), tier, outcome: "o".into() }],
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    #[test]
    fn tier_is_0_to_3() {
        for t in 0..=MAX_TOOL_TIER {
            assert!(one_tier(t).encode().is_ok(), "tier {t} must be accepted");
        }
        match one_tier(MAX_TOOL_TIER + 1).encode() {
            Err(PayloadError::Range { field, .. }) => assert_eq!(field, "supervision_episode.tool_trace[0].tier"),
            other => panic!("expected range error, got {other:?}"),
        }
        let over = br#"{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":4,"outcome":"o"}]}}"#;
        assert!(FramePayload::decode(over).is_err());
        let wide = br#"{"supervision_episode":{"state":"s","tool_trace":[{"tool":"t","tier":256,"outcome":"o"}]}}"#;
        assert!(FramePayload::decode(wide).is_err());
    }

    #[test]
    fn box_size_and_finiteness() {
        assert!(one_box(0.0, 0.0).encode().is_ok());
        assert!(one_box(-0.5, 1.0).encode().is_err());
        assert!(one_box(1.0, -1.0).encode().is_err());
        assert!(one_box(f64::NAN, 1.0).encode().is_err());
        assert!(one_box(1.0, f64::INFINITY).encode().is_err());
    }

    fn one_box_at(x: f64, y: f64, w: f64, h: f64) -> FramePayload {
        FramePayload {
            perceptual: Some(PerceptualPayload {
                boxes: vec![BoxPayload { x, y, w, h, label: String::new() }],
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    #[test]
    fn box_must_stay_in_the_unit_frame() {
        // Whole frame, and a box flush against the far edges, are valid.
        assert!(one_box_at(0.0, 0.0, 1.0, 1.0).encode().is_ok());
        assert!(one_box_at(0.25, 0.25, 0.75, 0.75).encode().is_ok());
        assert!(one_box_at(1.0, 1.0, 0.0, 0.0).encode().is_ok());
        // Origin past the frame, or extent past the far edge, is refused.
        assert!(one_box_at(1.5, 0.0, 0.0, 0.0).encode().is_err());
        assert!(one_box_at(-0.1, 0.0, 0.1, 0.1).encode().is_err());
        assert!(one_box_at(0.0, 0.0, 1.5, 0.1).encode().is_err());
        assert!(one_box_at(0.6, 0.0, 0.5, 0.1).encode().is_err());
        assert!(one_box_at(0.0, 0.6, 0.1, 0.5).encode().is_err());
    }

    #[test]
    fn box_exact_edge_accepted_just_over_rejected() {
        // x + w == 1 and y + h == 1 exactly are accepted.
        assert!(one_box_at(0.5, 0.5, 0.5, 0.5).encode().is_ok());
        // Within BOX_EPSILON of the edge (float rounding) is accepted.
        assert!(one_box_at(0.5, 0.5, 0.5 + BOX_EPSILON / 2.0, 0.5).encode().is_ok());
        assert!(one_box_at(0.5, 0.5, 0.5, 0.5 + BOX_EPSILON / 2.0).encode().is_ok());
        // Past BOX_EPSILON is rejected, on each axis, with the named field.
        for (x, y, w, h, field) in [
            (0.5, 0.5, 0.5 + 2.0 * BOX_EPSILON, 0.5, "perceptual.boxes[0].x"),
            (0.5, 0.5, 0.5, 0.5 + 2.0 * BOX_EPSILON, "perceptual.boxes[0].y"),
            (0.5, 0.0, 0.500001, 0.1, "perceptual.boxes[0].x"),
            (0.0, 0.5, 0.1, 0.500001, "perceptual.boxes[0].y"),
        ] {
            match one_box_at(x, y, w, h).encode() {
                Err(PayloadError::Range { field: f, .. }) => assert_eq!(f, field),
                other => panic!("({x},{y},{w},{h}) expected range error, got {other:?}"),
            }
        }
    }

    #[test]
    fn empty_sub_payloads_are_absent() {
        let p = FramePayload {
            token_window: Some(TokenWindowPayload::default()),
            error_test_case: Some(ErrorTestCasePayload::default()),
            trl_sample: Some(TRLSamplePayload::default()),
            supervision_episode: Some(SupervisionEpisodePayload::default()),
            perceptual: Some(PerceptualPayload::default()),
        };
        assert_eq!(p.encode().unwrap(), b"{}".to_vec());
        assert_eq!(FramePayload::decode(b"{\"perceptual\":{\"boxes\":[]}}").unwrap(), FramePayload::default());
    }

    #[test]
    fn non_object_top_level_is_rejected() {
        assert!(FramePayload::decode(b"null").is_err());
        assert!(FramePayload::decode(b"[]").is_err());
    }

    /// Compares two JSON values, treating integer-valued and float numbers as equal
    /// (`1` and `1.0` encode differently across Rust, Go and TS but mean the same).
    fn json_eq(a: &serde_json::Value, b: &serde_json::Value) -> bool {
        use serde_json::Value::*;
        match (a, b) {
            (Number(x), Number(y)) => x.as_f64() == y.as_f64(),
            (Array(x), Array(y)) => x.len() == y.len() && x.iter().zip(y).all(|(p, q)| json_eq(p, q)),
            (Object(x), Object(y)) => {
                x.len() == y.len() && x.iter().all(|(k, v)| y.get(k).is_some_and(|w| json_eq(v, w)))
            }
            _ => a == b,
        }
    }

    #[test]
    fn shared_conformance_fixture() {
        let doc: serde_json::Value = serde_json::from_str(CASES).unwrap();
        let cases = doc["cases"].as_array().unwrap();
        assert!(!cases.is_empty());
        for case in cases {
            let name = case["name"].as_str().unwrap();
            let input = serde_json::to_vec(&case["input"]).unwrap();
            match case["expect"].as_str().unwrap() {
                "ok" => {
                    let p = FramePayload::decode(&input).unwrap_or_else(|e| panic!("{name}: decode failed: {e}"));
                    let encoded: serde_json::Value = serde_json::from_slice(&p.encode().unwrap()).unwrap();
                    assert!(json_eq(&encoded, &case["canonical"]), "{name}: encoded {encoded} != canonical {}", case["canonical"]);
                }
                "error" => {
                    assert!(FramePayload::decode(&input).is_err(), "{name}: expected decode/validate error");
                }
                other => panic!("{name}: unknown expect {other}"),
            }
        }
    }
}
