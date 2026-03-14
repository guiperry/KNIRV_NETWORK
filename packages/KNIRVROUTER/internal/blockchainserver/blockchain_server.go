package blockchainserver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"KNIRVROUTER/internal/blockchain"
	constants "KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/types"
)

type BlockchainServer struct {
	Port          uint64                       `json:"port"`
	BlockchainPtr *blockchain.BlockchainStruct `json:"blockchain"`
	Server        *http.Server
	MiningLocked  bool `json:"mining_locked"`
}

func NewBlockchainServer(port uint64, blockchainPtr *blockchain.BlockchainStruct) *BlockchainServer {
	bcs := new(BlockchainServer)
	bcs.Port = port
	bcs.BlockchainPtr = blockchainPtr
	bcs.Server = &http.Server{Addr: fmt.Sprintf(":%d", bcs.Port)}
	return bcs
}

func (bcs *BlockchainServer) GetBlockchain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		blockchain := bcs.BlockchainPtr

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(blockchain); err != nil {
			http.Error(w, "Failed to marshal blocks to json", http.StatusInternalServerError)
			return
		}

	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		addr := r.URL.Query().Get("address")
		x := struct {
			Balance uint64 `json:"balance"`
		}{
			bcs.BlockchainPtr.CalculateTotalCrypto(addr),
		}

		mBalance, err := json.Marshal(x)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(mBalance)
	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetAllNonRewardedTxns(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		txnList := bcs.BlockchainPtr.GetAllTxns()
		byteSlice, err := json.Marshal(txnList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(byteSlice)
	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var transactions = bcs.BlockchainPtr.TransactionPool
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(transactions)
	if err != nil {
		http.Error(w, "Failed to marshal transactions to json", http.StatusInternalServerError)
		return
	}
}

func (bcs *BlockchainServer) SendTxnToTheBlockchain(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodPost {
		var txn = types.Transaction{}
		if err := json.NewDecoder(req.Body).Decode(&txn); err != nil {
			http.Error(w, "Invalid transaction format", http.StatusBadRequest)
			return
		}

		//Verify Transaction
		if !txn.VerifyTxn() {
			http.Error(w, "Invalid Txn Signature", http.StatusBadRequest)
			return
		}
		err := bcs.BlockchainPtr.AddTransaction(txn)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add transaction: %v", err), http.StatusInternalServerError)
			log.Printf("Failed to add transaction: %v", err)
			return
		}
		log.Println("Transaction successfully added to the pool with hash: ", txn.Hash())

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(txn); err != nil {
			http.Error(w, fmt.Sprintf("Failed to marshal transaction: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
	}
}

func CheckStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		io.WriteString(w, constants.BLOCKCHAIN_STATUS)
	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) SendPeersList(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodPost {
		peersMap, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Println("Error reading Peers")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var peersList map[string]bool
		err = json.Unmarshal(peersMap, &peersList)
		if err != nil {
			log.Println("Error Unmarshalling the Peers")

			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Replace with libp2p peer management
		// bcs.BlockchainPtr.NetworkManager.UpdatePeers(peersList)
		res := map[string]string{}
		res["status"] = "success"
		x, err := json.Marshal(res)
		if err != nil {
			log.Println("Error while marshalling the http Response to be sent.")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(x)
	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}
func (bcs *BlockchainServer) FetchLastNBlocks(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodGet {
		blocks := bcs.BlockchainPtr.Blocks
		blockchain1 := new(blockchain.BlockchainStruct)
		if len(blocks) < constants.FETCH_LAST_N_BLOCKS {
			blockchain1.Blocks = blocks
		} else {
			blockchain1.Blocks = blocks[len(blocks)-constants.FETCH_LAST_N_BLOCKS:]
		}
		blockJSON, err := json.Marshal(blockchain1.Blocks)
		if err != nil {
			http.Error(w, "Failed to marshal blocks to json", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(blockJSON)

	} else {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
	}
}

// GetChainHeaders returns just the headers of all blocks in the blockchain
func (bcs *BlockchainServer) GetChainHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
		return
	}

	// Lock to safely access the blockchain
	bcs.BlockchainPtr.Mutex.Lock()
	blocks := bcs.BlockchainPtr.Blocks
	bcs.BlockchainPtr.Mutex.Unlock()

	// Create headers from blocks
	headers := make([]blockchain.BlockHeader, len(blocks))
	for i, block := range blocks {
		headers[i] = blockchain.BlockHeader{
			BlockNumber: block.Number,
			Hash:        block.Hash(),
			PrevHash:    block.PreviousHash,
			Timestamp:   block.Time,
		}
	}

	// Marshal and return headers
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(headers); err != nil {
		http.Error(w, "Failed to marshal headers to json", http.StatusInternalServerError)
		return
	}
}

// GetBlocksRange returns a range of blocks from the blockchain
func (bcs *BlockchainServer) GetBlocksRange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
		return
	}

	// Parse start and end parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var start, end uint64
	var err error

	if startStr != "" {
		start, err = strconv.ParseUint(startStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid start parameter", http.StatusBadRequest)
			return
		}
	}

	if endStr != "" {
		end, err = strconv.ParseUint(endStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid end parameter", http.StatusBadRequest)
			return
		}
	}

	// Lock to safely access the blockchain
	bcs.BlockchainPtr.Mutex.Lock()
	blocks := bcs.BlockchainPtr.Blocks
	bcs.BlockchainPtr.Mutex.Unlock()

	// Filter blocks by range
	var filteredBlocks []*blockchain.Block
	for _, block := range blocks {
		if block.Number >= start && (end == 0 || block.Number <= end) {
			filteredBlocks = append(filteredBlocks, block)
		}
	}

	// Marshal and return blocks
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(filteredBlocks); err != nil {
		http.Error(w, "Failed to marshal blocks to json", http.StatusInternalServerError)
		return
	}
}

// GetFullChain returns the entire blockchain
func (bcs *BlockchainServer) GetFullChain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid Method", http.StatusBadRequest)
		return
	}

	// Lock to safely access the blockchain
	bcs.BlockchainPtr.Mutex.Lock()
	blocks := bcs.BlockchainPtr.Blocks
	txPool := bcs.BlockchainPtr.TransactionPool
	bcs.BlockchainPtr.Mutex.Unlock()

	// Create response data
	chainData := blockchain.PeerChainData{
		Blocks:          blocks,
		TransactionPool: txPool,
	}

	// Marshal and return chain data
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(chainData); err != nil {
		http.Error(w, "Failed to marshal chain data to json", http.StatusInternalServerError)
		return
	}
}

func (bcs *BlockchainServer) Start() {
	http.HandleFunc("/", bcs.GetBlockchain)
	http.HandleFunc("/balance", bcs.GetBalance)
	http.HandleFunc("/blocks", bcs.GetBlockchain)
	http.HandleFunc("/get_all_non_rewarded_txns", bcs.GetAllNonRewardedTxns)
	http.HandleFunc("/send_txn", bcs.SendTxnToTheBlockchain)
	http.HandleFunc("/transactions", bcs.handleGetTransactions)
	http.HandleFunc("/send_peers_list", bcs.SendPeersList)
	http.HandleFunc("/check_status", CheckStatus)
	http.HandleFunc("/fetch_last_n_blocks", bcs.FetchLastNBlocks)

	// Add new endpoints for the header-first approach
	http.HandleFunc("/chain_headers", bcs.GetChainHeaders)
	http.HandleFunc("/blocks_range", bcs.GetBlocksRange)
	http.HandleFunc("/full_chain", bcs.GetFullChain)

	log.Println("Launching webserver at port :", bcs.Port)
	log.Println("Blockchain server starting with mining difficulty:", constants.MINING_DIFFICULTY)
	log.Println("Mining reward set to:", constants.MINING_REWARD)
	log.Println("Database path:", constants.BLOCKCHAIN_DB_PATH)
	log.Println("Checking database directory:", constants.BLOCKCHAIN_DB_PATH)

	// Create database directory if it doesn't exist
	dbDir := constants.BLOCKCHAIN_DB_PATH
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Println("Creating database directory:", dbDir)
		os.MkdirAll(dbDir, 0755)
	}

	go func() {
		if err := bcs.Server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Failed to start blockchain server: %v", err)
		}
	}()
}

func (bcs *BlockchainServer) Stop() {
	if err := bcs.Server.Shutdown(nil); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
}

func init() {
	log.SetPrefix(constants.BLOCKCHAIN_NAME + ":")
}
