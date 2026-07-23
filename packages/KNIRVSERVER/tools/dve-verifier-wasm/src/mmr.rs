use sha2::{Digest, Sha256};

pub type Hash = [u8; 32];
const LEAF: u8 = 0x00;
const PARENT: u8 = 0x01;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Side {
    Left,
    Right,
}
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct Sibling {
    pub hash: Hash,
    pub side: Side,
}
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Proof {
    pub leaf_index: u64,
    pub tree_size: u64,
    pub path: Vec<Sibling>,
}

pub fn empty_root() -> Hash {
    Sha256::digest(b"knirv-mmr-empty").into()
}
pub fn leaf_hash(data: &[u8]) -> Hash {
    let mut h = Sha256::new();
    h.update([LEAF]);
    h.update(data);
    h.finalize().into()
}
pub fn parent_hash(left: Hash, right: Hash) -> Hash {
    let mut h = Sha256::new();
    h.update([PARENT]);
    h.update(left);
    h.update(right);
    h.finalize().into()
}

pub fn verify_proof(root: Hash, leaf: Hash, proof: &Proof, size: u64) -> bool {
    if proof.tree_size != size || proof.leaf_index >= size {
        return false;
    }
    let Some(sides) = proof_sides(size, proof.leaf_index) else {
        return false;
    };
    if sides.len() != proof.path.len() {
        return false;
    }
    proof
        .path
        .iter()
        .zip(sides)
        .try_fold(leaf, |node, (sibling, side)| {
            if sibling.side != side {
                return None;
            }
            Some(match side {
                Side::Left => parent_hash(sibling.hash, node),
                Side::Right => parent_hash(node, sibling.hash),
            })
        })
        .is_some_and(|node| node == root)
}

fn proof_sides(size: u64, mut index: u64) -> Option<Vec<Side>> {
    if size == 0 || index >= size {
        return None;
    }
    let peaks = size.count_ones() as usize;
    let mut peak = 0usize;
    let mut located = None;
    for height in (0..64u32).rev() {
        if size & (1u64 << height) == 0 {
            continue;
        }
        let count = 1u64 << height;
        if index < count {
            located = Some((height, index));
            break;
        }
        index -= count;
        peak += 1;
    }
    let (height, relative) = located?;
    let mut out = Vec::new();
    for level in 0..height {
        out.push(if relative & (1u64 << level) == 0 {
            Side::Right
        } else {
            Side::Left
        });
    }
    if peak > 0 {
        out.push(Side::Left)
    }
    for _ in peak + 1..peaks {
        out.push(Side::Right)
    }
    Some(out)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde::Deserialize;
    const GOLDEN: &str =
        include_str!("../../../../KNIRVORACLE/internal/oracle/mmr/testdata/mmr_golden.json");
    #[derive(Deserialize)]
    struct File {
        version: u64,
        algorithm: String,
        bagging: String,
        empty_root: String,
        tree_size: u64,
        root: String,
        leaves: Vec<Leaf>,
    }
    #[derive(Deserialize)]
    struct Leaf {
        index: u64,
        data_utf8: String,
        leaf_hash: String,
        path: Vec<Item>,
    }
    #[derive(Deserialize)]
    struct Item {
        hash: String,
        side: String,
    }
    fn hash(s: &str) -> Hash {
        let mut out = [0; 32];
        for (i, b) in out.iter_mut().enumerate() {
            *b = u8::from_str_radix(&s[i * 2..i * 2 + 2], 16).unwrap()
        }
        out
    }
    #[test]
    fn shared_golden_vectors() {
        let f: File = serde_json::from_str(GOLDEN).unwrap();
        assert_eq!(
            (f.version, f.algorithm.as_str(), f.bagging.as_str()),
            (1, "sha256", "left-fold")
        );
        assert_eq!(empty_root(), hash(&f.empty_root));
        let root = hash(&f.root);
        for leaf in f.leaves {
            let digest = hash(&leaf.leaf_hash);
            assert_eq!(leaf_hash(leaf.data_utf8.as_bytes()), digest);
            let path = leaf
                .path
                .into_iter()
                .map(|i| Sibling {
                    hash: hash(&i.hash),
                    side: if i.side == "left" {
                        Side::Left
                    } else {
                        Side::Right
                    },
                })
                .collect();
            assert!(verify_proof(
                root,
                digest,
                &Proof {
                    leaf_index: leaf.index,
                    tree_size: f.tree_size,
                    path
                },
                f.tree_size
            ));
        }
    }
}
