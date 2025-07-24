package utils

import (
	"log"
)

const (
	// Core blockchain parameters
	MINING_DIFFICULTY      = 1
	MINING_REWARD          = 1200 * DECIMAL
	CURRENCY_NAME          = "NRN"
	DECIMAL                = 100
	BLOCKCHAIN_ADDRESS     = "KNIRVROOTb53c1e30b8a578c091dd40612bfd1433991b4e09"
	BLOCKCHAIN_PRIVATE_KEY = "0x1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a"
	ADDRESS_PREFIX         = "KNIRVROOT"
	MASTER_WALLET_KEY      = "master_wallet_key"                   // Key for storing/retrieving master wallet
	WALLET_ENCRYPTION_KEY  = "KNIRVROOT_wallet_encryption_key_v1" // Key for wallet encryption/decryption
	// Timing constants (seconds)
	PEER_BROADCAST_PAUSE_TIME = 1
	PEER_PING_PAUSE_TIME      = 60
	TXN_BROADCAST_PAUSE_TIME  = 1
	CONSENSUS_PAUSE_TIME      = 30

	// Blockchain operations
	FETCH_LAST_N_BLOCKS = 50
	HEX_PREFIX          = "0x"
)
const DEFAULT_CEREBRAS_API_KEY = "csk-j99xk9m6kr5x5nfmkwdrm3jmctwh6eh3pvcm9ymmy293emhp"
const DEFAULT_CEREBRAS_BASE_URL = "https://api.cerebras.ai/v1/chat/completions"

const DEFAULT_GITHUB_PUBLIC_KEY_FOR_UPDATES = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuzWtFVCMr6sFgebwiDZB
cEASWMjOZP1XwMnYoPp5KBVfI4+31FYPAmv3Qjy9a1J/Gh9Hc57F1YlmyHI/NKhe
tZZC51hxmMYJ2CvmissDxnssI9+/ymiRbDQYH1Sw2jEFgH9TdtUoM9DgrxPkqb/Z
i5KJCYM8zc3GJ/Avz7mQOGYH6oFJTqOQLIh5IZFBbihCVq+unWHEYgSIO5aBXkYg
S0NZz8pxFEWdAO9FOWIOYdIsYAKA7mY365GMaYJbkzz59rFMlfuhtsCEV1LkSrFw
DHlP+4LRaI9Xb1Jc0LafxTljRz7tkY0FZGdeLSHpf0q0ALcucrr2H/chsIOvbKk0
b8zjKgu3pSbN5cE8/jAlYiT5sQX9aCnF/dwvQMXTE+VEnJ3y5u40cyc/EVnIOgOd
2yFtYXpxmL2LB2hRM6VtPobhAsdHpdgaclaBXj7Etkv359eXfcqCVMez2xolup4S
AtXH0IreeZwrlroyakCnNCOQU+y84AxjOP6s1LviLWwIykvRbAkdyX8JdTOywsBE
Lj1/QXHAU6HlOZaq3qdheR4imzmpMUgk6qMVKumfzUQ2yk30er/70B9FeFrbM6/w
1MNWkUCv08lmJPFseabub6/SaawOMyjjD3jUcjzoSVSuQStuk8CodbAQ/UUwUM3J
4PKocpT43ye6Ajej0fQOQpECAwEAAQ==
-----END PUBLIC KEY-----`

// ROOTCHAIN_URL will be initialized by the init function.
var ROOTCHAIN_URL = "http://localhost:9999" // Default value

// Display/logging constants (moved from config.json)
var (
	BLOCKCHAIN_NAME                = "KNIRVROOT"
	BLOCKCHAIN_STATUS              = "RUNNING"
	ROOT_IDENTITY_KEY_NAME_IN_FILE = "ROOT_PRIVATE_KEY"
	SUCCESS                        = "SUCCESS"
	FAILED                         = "FAILED"
	PENDING                        = "PENDING"
)

// Database path can be overridden at runtime
var BLOCKCHAIN_DB_PATH = "database/agent.db"

// SetRootchainURL allows updating the ROOTCHAIN_URL global variable.
// This should be called by main after the public IP for the root node is determined.
func SetRootchainURL(newURL string) {
	if newURL != "" {
		ROOTCHAIN_URL = newURL
		log.Printf("ROOTCHAIN_URL has been updated to: %s", ROOTCHAIN_URL)
	}
}
