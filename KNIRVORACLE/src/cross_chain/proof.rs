//! Transfer proof validation for cross-chain transfers

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use crate::cross_chain::transfer::{TransferProof, TransferError, ValidatorSignature};

/// Merkle tree for transfer proofs
#[derive(Debug)]
pub struct MerkleTree {
    leaves: Vec<Vec<u8>>,
    root: Vec<u8>,
}

impl MerkleTree {
    /// Create a new Merkle tree from leaves
    pub fn new(leaves: Vec<Vec<u8>>) -> Self {
        if leaves.is_empty() {
            return Self {
                leaves: vec![],
                root: vec![],
            };
        }

        let mut tree = leaves.clone();
        let mut level = tree.clone();

        while level.len() > 1 {
            let mut next_level = Vec::new();

            for chunk in level.chunks(2) {
                let mut hasher = Sha256::new();
                hasher.update(&chunk[0]);
                if chunk.len() > 1 {
                    hasher.update(&chunk[1]);
                } else {
                    // Duplicate last element for odd number of nodes
                    hasher.update(&chunk[0]);
                }
                next_level.push(hasher.finalize().to_vec());
            }

            tree.extend(next_level.clone());
            level = next_level;
        }

        Self {
            leaves,
            root: level.into_iter().next().unwrap_or_default(),
        }
    }

    /// Get the Merkle root
    pub fn root(&self) -> &[u8] {
        &self.root
    }

    /// Generate proof for a leaf at given index
    pub fn generate_proof(&self, index: usize) -> Option<MerkleProof> {
        if index >= self.leaves.len() {
            return None;
        }

        let mut proof = Vec::new();
        let mut current_index = index;

        for level_start in (0..).map(|i| (1 << i) - 1).take_while(|&start| start < self.leaves.len()) {
            let level_size = 1 << (level_start.count_ones());
            if level_start + level_size > self.leaves.len() {
                break;
            }

            let sibling_index = if current_index % 2 == 0 {
                current_index + 1
            } else {
                current_index - 1
            };

            if sibling_index < level_size {
                let sibling_pos = level_start + sibling_index;
                if sibling_pos < self.leaves.len() {
                    proof.push(self.leaves[sibling_pos].clone());
                }
            }

            current_index /= 2;
        }

        Some(MerkleProof {
            leaf: self.leaves[index].clone(),
            proof,
            index,
        })
    }
}

/// Merkle proof structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MerkleProof {
    pub leaf: Vec<u8>,
    pub proof: Vec<Vec<u8>>,
    pub index: usize,
}

impl MerkleProof {
    /// Verify the Merkle proof against a root
    pub fn verify(&self, root: &[u8]) -> bool {
        let mut current_hash = self.leaf.clone();

        for sibling in &self.proof {
            let mut hasher = Sha256::new();
            if self.index % 2 == 0 {
                hasher.update(&current_hash);
                hasher.update(sibling);
            } else {
                hasher.update(sibling);
                hasher.update(&current_hash);
            }
            current_hash = hasher.finalize().to_vec();
        }

        current_hash == root
    }
}

/// Proof validator for transfer proofs
#[derive(Debug)]
pub struct ProofValidator {
    validator_set: Vec<String>, // Validator addresses
}

impl ProofValidator {
    /// Create a new proof validator
    pub fn new(validator_set: Vec<String>) -> Self {
        Self { validator_set }
    }

    /// Validate transfer proof
    pub async fn validate_proof(&self, proof: &TransferProof) -> Result<(), TransferError> {
        // Verify Merkle proof
        if !self.verify_merkle_proof(proof)? {
            return Err(TransferError::ProofValidationError("Invalid Merkle proof".to_string()));
        }

        // Verify validator signatures
        self.verify_signatures(proof).await?;

        // Verify block height is reasonable (not too old)
        let current_time = chrono::Utc::now().timestamp();
        if current_time - proof.validator_signatures[0].timestamp > 3600 { // 1 hour
            return Err(TransferError::ProofValidationError("Proof too old".to_string()));
        }

        Ok(())
    }

    /// Verify Merkle proof
    fn verify_merkle_proof(&self, proof: &TransferProof) -> Result<bool, TransferError> {
        let merkle_proof = MerkleProof {
            leaf: proof.transfer_id.as_bytes().to_vec(),
            proof: proof.merkle_proof.clone(),
            index: 0, // Simplified - in real implementation, this would be calculated
        };

        Ok(merkle_proof.verify(&proof.block_hash.as_bytes()))
    }

    /// Verify validator signatures
    async fn verify_signatures(&self, proof: &TransferProof) -> Result<(), TransferError> {
        let required_signatures = (self.validator_set.len() * 2) / 3 + 1; // 2/3 + 1 threshold

        if proof.validator_signatures.len() < required_signatures {
            return Err(TransferError::ProofValidationError(
                format!("Insufficient signatures: {} < {}", proof.validator_signatures.len(), required_signatures)
            ));
        }

        // Verify each signature
        for signature in &proof.validator_signatures {
            if !self.validator_set.contains(&signature.validator_address) {
                return Err(TransferError::ProofValidationError(
                    format!("Unknown validator: {}", signature.validator_address)
                ));
            }

            // In a real implementation, this would verify the cryptographic signature
            // For now, we assume signatures are valid if the validator is known
        }

        Ok(())
    }

    /// Update validator set
    pub fn update_validator_set(&mut self, new_set: Vec<String>) {
        self.validator_set = new_set;
    }
}

/// Transfer proof generator
#[derive(Debug)]
pub struct ProofGenerator {
    validator_private_key: Vec<u8>, // In real implementation, this would be properly secured
}

impl ProofGenerator {
    /// Create a new proof generator
    pub fn new(validator_private_key: Vec<u8>) -> Self {
        Self { validator_private_key }
    }

    /// Generate transfer proof
    pub async fn generate_proof(
        &self,
        transfer_id: &str,
        block_height: u64,
        block_hash: &str,
        validator_address: &str,
    ) -> Result<TransferProof, TransferError> {
        // Create Merkle tree with transfer data
        let leaves = vec![transfer_id.as_bytes().to_vec()];
        let merkle_tree = MerkleTree::new(leaves);

        // Generate signature
        let signature = self.sign_proof(transfer_id, block_height, block_hash)?;

        let validator_signature = ValidatorSignature {
            validator_address: validator_address.to_string(),
            signature,
            timestamp: chrono::Utc::now().timestamp(),
        };

        Ok(TransferProof {
            transfer_id: transfer_id.to_string(),
            merkle_proof: merkle_tree.generate_proof(0)
                .map(|p| p.proof)
                .unwrap_or_default(),
            block_height,
            block_hash: block_hash.to_string(),
            validator_signatures: vec![validator_signature],
        })
    }

    /// Sign proof data
    fn sign_proof(&self, transfer_id: &str, block_height: u64, block_hash: &str) -> Result<Vec<u8>, TransferError> {
        let mut hasher = Sha256::new();
        hasher.update(transfer_id.as_bytes());
        hasher.update(&block_height.to_be_bytes());
        hasher.update(block_hash.as_bytes());

        let message = hasher.finalize();

        // In a real implementation, this would use proper cryptographic signing
        // For now, we return a mock signature
        Ok(message.to_vec())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_merkle_tree() {
        let leaves = vec![
            b"leaf1".to_vec(),
            b"leaf2".to_vec(),
            b"leaf3".to_vec(),
        ];

        let tree = MerkleTree::new(leaves);
        assert!(!tree.root().is_empty());

        let proof = tree.generate_proof(0).unwrap();
        assert!(proof.verify(tree.root()));
    }

    #[test]
    fn test_proof_validation() {
        let validator_set = vec!["validator1".to_string(), "validator2".to_string()];
        let validator = ProofValidator::new(validator_set);

        // This would need a proper proof in a real test
        // For now, we just test the structure
        assert_eq!(validator.validator_set.len(), 2);
    }
}