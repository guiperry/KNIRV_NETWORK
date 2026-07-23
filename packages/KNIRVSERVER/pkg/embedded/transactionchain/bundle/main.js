// miner.v11.js
const express = require('express');
const app = express();
const crypto = require('crypto'); //Import crypto
const bodyParser = require('body-parser');
const sqlite3 = require('sqlite3').verbose();
const fs = require('node:fs');
const path = require('node:path');

// Logging: every line is prefixed so it's unambiguous which embedded service
// a given log line came from when interleaved with KNIRVORACLE/KNIRVCHAIN/etc.
const LOG_PREFIX = '[Transaction Chain]';
function logInfo(...args) { console.log(LOG_PREFIX, ...args); }
function logError(...args) { console.error(LOG_PREFIX, ...args); }

// Config
const DEFAULT_DIFFICULTY = 0;
const DIFFICULTY = process.env.BLOCK_DIFFICULTY ? parseInt(process.env.BLOCK_DIFFICULTY, 10) : DEFAULT_DIFFICULTY;
const PORT = parseInt(process.env.PORT || '3000');
// SOCKET_PATH is how KNIRVSERVER's Go supervisor (pkg/embedded/transactionchain/process.go)
// actually reaches this service — over a Unix domain socket, never a TCP port
// (see that file's own doc comment: "nothing exposes a TCP port for it").
// When set, it takes priority over PORT.
const SOCKET_PATH = process.env.SOCKET_PATH || '';
const SALT = process.env.SALT || 'your-secret-salt'; // Ensure that you set this environment variable to make the mining process more secure
const DATA_PATH = process.env.DATA_PATH || __dirname;
const CHAIN_ID = process.env.CHAIN_ID || 'transaction-chain-1';

// resolveNetworkMode mirrors the Go services' own KNIRV_NETWORK_MODE /
// KNIRV_TESTNET resolution (see e.g. KNIRVGATEWAY's config.go).
function resolveNetworkMode() {
    const mode = (process.env.KNIRV_NETWORK_MODE || '').trim().toLowerCase();
    if (mode) return mode;
    if (['true', '1', 'yes'].includes((process.env.KNIRV_TESTNET || '').trim().toLowerCase())) return 'testnet';
    return 'production';
}

// resolveHeartbeatIntervalMs decides how often to mine a block even with an
// empty transaction pool. This used to be unconditional (mining MAX_BLOCKS=500
// empty blocks 5s apart on every single startup, regardless of network mode
// or pool contents) — pure noise, since nothing rewards empty blocks. Now:
// an explicit BLOCK_TIME always wins; otherwise testnet gets a slow 30-minute
// keep-alive heartbeat and production is fully event-driven (0 = disabled —
// mining only ever happens when a transaction actually arrives).
function resolveHeartbeatIntervalMs() {
    if (process.env.BLOCK_TIME) {
        return parseInt(process.env.BLOCK_TIME, 10);
    }
    const mode = resolveNetworkMode();
    if (mode === 'production' || mode === 'prod' || mode === 'mainnet') {
        return 0;
    }
    return 30 * 60 * 1000;
}
const HEARTBEAT_INTERVAL_MS = resolveHeartbeatIntervalMs();

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
                logError("Error creating table:", err);
                reject(err);
            } else {
               logInfo("Blocks table is ready");
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

         logInfo(`${new Date().toISOString()} Block Mined: ${this.hash}`);
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
                logError("Error loading chain from SQLite:", err);
                 this.chain = [this.createGenesisBlock()];
                 this.saveBlock(this.chain[0])
                 .then(() => {resolve()})
                 .catch((e) => {
                    logError("Error saving genesis block to the database, exiting...", e);
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
                 logError("Error parsing data from database", e)
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
            logInfo(`Successfully added block ${newBlock.index}`);
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
                logError("Error saving block", err);
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

// mineBlock mines a single block. With force=false (the default — used by
// the /transactions and /transaction endpoints right after queuing a real
// transaction) it skips mining entirely when the pool is empty, matching the
// Rust Validation Chain's own "no transactions in the pool, skipping block
// creation" behavior — there is no reason to mine an empty block, since
// nothing rewards one. force=true is only used by the heartbeat interval
// below, which deliberately still wants a block on its slow keep-alive
// cadence even when idle.
async function mineBlock(force = false) {

    if(mining) return; //If mining is already in progress, prevent additional executions

    if (!force && transactionPool.length === 0) {
        logInfo("No transactions in the pool, skipping block creation.");
        return;
    }

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
        logError("Error mining a new block:", error);
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

// /wallet/credit is how KNIRVSERVER's launcher provisions a wallet's initial
// balance (see main.go's creditTransactionChainWallet) — e.g. the root.key
// holder's one-time provisional funding on startup. There is no separate
// ledger entry type for a credit: it is recorded as an ordinary transaction
// from a synthetic treasury address, same as any other transfer, so
// calculateBalance and block explorers see it consistently. This always
// mines immediately (force=true) regardless of the heartbeat policy — a
// wallet credit is real activity, not an empty block.
app.post('/wallet/credit', async (req, res) => {
    const { address, amount, reason } = req.body || {};
    if (typeof address !== 'string' || !address) {
        res.status(400).json({ error: 'address is required' });
        return;
    }
    const numericAmount = Number(amount);
    if (!Number.isFinite(numericAmount) || numericAmount <= 0) {
        res.status(400).json({ error: 'amount must be a positive number' });
        return;
    }
    const transaction = {
        from: 'TREASURY',
        to: address,
        amount: numericAmount,
        type: 'credit',
        data: { reason: reason || 'wallet credit' },
    };
    const transactionHash = hashTransaction(transaction);
    transactionPool.push(transaction);

    await mineBlock(true);
    res.status(200).json({ transactionHash, tx_hash: transactionHash, address, amount: numericAmount });
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
        logError("Failed to create table, exiting...", e);
        process.exit(1);
        return;
    }

   // initializeApi(blockchain, index, transactionPool, transactionMap, mining);

    await blockchain.loadChainFromDB(); //Load blockchain before server starts listening
    index = blockchain.chain.length; // Initialize after loading

    // Bind the Unix socket KNIRVSERVER's Go supervisor expects when it set
    // SOCKET_PATH (the normal embedded-service path — see process.go). Only
    // fall back to a bare TCP PORT for standalone/manual runs where
    // SOCKET_PATH was never set.
    let listenTarget = PORT;
    let listenDescription = `http://localhost:${PORT}`;
    if (SOCKET_PATH) {
        try {
            fs.unlinkSync(SOCKET_PATH); // clear a stale socket file from a previous run
        } catch (e) {
            if (e.code !== 'ENOENT') throw e;
        }
        listenTarget = SOCKET_PATH;
        listenDescription = `unix:${SOCKET_PATH}`;
    }

    app.listen(listenTarget, () => {  // Start the server *before* mining. This is correct.

         logInfo(`Server running on ${listenDescription}`);

         // Mining otherwise only happens when a transaction actually arrives
         // (see the /transactions and /transaction handlers below). This
         // heartbeat is purely a liveness/keep-alive signal for idle
         // periods — disabled entirely (HEARTBEAT_INTERVAL_MS === 0) in
         // production, where there is no reason to mine empty blocks at all.
         if (HEARTBEAT_INTERVAL_MS > 0) {
             logInfo(`Heartbeat mining enabled: a block is forced every ${HEARTBEAT_INTERVAL_MS}ms even with an empty pool.`);
             setInterval(() => { mineBlock(true); }, HEARTBEAT_INTERVAL_MS);
         } else {
             logInfo("Heartbeat mining disabled — mining only when a transaction is submitted.");
         }
    });
}
start();

module.exports = {
    Block,
    Blockchain
};
