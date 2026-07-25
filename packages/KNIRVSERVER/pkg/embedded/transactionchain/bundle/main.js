// miner.v11.js
const express = require('express');
const app = express();
const crypto = require('crypto'); //Import crypto
const bodyParser = require('body-parser');
const sqlite3 = require('sqlite3').verbose();
const fs = require('node:fs');
const path = require('node:path');

// Config
const DEFAULT_DIFFICULTY = 0;
const DIFFICULTY = process.env.BLOCK_DIFFICULTY ? parseInt(process.env.BLOCK_DIFFICULTY, 10) : DEFAULT_DIFFICULTY;
// SOCKET_PATH is how KNIRVSERVER always runs this in-process (proxied
// through KNIRVGATEWAY, never a directly exposed TCP port). PORT is a
// standalone-dev-only fallback for running this file outside KNIRVSERVER.
const SOCKET_PATH = process.env.SOCKET_PATH || '';
const PORT = parseInt(process.env.PORT || '3000');
const SALT = process.env.SALT || 'your-secret-salt'; // Ensure that you set this environment variable to make the mining process more secure
const DATA_PATH = process.env.DATA_PATH || __dirname;
const CHAIN_ID = process.env.CHAIN_ID || 'transaction-chain-1';

// SQLite setup
fs.mkdirSync(DATA_PATH, { recursive: true });
const dbPath = path.join(DATA_PATH, 'blockchain.db');
const db = new sqlite3.Database(dbPath);

class AsyncMutex {
    constructor() {
      this.isLocked = false;
      this.queue = [];
    }

    async runExclusive(callback) {
      if (this.isLocked) {
        return new Promise((resolve) => {
          this.queue.push(resolve);
        }).then(() => this.runExclusive(callback));
      }

      this.isLocked = true;
      try {
        return await callback();
      } finally {
         this.isLocked = false;
         if(this.queue.length > 0) {
           this.queue.shift()()
         }
      }
    }
}


async function ensureTableExists() {
    return new Promise((resolve, reject) => {
        db.run(`
            CREATE TABLE IF NOT EXISTS blocks (
                block_index INTEGER PRIMARY KEY,
                timestamp INTEGER,
                data TEXT,
                previousHash TEXT,
                nonce INTEGER,
                hash TEXT
            )
        `, (err) => {
            if (err) {
                console.error("Error creating table:", err);
                reject(err);
            } else {
               console.log("Blocks table is ready");
                resolve();
            }
        });
    });
}



class Block {
    constructor(index, timestamp, data, previousHash, nonce = 0) {
        if (typeof index !== 'number') {
            throw new Error('Error: Invalid index, needs to be a Number');
        }
        if (typeof timestamp !== 'number') {
            throw new Error('Error: Invalid timestamp, needs to be a Number');
        }
        if (data !== null && typeof data !== 'object') {
            throw new Error('Error: Invalid data format on block, needs to be null or an object');
        }
        if (typeof previousHash !== 'string') {
            throw new Error('Error: Invalid previousHash, needs to be string');
        }
         if (typeof nonce !== 'number') {
            throw new Error('Error: Invalid nonce type, need to be number.')
        }
        this.index = index;
        this.timestamp = timestamp;
        this.data = data;
        this.previousHash = previousHash;
        this.nonce = nonce;
        this.hash = this.calculateHash();
    }

    calculateHash() {
        const hash = crypto.createHash('sha256');
        hash.update(
            this.index +
            this.previousHash +
            this.timestamp +
            JSON.stringify(this.data) +
             this.nonce +
             SALT
        );
        return hash.digest('hex');
    }

     mineBlock(difficulty) {
        while (this.hash.substring(0, difficulty) !== '0'.repeat(difficulty)) {
            this.nonce = Math.floor(Math.random() * 1000000000); //Update the nonce to a random value to make the mining process more difficult.
            this.hash = this.calculateHash();
        }

         console.log(`[INFO] ${new Date().toISOString()} Block Mined: ${this.hash}`);
    }


    getBlock() {
        return {
            index: this.index,
            timestamp: this.timestamp,
            data: this.data,
            previousHash: this.previousHash,
            nonce: this.nonce,
            hash: this.hash,
        };
    }
}

class Blockchain {
    constructor() {
        this.chain = [];
        this.isChainLoading = true;
    }

    async loadChainFromDB() {
        return new Promise((resolve, reject) => {
          db.all('SELECT * FROM blocks ORDER BY block_index ASC', (err, rows) => {
             if(err) {
                console.error("Error loading chain from SQLite:", err);
                 this.chain = [this.createGenesisBlock()];
                 this.saveBlock(this.chain[0])
                 .then(() => {resolve()})
                 .catch((e) => {
                    console.error("Error saving genesis block to the database, exiting...", e);
                   reject(e)
                   process.exit(1)
                   return
                 })
                 return;
              }

             if(rows.length === 0) {
                 this.chain = [this.createGenesisBlock()];
                 this.saveBlock(this.chain[0]).then(resolve).catch(reject);
                 return;
              }

              try {
                  for(const row of rows) {
                     this.chain.push(new Block(row.block_index, row.timestamp, JSON.parse(row.data), row.previousHash, row.nonce, row.hash));
                  }
                   resolve();
             } catch (e) {
                 console.error("Error parsing data from database", e)
                 this.chain = [this.createGenesisBlock()];
                 this.saveBlock(this.chain[0]).then(resolve).catch(reject);
                 return; // Important, exit after the catch block finishes execution
             }
          })
      }).then(() => {
         this.isChainLoading = false;
          index = this.chain.length;
      })
    }

    createGenesisBlock() {
        return new Block(0, Date.now(), { message: 'Genesis Block' }, '0');
    }

    getLatestBlock() {
        return this.chain[this.chain.length - 1];
    }

     async addBlock(newBlock) {
        if (this.isChainLoading) { // Check if the chain is still loading
            // Wait before attempting to add a block
            await new Promise(resolve => setTimeout(resolve, 500)); //Wait 500 ms
            return this.addBlock(newBlock); // Retry after waiting.
          }

        //Critical section.  Serialize block additions by introducing an async mutex.
         const addBlockMutex = new AsyncMutex()
         await addBlockMutex.runExclusive(async () => {
            newBlock.previousHash = this.getLatestBlock().hash;
            await this.saveBlock(newBlock);
             newBlock.mineBlock(DIFFICULTY);
             this.chain.push(newBlock);
            console.log(`[INFO] Successfully added block ${newBlock.index}`);
        });
    }


   async saveBlock(block) {
     return new Promise((resolve, reject) => {
       db.run(
            'INSERT INTO blocks (block_index, timestamp, data, previousHash, nonce, hash) VALUES (?, ?, ?, ?, ?, ?)',
           [
            block.index,
            block.timestamp,
            JSON.stringify(block.data),
            block.previousHash,
            block.nonce,
            block.hash,
            ],
            (err) => {
               if(err) {
                console.error("Error saving block", err);
                reject(err)
               } else {
                 resolve()
               }
            }
        );
     });
    }

    getBlockchain() {
        return this.chain.map((block) => block.getBlock());
    }
}

function hashTransaction(transaction) {
    return crypto.createHash('sha256').update(JSON.stringify(transaction) + SALT).digest('hex');
}

function normalizeTransaction(transaction = {}, txHash, blockIndex = null, latestHeight = null) {
    const amount = Number.isFinite(transaction.amount) ? transaction.amount : (Number.isFinite(transaction.value) ? transaction.value : 0);
    const timestamp = Number.isFinite(transaction.timestamp) ? transaction.timestamp : Date.now();
    const confirmations = blockIndex == null || latestHeight == null ? 0 : Math.max(1, (latestHeight - blockIndex) + 1);

    return {
        transaction_hash: txHash,
        from: transaction.from || '',
        to: transaction.to || '',
        value: amount,
        amount,
        data: transaction.data || null,
        timestamp,
        signature: transaction.signature || null,
        public_key: transaction.public_key || transaction.publicKey || '',
        type: transaction.type || 'transfer',
        fee: Number.isFinite(transaction.fee) ? transaction.fee : 0,
        status: blockIndex == null ? 'pending' : 'confirmed',
        chain_id: transaction.chain_id || transaction.chainID || CHAIN_ID,
        block_height: blockIndex == null ? null : blockIndex,
        confirmations,
        pqc_signature: transaction.pqc_signature || transaction.pqcSignature || null,
    };
}

function normalizeBlock(block) {
    const latestHeight = blockchain.getLatestBlock()?.index ?? 0;
    const transactionHashes = block.data?.transactionHashes || [];
    const transactions = (block.data?.transactions || []).map((transaction, idx) => normalizeTransaction(
        transaction,
        transactionHashes[idx] || hashTransaction(transaction),
        block.index,
        latestHeight,
    ));

    return {
        block_number: block.index,
        index: block.index,
        timestamp: block.timestamp,
        transactions,
        hash: block.hash,
        prev_hash: block.previousHash,
        previousHash: block.previousHash,
    };
}

function findTransaction(txHash) {
    const latestHeight = blockchain.getLatestBlock()?.index ?? 0;

    if (transactionMap[txHash]) {
        const { blockIndex, transaction } = transactionMap[txHash];
        return {
            blockIndex,
            transaction,
            normalized: normalizeTransaction(transaction, txHash, blockIndex, latestHeight),
        };
    }

    for (const block of blockchain.chain) {
        const transactionHashes = block.data?.transactionHashes || [];
        const transactions = block.data?.transactions || [];

        for (let idx = 0; idx < transactionHashes.length; idx += 1) {
            if (transactionHashes[idx] === txHash) {
                const transaction = transactions[idx];
                return {
                    blockIndex: block.index,
                    transaction,
                    normalized: normalizeTransaction(transaction, txHash, block.index, latestHeight),
                };
            }
        }
    }

    return null;
}

function calculateBalance(address) {
    let balance = 0;

    for (const block of blockchain.chain) {
        for (const transaction of block.data?.transactions || []) {
            const amount = Number.isFinite(transaction.amount) ? transaction.amount : (Number.isFinite(transaction.value) ? transaction.value : 0);
            if (transaction.to === address) {
                balance += amount;
            }
            if (transaction.from === address) {
                balance -= amount;
            }
        }
    }

    return balance;
}

// Transaction Pool
const transactionPool = [];
const transactionMap = {}

// Init
const blockchain = new Blockchain();
let index = blockchain.chain.length;

let mining = false; // Flag to control the mining loop

async function mineBlock() { // Function to mine a single block

    if(mining) return; //If mining is already in progress, prevent additional executions

    mining = true; //Set mining flag
    try {
        const prevBlock = blockchain.getLatestBlock();
        const previousHash = prevBlock ? prevBlock.hash : "0";
        const transactions = [...transactionPool]; //Create a new array with transactions from the transactionPool and assign it to transactions.
        transactionPool.length = 0; // Clear the pool *after* copying transactions
        const transactionHashes = transactions.map((tx) => {
            const txHash = hashTransaction(tx);
            transactionMap[txHash] = { blockIndex: index, transaction: tx };
            return txHash;
        });
        const newBlock = new Block(index, Date.now(), { transactions: transactions, transactionHashes: transactionHashes }, previousHash);


        await blockchain.addBlock(newBlock);
        index++;
    } catch (error) {
        console.error("Error mining a new block:", error);
    } finally{
        mining = false; // Reset the flag *always* in the finally block
    }
}

// API Setup (moved from api.js)
app.use(bodyParser.json());  //Middleware setup

app.get('/health', (req, res) => {
    res.json({
        status: 'ok',
        healthy: true,
        chain_id: CHAIN_ID,
        height: blockchain.getLatestBlock()?.index ?? 0,
        pending_transactions: transactionPool.length,
    });
});

app.post('/transactions', async (req, res) => { //Transaction endpoint
    const transaction = req.body;
    const transactionHash = hashTransaction(transaction);
    transactionPool.push(transaction);

    mineBlock();
    res.status(201).json({ transactionHash, tx_hash: transactionHash });
});

app.post('/transaction', async (req, res) => {
    const transaction = req.body;
    const transactionHash = hashTransaction(transaction);
    transactionPool.push(transaction);

    mineBlock();
    res.status(201).json({ transactionHash, tx_hash: transactionHash });
});

// Internal-only: credits a balance without a signed transaction (used by
// KNIRVSERVER to seed a wallet from KNIRVORACLE's faucet). Goes through the
// same transaction+mine pipeline as everything else, so it stays consistent
// with mining only happening when there's something to mine.
app.post('/wallet/credit', async (req, res) => {
    const { address, amount, reason } = req.body || {};
    if (!address || !Number.isFinite(amount)) {
        res.status(400).json({ message: 'address and numeric amount are required' });
        return;
    }

    const transaction = {
        from: '',
        to: address,
        amount,
        value: amount,
        type: 'credit',
        data: { reason: reason || 'wallet-credit' },
        timestamp: Date.now(),
    };
    const transactionHash = hashTransaction(transaction);
    transactionPool.push(transaction);

    mineBlock();
    res.status(201).json({ transactionHash, tx_hash: transactionHash, address, amount });
});

app.get('/blocks', async (req, res) => {  // Your blocks endpoint
    res.json(blockchain.getBlockchain());
});

app.get('/chain', async (req, res) => {
    res.json({
        chain_id: CHAIN_ID,
        height: blockchain.getLatestBlock()?.index ?? 0,
        blocks: blockchain.getBlockchain().map(normalizeBlock),
    });
});

app.get('/chain/height', async (req, res) => {
    res.json({
        height: blockchain.getLatestBlock()?.index ?? 0,
    });
});

app.get('/transactions/:hash', (req, res) => {  // Your transaction lookup endpoint
    const txHash = req.params.hash;
    const result = findTransaction(txHash);
    if (result) {
        res.json({ blockIndex: result.blockIndex, transaction: result.transaction });
    } else {
        res.status(404).json({ message: "Transaction not found" });
    }
});

app.get('/chain/tx/:hash', (req, res) => {
    const txHash = req.params.hash;
    const result = findTransaction(txHash);
    if (!result) {
        res.status(404).json({ message: 'Transaction not found' });
        return;
    }

    res.json(result.normalized);
});

app.get('/txn_pool', (req, res) => {
    res.json(transactionPool.map((transaction) => normalizeTransaction(transaction, hashTransaction(transaction))));
});

app.get('/account/:address/balance', (req, res) => {
    const { address } = req.params;
    res.json({
        address,
        balance: calculateBalance(address),
        chain_id: CHAIN_ID,
    });
});


// ... other API endpoints (moved into miner.js)


// Normal miner start-up logic: create table, load chain
async function start() {
    try {
        await ensureTableExists();
    } catch (e) {
        console.error("Failed to create table, exiting...", e);
        process.exit(1);
        return;
    }

   // initializeApi(blockchain, index, transactionPool, transactionMap, mining);

    await blockchain.loadChainFromDB(); //Load blockchain before server starts listening
    index = blockchain.chain.length; // Initialize after loading

    // No idle/scheduled mining: blocks are only mined when a transaction (or
    // an internal wallet credit) actually arrives — see mineBlock() calls in
    // the /transactions, /transaction, and /wallet/credit handlers above.
    if (SOCKET_PATH) {
        fs.mkdirSync(path.dirname(SOCKET_PATH), { recursive: true });
        try {
            fs.unlinkSync(SOCKET_PATH); // clear a stale socket file from a previous run
        } catch (e) {
            if (e.code !== 'ENOENT') throw e;
        }
        app.listen(SOCKET_PATH, () => {
            console.log(`[INFO] Server running on unix socket ${SOCKET_PATH}`);
        });
    } else {
        app.listen(PORT, () => {
            console.log(`[INFO] Server running on http://localhost:${PORT}`);
        });
    }
}
start();

module.exports = {
    Block,
    Blockchain
};
