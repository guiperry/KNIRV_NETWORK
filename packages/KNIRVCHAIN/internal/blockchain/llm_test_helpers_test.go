package blockchain

import (
	"path/filepath"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/stretchr/testify/require"

	"KNIRVCHAIN/internal/database"
	pb "KNIRVCHAIN/internal/protocol/proto"

	"google.golang.org/protobuf/proto"
)

const testChainID = "knirv-1"

// testSigner holds a secp256k1 keypair whose bech32 address doubles as the
// sender and the model owner, which the hardened signature verification now
// requires for LLM rooting transactions.
type testSigner struct {
	privateKey []byte
	address    string
}

func newTestSigner(t *testing.T) *testSigner {
	t.Helper()
	key, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	pub := ethcrypto.CompressPubkey(&key.PublicKey)
	address, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	require.NoError(t, err)
	privateKey := make([]byte, 32)
	key.D.FillBytes(privateKey)
	return &testSigner{privateKey: privateKey, address: address}
}

// newSignedLLMRooting builds and signs an LLM rooting transaction whose sender
// (From) matches the model owner on the signer address.
func (s *testSigner) newSignedLLMRooting(t *testing.T, llmData LLMRootingData, fee uint64) *Transaction {
	t.Helper()
	if llmData.ModelOwner == "" {
		llmData.ModelOwner = s.address
	}
	require.Equal(t, s.address, llmData.ModelOwner, "model owner must equal signer address")
	tx, err := NewLLMRootingTransaction(s.address, llmData, fee)
	require.NoError(t, err)
	return s.signTransaction(t, tx)
}

// signTransaction signs an already-built transaction and wires the canonical
// SIGN_MODE_DIRECT fields (PublicKey, BodyBytes, AuthInfoBytes, Signature,
// ChainID) that VerifySignature/VerifyTxn require.
func (s *testSigner) signTransaction(t *testing.T, tx *Transaction) *Transaction {
	t.Helper()
	signed, err := knirvsigning.SignTransaction(s.privateKey, knirvsigning.SignRequest{
		Action: knirvsigning.Action{
			Action:        tx.Type,
			Sender:        tx.From,
			Recipient:     tx.To,
			Amount:        tx.Value,
			Payload:       tx.Data,
			TimestampUnix: tx.Timestamp,
		},
		ChainID:       testChainID,
		AccountNumber: tx.AccountNumber,
		Sequence:      tx.Sequence,
		Fee: knirvsigning.Fee{
			Denom:    "unrn",
			Amount:   "0",
			GasLimit: tx.Fee,
			Payer:    tx.From,
		},
	})
	require.NoError(t, err)
	wire := signed.Wire()
	tx.PublicKey = wire.PublicKey
	tx.BodyBytes = wire.BodyBytes
	tx.AuthInfoBytes = wire.AuthInfoBytes
	tx.Signature = signed.Signatures[0]
	tx.ChainID = testChainID
	// With the canonical signing fields populated, Hash() switches to the
	// signed TxRaw hash; align the stored hash so VerifySignature's final
	// "transaction hash does not match signed TxRaw" check passes.
	tx.TransactionHash = tx.Hash()
	return tx
}

// llmRootingDataFromTx decodes the protobuf payload of an LLM rooting transaction.
func llmRootingDataFromTx(t *testing.T, tx *Transaction) LLMRootingData {
	t.Helper()
	var protoData pb.LLMRootingDataProto
	require.NoError(t, proto.Unmarshal(tx.Data, &protoData))
	return LLMRootingData{
		ModelName:   protoData.ModelName,
		ModelOwner:  protoData.ModelOwner,
		APIEndpoint: protoData.ApiEndpoint,
		MetadataCID: protoData.MetadataCid,
		CMU:         protoData.Cmu,
	}
}

// createTestBlockchain builds a blockchain with a live LevelDB connection so
// the hardened block-addition path (which persists balances) can run.
func createTestBlockchain(t *testing.T) *BlockchainStruct {
	t.Helper()
	dbDir := filepath.Join(t.TempDir(), "testdb")
	db, err := database.NewLevelDB(dbDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	bc := &BlockchainStruct{
		TransactionPool:        []*Transaction{},
		Blocks:                 []*Block{},
		ChainAddress:           "test-chain",
		Reflections:            make(map[string]bool),
		MiningLocked:           false,
		OwnerAddress:           "test-owner",
		WalletAddress:          "test-wallet",
		db:                     db,
		txnSignal:              make(chan struct{}, 1),
		isActivelyMining:       false,
		NetworkAuthors:         make(map[string]bool),
		PoAuDEnabled:           false,
		TransactionPoolManager: NewTransactionPoolManager(nil),
	}

	// Add genesis block (mirrors the deterministic production genesis at
	// blockchain_struct.go:704-708: BlockHash must equal Hash() so AddBlock's
	// PrevHash comparison against Blocks[0].Hash() succeeds)
	genesisBlock := NewBlock([]byte{}, 0, 0)
	genesisBlock.ProposerAddress = "genesis"
	genesisBlock.BlockHash = genesisBlock.Hash()
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// Set self-reference for transaction pool manager
	bc.TransactionPoolManager.blockchain = bc

	// Add test validator
	bc.AddNetworkAuthor("validator1")

	return bc
}

// rootModel is a convenience wrapper: create the blockchain, create and sign
// an LLM rooting transaction for the owner, add it to the pool, propose a
// PoAuD block, ensure it meets the mining difficulty, and add it to the chain.
func rootModel(t *testing.T, bc *BlockchainStruct, signer *testSigner, llmData LLMRootingData) *Transaction {
	t.Helper()
	tx := signer.newSignedLLMRooting(t, llmData, 0)
	require.NoError(t, bc.AddTransactionToTransactionPool(tx))

	block, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block)
	block.MineBlock()

	require.NoError(t, bc.AddBlock(block))
	return tx
}
