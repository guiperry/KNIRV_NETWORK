//! The single `knirv://` URI grammar shared by every KNIRV language SDK.

use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use url::Url;

/// Registry of routable resource types.
pub const RESOURCE_TYPES: &[&str] = &[
    "chain",
    "nrn",
    "ulora",
    "mcpcontext",
    "resource",
    "tool",
    "prompt",
    "memoryservice",
    "memory",
    "capability",
    "dev",
    "errornode",
    "skillnode",
    "ideanode",
    "propertynode",
    "usdscene",
    "usdasset",
];

/// Query keys reserved by the grammar.
pub const QUERY_KEYS: &[&str] = &["tenant", "snapshot", "sha256"];

/// Classifies a parsed URI.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Kind {
    Canonical,
    LegacyDve,
    ContentPointer,
    Action,
    Invalid,
}

/// One of the three historical DVE URI shapes.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct LegacyDve {
    pub form: String,
    pub id: String,
    pub hash: Option<String>,
    pub name: Option<String>,
}

/// The on-chain `knirv://network/<kind>_<hash>` shape.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ContentPointer {
    pub kind: String,
    pub hash: String,
}

/// A QR-login or signing-approval link.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Action {
    pub name: String,
    pub canonical: String,
    pub query: BTreeMap<String, String>,
}

/// Typed result of [`parse`].
#[derive(Clone, Debug)]
pub struct Result {
    pub kind: Kind,
    pub canonical: String,
    pub id: String,
    pub resource_type: String,
    pub path: String,
    pub query: BTreeMap<String, String>,
    pub legacy_dve: Option<LegacyDve>,
    pub content_pointer: Option<ContentPointer>,
    pub action: Option<Action>,
    pub reason: Option<String>,
}

impl Result {
    pub fn is_valid(&self) -> bool {
        !matches!(self.kind, Kind::Invalid)
    }
}

fn invalid(reason: impl Into<String>) -> Result {
    Result {
        kind: Kind::Invalid,
        canonical: String::new(),
        id: String::new(),
        resource_type: String::new(),
        path: String::new(),
        query: BTreeMap::new(),
        legacy_dve: None,
        content_pointer: None,
        action: None,
        reason: Some(reason.into()),
    }
}

/// Parses a raw URI without returning an error.
pub fn parse(raw: &str) -> Result {
    let value = raw.trim();
    let url = match Url::parse(value) {
        Ok(url) => url,
        Err(_) => return invalid("malformed URI"),
    };
    if url.scheme() != "knirv" {
        return invalid("scheme must be knirv://");
    }
    if !url.username().is_empty() || url.password().is_some() {
        return invalid("userinfo is not allowed");
    }
    if url.fragment().is_some() {
        return invalid("fragment is not allowed");
    }
    let host = match url.host_str() {
        Some(host) if url.port().is_none() && !host.is_empty() => host.to_lowercase(),
        _ => return invalid("host must be a single DNS label"),
    };
    let mut query = normalize_query(&url);
    let path = url.path().to_string();

    if !host.contains('.') {
        return parse_apex(&host, &path, &mut query);
    }

    let labels: Vec<&str> = host.split('.').collect();
    if labels.len() != 2 {
        return invalid("host must have exactly two labels");
    }
    let raw_id = labels[0];
    let resource_type = labels[1];
    if raw_id.is_empty() {
        return invalid("id must not be empty");
    }
    if !RESOURCE_TYPES.contains(&resource_type) {
        return invalid(format!("unknown resource type: {resource_type}"));
    }
    let id = normalize_id(raw_id);
    let path = if path.is_empty() {
        "/".to_string()
    } else if path.starts_with('/') {
        path
    } else {
        format!("/{path}")
    };
    if resource_type == "usdasset" {
        if let Some(sha) = query.get("sha256") {
            if !sha.to_lowercase().starts_with(&id) {
                return invalid("usdasset id must be a prefix of sha256");
            }
        }
    }
    let canonical = canonical_uri(&id, resource_type, &path, &query);
    Result {
        kind: Kind::Canonical,
        canonical,
        id,
        resource_type: resource_type.to_string(),
        path,
        query,
        legacy_dve: None,
        content_pointer: None,
        action: None,
        reason: None,
    }
}

fn parse_apex(host: &str, path: &str, query: &mut BTreeMap<String, String>) -> Result {
    if host == "auth" || host == "sign" {
        let canonical = format!("knirv://{host}{}", canonical_query(query));
        return Result {
            kind: Kind::Action,
            canonical: canonical.clone(),
            id: String::new(),
            resource_type: String::new(),
            path: String::new(),
            query: query.clone(),
            legacy_dve: None,
            content_pointer: None,
            action: Some(Action {
                name: host.to_string(),
                canonical,
                query: query.clone(),
            }),
            reason: None,
        };
    }
    if host == "network" {
        let trimmed = path.trim_start_matches('/');
        if trimmed.contains('/') {
            return invalid("content pointer must be network/<kind>_<hash>");
        }
        if let Some((kind, hash)) = trimmed.split_once('_') {
            if kind.is_empty() || hash.is_empty() {
                return invalid("content pointer must be network/<kind>_<hash>");
            }
            return Result {
                kind: Kind::ContentPointer,
                canonical: format!("knirv://network/{kind}_{hash}"),
                id: String::new(),
                resource_type: String::new(),
                path: format!("/{trimmed}"),
                query: query.clone(),
                legacy_dve: None,
                content_pointer: Some(ContentPointer {
                    kind: kind.to_string(),
                    hash: hash.to_string(),
                }),
                action: None,
                reason: None,
            };
        }
        return invalid("content pointer must be network/<kind>_<hash>");
    }

    let segments: Vec<&str> = path
        .trim_start_matches('/')
        .split('/')
        .filter(|s| !s.is_empty())
        .collect();
    if host == "dve" && segments.len() >= 2 {
        return Result {
            kind: Kind::LegacyDve,
            canonical: format!("knirv://{host}{path}"),
            id: String::new(),
            resource_type: String::new(),
            path: path.to_string(),
            query: query.clone(),
            legacy_dve: Some(LegacyDve {
                form: "dve/id/name".to_string(),
                id: segments[0].to_string(),
                hash: None,
                name: Some(segments[1].to_string()),
            }),
            content_pointer: None,
            action: None,
            reason: None,
        };
    }
    if segments.len() == 1 {
        return Result {
            kind: Kind::LegacyDve,
            canonical: format!("knirv://{host}{path}"),
            id: String::new(),
            resource_type: String::new(),
            path: path.to_string(),
            query: query.clone(),
            legacy_dve: Some(LegacyDve {
                form: "hash/id".to_string(),
                id: segments[0].to_string(),
                hash: Some(host.to_string()),
                name: None,
            }),
            content_pointer: None,
            action: None,
            reason: None,
        };
    }
    Result {
        kind: Kind::LegacyDve,
        canonical: format!("knirv://{host}"),
        id: String::new(),
        resource_type: String::new(),
        path: String::new(),
        query: query.clone(),
        legacy_dve: Some(LegacyDve {
            form: "id".to_string(),
            id: host.to_string(),
            hash: None,
            name: None,
        }),
        content_pointer: None,
        action: None,
        reason: None,
    }
}

fn normalize_query(url: &Url) -> BTreeMap<String, String> {
    let mut out = BTreeMap::new();
    for (key, value) in url.query_pairs() {
        out.insert(key.to_lowercase(), value.into_owned());
    }
    out
}

fn canonical_uri(
    id: &str,
    resource_type: &str,
    path: &str,
    query: &BTreeMap<String, String>,
) -> String {
    format!(
        "knirv://{id}.{resource_type}{path}{}",
        canonical_query(query)
    )
}

fn canonical_query(query: &BTreeMap<String, String>) -> String {
    if query.is_empty() {
        return String::new();
    }
    let mut parts = Vec::with_capacity(query.len());
    for (key, value) in query {
        parts.push(format!("{}={}", percent_encode(key), percent_encode(value)));
    }
    format!("?{}", parts.join("&"))
}

fn percent_encode(value: &str) -> String {
    let mut out = String::with_capacity(value.len());
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(byte as char)
            }
            _ => out.push_str(&format!("%{byte:02X}")),
        }
    }
    out
}

/// Lowercases valid IDs and derives a host-safe ID for anything else.
pub fn normalize_id(raw: &str) -> String {
    let lower = raw.to_lowercase();
    if valid_id(&lower) {
        return lower;
    }
    derive_id(raw)
}

/// Maps a non-conforming raw ID to `base32(sha256(raw))[:32]` lowercase.
pub fn derive_id(raw: &str) -> String {
    let digest = Sha256::digest(raw.as_bytes());
    let encoded = base32(&digest).to_lowercase();
    encoded[..32].to_string()
}

fn valid_id(id: &str) -> bool {
    if id.is_empty() || id.len() > 63 {
        return false;
    }
    id.bytes()
        .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'-')
}

fn base32(data: &[u8]) -> String {
    const ALPHABET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
    let mut out = String::new();
    let mut buffer: u32 = 0;
    let mut bits: u32 = 0;
    for byte in data {
        buffer = (buffer << 8) | u32::from(*byte);
        bits += 8;
        while bits >= 5 {
            bits -= 5;
            out.push(ALPHABET[((buffer >> bits) & 0x1f) as usize] as char);
        }
    }
    if bits > 0 {
        out.push(ALPHABET[((buffer << (5 - bits)) & 0x1f) as usize] as char);
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::Value;

    fn vectors() -> Value {
        serde_json::from_str(include_str!("../../testvectors/uri/uri_v1.json"))
            .expect("vector json")
    }

    #[test]
    fn registry_matches_vectors() {
        let value = vectors();
        let types: Vec<String> = value["resource_types"]
            .as_array()
            .expect("types")
            .iter()
            .map(|v| v.as_str().expect("type").to_string())
            .collect();
        for name in &types {
            assert!(
                RESOURCE_TYPES.contains(&name.as_str()),
                "missing resource type {name}"
            );
        }
        assert_eq!(types.len(), RESOURCE_TYPES.len());
    }

    #[test]
    fn derive_matches_vectors() {
        let value = vectors();
        for case in value["derive"].as_array().expect("derive") {
            let raw = case["raw"].as_str().expect("raw");
            let want = case["id"].as_str().expect("id");
            assert_eq!(derive_id(raw), want, "raw {raw}");
        }
    }

    #[test]
    fn parse_matches_vectors() {
        let value = vectors();
        for case in value["cases"].as_array().expect("cases") {
            let name = case["name"].as_str().expect("name");
            let input = case["input"].as_str().expect("input");
            let kind = case["kind"].as_str().expect("kind");
            let got = parse(input);
            match kind {
                "invalid" => {
                    assert_eq!(got.kind, Kind::Invalid, "{name}");
                    assert_eq!(got.reason.as_deref(), case["reason"].as_str(), "{name}");
                }
                "canonical" => {
                    assert_eq!(got.kind, Kind::Canonical, "{name}");
                    assert_eq!(
                        got.canonical,
                        case["canonical"].as_str().expect("canonical"),
                        "{name}"
                    );
                    assert_eq!(got.id, case["id"].as_str().expect("id"), "{name}");
                    assert_eq!(
                        got.resource_type,
                        case["resource_type"].as_str().expect("type"),
                        "{name}"
                    );
                    assert_eq!(got.path, case["path"].as_str().expect("path"), "{name}");
                    for (key, value) in case["query"].as_object().expect("query") {
                        assert_eq!(
                            got.query.get(key).map(String::as_str),
                            Some(value.as_str().expect("value")),
                            "{name}"
                        );
                    }
                }
                "legacy_dve" => {
                    assert_eq!(got.kind, Kind::LegacyDve, "{name}");
                    assert_eq!(
                        got.canonical,
                        case["canonical"].as_str().expect("canonical"),
                        "{name}"
                    );
                    let legacy = got.legacy_dve.expect("legacy");
                    assert_eq!(
                        legacy.form,
                        case["legacy_dve"]["form"].as_str().expect("form"),
                        "{name}"
                    );
                    assert_eq!(
                        legacy.id,
                        case["legacy_dve"]["id"].as_str().expect("id"),
                        "{name}"
                    );
                }
                "content_pointer" => {
                    assert_eq!(got.kind, Kind::ContentPointer, "{name}");
                    assert_eq!(
                        got.canonical,
                        case["canonical"].as_str().expect("canonical"),
                        "{name}"
                    );
                    let pointer = got.content_pointer.expect("pointer");
                    assert_eq!(
                        pointer.kind,
                        case["content_pointer"]["kind"].as_str().expect("kind"),
                        "{name}"
                    );
                    assert_eq!(
                        pointer.hash,
                        case["content_pointer"]["hash"].as_str().expect("hash"),
                        "{name}"
                    );
                }
                "action" => {
                    assert_eq!(got.kind, Kind::Action, "{name}");
                    assert_eq!(
                        got.canonical,
                        case["canonical"].as_str().expect("canonical"),
                        "{name}"
                    );
                    let action = got.action.expect("action");
                    assert_eq!(
                        action.name,
                        case["action"]["name"].as_str().expect("name"),
                        "{name}"
                    );
                }
                other => panic!("unknown kind {other}"),
            }
        }
    }
}
