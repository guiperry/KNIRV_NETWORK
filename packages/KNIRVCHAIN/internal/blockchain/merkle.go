package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

const (
	merkleLeafDomain   byte = 0x00
	merkleParentDomain byte = 0x01
)

var emptyTxMerkleRoot = sha256.Sum256([]byte("knirv-tx-merkle-empty"))
var blockHashDomain = []byte("knirv-block-hash-v1")

// BlockCoreDigest commits the non-transaction, non-author block metadata. The
// final block hash separately incorporates TxMerkleRoot and the normalized
// proposer digest so the validity circuit can prove both bindings.
func BlockCoreDigest(b *Block) [sha256.Size]byte {
	buf := appendBytes(nil, []byte("knirv-block-core-v1"))
	if b == nil {
		return sha256.Sum256(append(buf, 0))
	}
	buf = append(buf, 1)
	buf = appendUint64(buf, b.BlockNumber)
	buf = appendBytes(buf, b.PrevHash)
	buf = appendUint64(buf, uint64(b.Timestamp))
	buf = appendUint64(buf, uint64(int64(b.Nonce)))
	buf = appendUint64(buf, b.Header.Height)
	buf = appendUint64(buf, uint64(b.Header.Version))
	return sha256.Sum256(buf)
}

func BlockAuthorDigest(author string) [sha256.Size]byte {
	return sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(author))))
}

// BlockHashFromParts is shared conceptually with the PLONK circuit:
// SHA256(domain || coreDigest || txRoot || authorDigest).
func BlockHashFromParts(core, txRoot, author [sha256.Size]byte) [sha256.Size]byte {
	preimage := make([]byte, 0, len(blockHashDomain)+3*sha256.Size)
	preimage = append(preimage, blockHashDomain...)
	preimage = append(preimage, core[:]...)
	preimage = append(preimage, txRoot[:]...)
	preimage = append(preimage, author[:]...)
	return sha256.Sum256(preimage)
}

func ContentBlockHash(b *Block) [sha256.Size]byte {
	return BlockHashFromParts(BlockCoreDigest(b), TxMerkleRoot(b.Transactions), BlockAuthorDigest(b.ProposerAddress))
}

// TxMerkleRoot commits to the canonical contents and ordering of every
// transaction in a block. Odd-width levels duplicate their final node, matching
// the chain's existing validationStorageRoot convention. An empty block uses a
// stable domain-specific root.
func TxMerkleRoot(txs []*Transaction) [sha256.Size]byte {
	if len(txs) == 0 {
		return emptyTxMerkleRoot
	}
	level := make([][sha256.Size]byte, len(txs))
	for i, tx := range txs {
		level[i] = txMerkleLeafHash(canonicalTransactionBytes(tx))
	}
	for len(level) > 1 {
		next := make([][sha256.Size]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, txMerkleParentHash(level[i], right))
		}
		level = next
	}
	return level[0]
}

// TxMerkleLeafHashes returns the domain-separated transaction leaf hashes used
// by TxMerkleRoot. It is exported for the checkpoint validity prover: the
// PLONK circuit recomputes the tree from these private leaves instead of
// accepting a caller-supplied transaction root.
func TxMerkleLeafHashes(txs []*Transaction) [][sha256.Size]byte {
	leaves := make([][sha256.Size]byte, len(txs))
	for i, tx := range txs {
		leaves[i] = txMerkleLeafHash(canonicalTransactionBytes(tx))
	}
	return leaves
}

// BlockStateRoot extends the cross-block accumulator with the previous
// accumulator, this block's transaction root, and this block's hash:
// SHA256(0x01 || prevAccum || txRoot || blockHash).
func BlockStateRoot(b *Block, prevAccum [sha256.Size]byte) [sha256.Size]byte {
	txRoot := TxMerkleRoot(nil)
	var blockHash []byte
	if b != nil {
		txRoot = TxMerkleRoot(b.Transactions)
		blockHash = b.BlockHash
	}
	preimage := make([]byte, 1, 1+sha256.Size+sha256.Size+len(blockHash))
	preimage[0] = merkleParentDomain
	preimage = append(preimage, prevAccum[:]...)
	preimage = append(preimage, txRoot[:]...)
	preimage = append(preimage, blockHash...)
	return sha256.Sum256(preimage)
}

// PopulateBlockMerkleRoots writes the per-block transaction root and the
// cross-block accumulator into the header before the block is persisted.
func PopulateBlockMerkleRoots(b *Block, prevAccum [sha256.Size]byte) {
	if b == nil {
		return
	}
	b.Header.TxRoot = TxMerkleRoot(b.Transactions)
	b.Header.AccumRoot = BlockStateRoot(b, prevAccum)
}

// RecomputeAccumRoot independently reconstructs the accumulator from stored
// block contents. It deliberately ignores stored header roots so callers can
// detect corruption or migrate pre-upgrade blocks whose root fields are zero.
func RecomputeAccumRoot(blocks []*Block) [sha256.Size]byte {
	var accumulator [sha256.Size]byte
	for _, block := range blocks {
		accumulator = BlockStateRoot(block, accumulator)
	}
	return accumulator
}

func txMerkleLeafHash(data []byte) [sha256.Size]byte {
	preimage := make([]byte, 1, 1+len(data))
	preimage[0] = merkleLeafDomain
	preimage = append(preimage, data...)
	return sha256.Sum256(preimage)
}

func txMerkleParentHash(left, right [sha256.Size]byte) [sha256.Size]byte {
	var preimage [1 + 2*sha256.Size]byte
	preimage[0] = merkleParentDomain
	copy(preimage[1:1+sha256.Size], left[:])
	copy(preimage[1+sha256.Size:], right[:])
	return sha256.Sum256(preimage[:])
}

// canonicalTransactionBytes is a versioned, length-delimited encoding of every
// consensus-relevant transaction field. It excludes DeepCopy and BlockHash:
// DeepCopy is an in-memory implementation detail, while BlockHash would create a
// cycle between the transaction root and the enclosing block hash.
func canonicalTransactionBytes(tx *Transaction) []byte {
	buf := make([]byte, 0, 256)
	buf = appendBytes(buf, []byte("knirv-tx-merkle-v1"))
	if tx == nil {
		return append(buf, 0)
	}
	buf = append(buf, 1)
	buf = appendString(buf, tx.From)
	buf = appendString(buf, tx.To)
	buf = appendUint64(buf, tx.Value)
	buf = appendBytes(buf, tx.Data)
	buf = appendString(buf, tx.Status)
	buf = appendUint64(buf, uint64(tx.Timestamp))
	buf = appendString(buf, tx.TransactionHash)
	buf = appendString(buf, tx.PublicKey)
	buf = appendBytes(buf, tx.Signature)
	buf = appendUint64(buf, tx.Fee)
	buf = appendString(buf, tx.Type)
	if tx.Verified {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	return buf
}

func appendString(dst []byte, value string) []byte {
	return appendBytes(dst, []byte(value))
}

func appendBytes(dst, value []byte) []byte {
	dst = appendUint64(dst, uint64(len(value)))
	return append(dst, value...)
}

func appendUint64(dst []byte, value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return append(dst, encoded[:]...)
}
