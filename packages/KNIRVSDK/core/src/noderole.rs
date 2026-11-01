//! Node role resolution: what a KNIRV node is (root, bootnode or client) and
//! every capability that follows from it. Pure and deterministic: key-file
//! discovery results go in, a decision comes out. The shared vectors are
//! `testvectors/noderole/roles_v1.json`.
//!
//! Rules (D-R1..D-R4):
//!
//! ```text
//! root.key found   kind=root      chain=Root      oracle=active   validator=false
//! boot.key only    kind=bootnode  chain=Bootnode  oracle=standby  validator=true
//! neither          kind=client    chain=Client    oracle=off      validator=false
//! ```
//!
//! root.key wins when both exist. A privileged role is never inherited without
//! its key file; an existing-but-locked key keeps its kind and marks the node
//! degraded. Overrides may only lower privilege.

use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum Kind {
    Root,
    Bootnode,
    Client,
}

impl Kind {
    fn as_str(self) -> &'static str {
        match self {
            Kind::Root => "root",
            Kind::Bootnode => "bootnode",
            Kind::Client => "client",
        }
    }
}

#[derive(Clone, Copy, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum OracleMode {
    Active,
    Standby,
    Off,
}

impl OracleMode {
    fn as_str(self) -> &'static str {
        match self {
            OracleMode::Active => "active",
            OracleMode::Standby => "standby",
            OracleMode::Off => "off",
        }
    }
    fn rank(self) -> u8 {
        match self {
            OracleMode::Active => 2,
            OracleMode::Standby => 1,
            OracleMode::Off => 0,
        }
    }
    fn parse(value: &str) -> Option<Self> {
        match value.trim().to_lowercase().as_str() {
            "active" => Some(Self::Active),
            "standby" => Some(Self::Standby),
            "off" => Some(Self::Off),
            _ => None,
        }
    }
}

pub const CHAIN_ROOT: &str = "Root";
pub const CHAIN_BOOTNODE: &str = "Bootnode";
pub const CHAIN_PEER: &str = "Peer";
pub const CHAIN_CLIENT: &str = "Client";
pub const SIGNER_ROOT: &str = "root-owner";
pub const SIGNER_BOOTNODE: &str = "bootnode-master";

/// Privilege order of KNIRVCHAIN roles. Peer sits between Bootnode and Client.
fn chain_rank(role: &str) -> Option<u8> {
    match role {
        CHAIN_ROOT => Some(3),
        CHAIN_BOOTNODE => Some(2),
        CHAIN_PEER => Some(1),
        CHAIN_CLIENT => Some(0),
        _ => None,
    }
}

fn canonical_chain_role(role: &str) -> Option<&'static str> {
    match role.trim().to_lowercase().as_str() {
        "root" => Some(CHAIN_ROOT),
        "bootnode" => Some(CHAIN_BOOTNODE),
        "peer" => Some(CHAIN_PEER),
        "client" => Some(CHAIN_CLIENT),
        _ => None,
    }
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Override {
    #[serde(default)]
    pub chain_role: String,
    #[serde(default)]
    pub oracle_mode: String,
    #[serde(default)]
    pub validator: Option<bool>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Inputs {
    #[serde(default)]
    pub root_key_path: String,
    #[serde(default)]
    pub boot_key_path: String,
    #[serde(default)]
    pub root_unlocked: bool,
    #[serde(default)]
    pub boot_unlocked: bool,
    #[serde(default, rename = "override")]
    pub overrides: Override,
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
pub struct Resolution {
    pub kind: Kind,
    pub key_path: String,
    pub chain_role: String,
    pub oracle: OracleMode,
    pub validator: bool,
    pub signer: String,
    pub degraded: bool,
    pub reason: String,
    /// Overrides rejected for raising privilege or naming an unknown value; each
    /// entry is a complete sentence suitable for a WARN log.
    pub ignored: Vec<String>,
}

pub fn resolve(input: &Inputs) -> Resolution {
    let mut r = if !input.root_key_path.is_empty() {
        // root.key wins over boot.key, as it always has.
        let mut r = Resolution {
            kind: Kind::Root,
            key_path: input.root_key_path.clone(),
            chain_role: CHAIN_ROOT.into(),
            oracle: OracleMode::Active,
            validator: false, // D-R4: root validator mode stays off unless overridden
            signer: SIGNER_ROOT.into(),
            degraded: false,
            reason: String::new(),
            ignored: Vec::new(),
        };
        if !input.root_unlocked {
            r.degraded = true;
            r.reason = "root.key is present but locked; the Oracle starts degraded".into();
        }
        r
    } else if !input.boot_key_path.is_empty() {
        let mut r = Resolution {
            kind: Kind::Bootnode,
            key_path: input.boot_key_path.clone(),
            chain_role: CHAIN_BOOTNODE.into(),
            oracle: OracleMode::Standby,
            validator: true, // D-R1: a boot.key is what makes a node a validator
            signer: SIGNER_BOOTNODE.into(),
            degraded: false,
            reason: String::new(),
            ignored: Vec::new(),
        };
        if !input.boot_unlocked {
            r.degraded = true;
            r.reason = "boot.key is present but locked; standby replication is paused".into();
        }
        r
    } else {
        Resolution {
            kind: Kind::Client,
            key_path: String::new(),
            chain_role: CHAIN_CLIENT.into(),
            oracle: OracleMode::Off,
            validator: false,
            signer: String::new(),
            degraded: false,
            reason: String::new(),
            ignored: Vec::new(),
        }
    };
    apply(&mut r, &input.overrides);
    r
}

/// Applies an override to an existing resolution. A child process that read a
/// resolution from its environment uses this so configuration can still lower
/// what its parent decided.
pub fn with_override(mut resolution: Resolution, o: &Override) -> Resolution {
    apply(&mut resolution, o);
    resolution
}

fn apply(r: &mut Resolution, o: &Override) {
    if !o.chain_role.is_empty() {
        let have = chain_rank(&r.chain_role).unwrap_or(0);
        match canonical_chain_role(&o.chain_role) {
            None => r.ignored.push(format!(
                "chain.role {:?} is not a KNIRVCHAIN role (root, bootnode, peer, client); ignored",
                o.chain_role
            )),
            Some(want) if chain_rank(want).unwrap_or(0) > have => r.ignored.push(format!(
                "chain.role {:?} would raise this {} node's role from {}; overrides may only lower privilege, so it was ignored",
                o.chain_role,
                r.kind.as_str(),
                r.chain_role
            )),
            Some(want) => r.chain_role = want.into(),
        }
    }

    if !o.oracle_mode.is_empty() {
        match OracleMode::parse(&o.oracle_mode) {
            None => r.ignored.push(format!(
                "oracle mode {:?} is not active, standby or off; ignored",
                o.oracle_mode
            )),
            Some(want) if want.rank() > r.oracle.rank() => r.ignored.push(format!(
                "oracle mode {:?} would raise this {} node's mode from {}; overrides may only lower privilege, so it was ignored",
                o.oracle_mode,
                r.kind.as_str(),
                r.oracle.as_str()
            )),
            Some(OracleMode::Off) if r.oracle != OracleMode::Off => {
                // Turning the Oracle off also clears the validator flag: a node
                // with no Oracle cannot validate.
                r.oracle = OracleMode::Off;
                r.validator = false;
            }
            Some(want) => r.oracle = want,
        }
    }

    match o.validator {
        Some(true) if !r.validator => r.ignored.push(format!(
            "validator mode cannot be enabled on a {} node by configuration; ignored",
            r.kind.as_str()
        )),
        Some(false) => r.validator = false,
        _ => {}
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::Value;

    #[test]
    fn matches_the_shared_vectors() {
        let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../testvectors/noderole/roles_v1.json");
        let file: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
        for case in file["cases"].as_array().unwrap() {
            let name = case["name"].as_str().unwrap();
            let inputs: Inputs = serde_json::from_value(case["inputs"].clone()).unwrap();
            let got = resolve(&inputs);
            let want = &case["want"];
            assert_eq!(got.kind.as_str(), want["kind"], "{name}");
            assert_eq!(got.chain_role, want["chain_role"].as_str().unwrap(), "{name}");
            assert_eq!(got.oracle.as_str(), want["oracle"], "{name}");
            assert_eq!(got.validator, want["validator"].as_bool().unwrap(), "{name}");
            assert_eq!(got.signer, want["signer"].as_str().unwrap(), "{name}");
            assert_eq!(got.degraded, want["degraded"].as_bool().unwrap(), "{name}");
            assert_eq!(got.ignored.len() as u64, want.get("ignored").and_then(Value::as_u64).unwrap_or(0), "{name}: {:?}", got.ignored);
        }
    }

    #[test]
    fn overrides_only_lower() {
        let root = resolve(&Inputs { root_key_path: "/k".into(), root_unlocked: true, ..Default::default() });
        let up = with_override(root.clone(), &Override { oracle_mode: "standby".into(), ..Default::default() });
        assert_eq!(up.oracle, OracleMode::Standby);
        let client = resolve(&Inputs::default());
        let raised = with_override(client, &Override { chain_role: "root".into(), ..Default::default() });
        assert_eq!(raised.chain_role, CHAIN_CLIENT);
        assert_eq!(raised.ignored.len(), 1);
    }
}
