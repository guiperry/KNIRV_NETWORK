package constants

// Constants for the blockchain
const (
	BLOCKCHAIN_NAME           = "KNIRVCHAIN"
	HEX_PREFIX                = "0x"
	SUCCESS                   = "success"
	FAILED                    = "failed"
	PENDING                   = "pending"
	MINING_DIFFICULTY         = 3
	MINING_REWARD             = 1200 * DECIMAL
	CURRENCY_NAME             = "nrn"
	DECIMAL                   = 100
	BLOCKCHAIN_ADDRESS        = "KNIRVCHAIN_Faucet"
	BLOCKCHAIN_KEY            = "blockchain_key"
	ADDRESS_PREFIX            = "knirvchain"
	TXN_VERIFICATION_SUCCESS  = "verification_success"
	TXN_VERIFICATION_FAILURE  = "verification_failure"
	BLOCKCHAIN_STATUS         = "RUNNING"
	PEER_BROADCAST_PAUSE_TIME = 3  // In seconds
	PEER_PING_PAUSE_TIME      = 60 // In seconds
	TXN_BROADCAST_PAUSE_TIME  = 1  // In seconds
	FETCH_LAST_N_BLOCKS       = 50
	CONSENSUS_PAUSE_TIME      = 10 // In seconds
	PEER_DISCOVERY_INTERVAL   = 30 // In seconds
	EMPTY_BLOCK_INTERVAL      = 60 // In seconds

	// Transaction origin types
	ORIGIN_PRIVATE = "nrn"   // Private transactions (nrn://)
	ORIGIN_PUBLIC  = "chain" // Public transactions (chain://)

	// P2P constants
	P2P_MAX_PEERS        = 50      // Maximum number of peers to connect to
	P2P_PING_INTERVAL    = 30      // Seconds between ping messages
	P2P_INACTIVE_TIMEOUT = 90      // Seconds before considering a peer inactive
	P2P_MAX_MESSAGE_SIZE = 1048576 // 1MB maximum message size
)

// Database path can be overridden at runtime
// This should be a path to a directory where LevelDB will store its files
var BLOCKCHAIN_DB_PATH = "database"
