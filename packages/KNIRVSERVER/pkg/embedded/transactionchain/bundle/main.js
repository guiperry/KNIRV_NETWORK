// miner.v11.js
const express = require('express');
const app = express();
const crypto = require('crypto'); //Import crypto
const bodyParser = require('body-parser');
const sqlite3 = require('sqlite3').verbose();
const fs = require('node:fs');
const path = require('node:path');

// Config
const DEMO_ENABLED = String(process.env.KNIRV_ENABLE_DEMO || '').toLowerCase() === 'true';
const DEFAULT_DIFFICULTY = 2;
const configuredDifficulty = process.env.BLOCK_DIFFICULTY ? parseInt(process.env.BLOCK_DIFFICULTY, 10) : DEFAULT_DIFFICULTY;
if (!Number.isInteger(configuredDifficulty) || configuredDifficulty < 0 || configuredDifficulty > 8) {
    throw new Error('BLOCK_DIFFICULTY must be an integer between 0 and 8');
}
const DIFFICULTY = configuredDifficulty === 0 && !DEMO_ENABLED ? DEFAULT_DIFFICULTY : configuredDifficulty;
// SOCKET_PATH takes precedence: KNIRVSERVER spawns this process bound to a
// Unix domain socket, proxied by KNIRVSERVER/KNIRVGATEWAY, rather than
// exposing its own TCP port. PORT is kept as a fallback for standalone/dev
// use outside that supervision.
const SOCKET_PATH = process.env.SOCKET_PATH || '';
const PORT = parseInt(process.env.PORT || '3000');
const DATA_PATH = process.env.DATA_PATH || __dirname;
const CHAIN_ID = process.env.CHAIN_ID || 'transaction-chain-1';
const INTERNAL_AUTH_TOKEN = process.env.KNIRV_INTERNAL_AUTH_TOKEN || '';

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
             this.nonce
        );
        return hash.digest('hex');
    }

     mineBlock(difficulty) {
        while (this.hash.substring(0, difficulty) !== '0'.repeat(difficulty)) {
            this.nonce += 1;
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
        this.addBlockMutex = new AsyncMutex();
    }

    async loadChainFromDB() {
        return new Promise((resolve, reject) => {
          db.all('SELECT * FROM blocks ORDER BY block_index ASC', (err, rows) => {
             if(err) {
                console.error("Error loading chain from SQLite:", err);
                 reject(err);
                 return;
              }

             if(rows.length === 0) {
                 this.chain = [this.createGenesisBlock()];
                 this.saveBlock(this.chain[0]).then(resolve).catch(reject);
                 return;
              }

              try {
                  for(const row of rows) {
                     const block = new Block(row.block_index, row.timestamp, JSON.parse(row.data), row.previousHash, row.nonce);
                     if (block.hash !== row.hash) throw new Error(`block ${row.block_index} hash does not match persisted content`);
                     if (row.block_index !== this.chain.length) throw new Error(`non-contiguous block index ${row.block_index}`);
                     if (row.block_index === 0) {
                        if (row.previousHash !== '0') throw new Error('genesis previousHash must be 0');
                     } else if (row.previousHash !== this.chain[this.chain.length - 1].hash) {
                        throw new Error(`block ${row.block_index} does not link to its predecessor`);
                     }
                     if (row.block_index > 0 && !row.hash.startsWith('0'.repeat(DIFFICULTY))) throw new Error(`block ${row.block_index} does not satisfy configured difficulty`);
                     this.chain.push(block);
                  }
                   resolve();
             } catch (e) {
                 console.error("Stored blockchain validation failed", e);
                 reject(e);
                 return;
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

         await this.addBlockMutex.runExclusive(async () => {
            newBlock.previousHash = this.getLatestBlock().hash;
            newBlock.hash = newBlock.calculateHash();
             newBlock.mineBlock(DIFFICULTY);
            await this.saveBlock(newBlock);
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

function encodeVarint(input) {
    let value = BigInt(input);
    const out = [];
    do {
        let byte = Number(value & 0x7fn);
        value >>= 7n;
        if (value) byte |= 0x80;
        out.push(byte);
    } while (value);
    return Buffer.from(out);
}

function bytesField(number, value) {
    const bytes = Buffer.from(value || []);
    if (!bytes.length) return Buffer.alloc(0);
    return Buffer.concat([encodeVarint((number << 3) | 2), encodeVarint(bytes.length), bytes]);
}

function stringField(number, value) {
    return value ? bytesField(number, Buffer.from(value, 'utf8')) : Buffer.alloc(0);
}

function uintField(number, value) {
    if (!value || BigInt(value) === 0n) return Buffer.alloc(0);
    return Buffer.concat([encodeVarint(number << 3), encodeVarint(value)]);
}

function readFields(input) {
    const data = Buffer.from(input);
    const fields = [];
    let offset = 0;
    const readVarint = () => {
        let value = 0n;
        let shift = 0n;
        while (offset < data.length) {
            const byte = data[offset++];
            value |= BigInt(byte & 0x7f) << shift;
            if (!(byte & 0x80)) return value;
            shift += 7n;
        }
        throw new Error('truncated protobuf varint');
    };
    while (offset < data.length) {
        const tag = readVarint();
        const number = Number(tag >> 3n);
        const wire = Number(tag & 7n);
        if (wire === 0) fields.push({ number, wire, value: readVarint() });
        else if (wire === 2) {
            const length = Number(readVarint());
            if (offset + length > data.length) throw new Error('truncated protobuf field');
            fields.push({ number, wire, value: data.subarray(offset, offset + length) });
            offset += length;
        } else throw new Error(`unsupported protobuf wire type ${wire}`);
    }
    return fields;
}

function oneBytes(fields, number) {
    const values = fields.filter((field) => field.number === number && field.wire === 2);
    if (values.length !== 1) throw new Error(`expected one protobuf field ${number}`);
    return values[0].value;
}

function parseAction(bodyBytes) {
    const txBody = readFields(bodyBytes);
    const message = oneBytes(txBody, 1);
    const anyFields = readFields(message);
    if (oneBytes(anyFields, 1).toString('utf8') !== '/knirv.signing.v1.Action') throw new Error('unsupported transaction action');
    const actionFields = readFields(oneBytes(anyFields, 2));
    const text = (n) => {
        const found = actionFields.find((field) => field.number === n && field.wire === 2);
        return found ? found.value.toString('utf8') : '';
    };
    const bytes = (n) => actionFields.find((field) => field.number === n && field.wire === 2)?.value || Buffer.alloc(0);
    const integer = (n) => actionFields.find((field) => field.number === n && field.wire === 0)?.value || 0n;
    if (text(1) !== 'knirv.action.v1') throw new Error('unsupported action schema');
    return { action: text(2), sender: text(3), recipient: text(4), amount: integer(5), payload: bytes(6), timestamp: integer(7) };
}

function parseSequence(authInfoBytes) {
    const authFields = readFields(authInfoBytes);
    const signerInfo = oneBytes(authFields, 1);
    const signerFields = readFields(signerInfo);
    const sequence = signerFields.find((field) => field.number === 3 && field.wire === 0)?.value || 0n;
    if (sequence > BigInt(Number.MAX_SAFE_INTEGER)) throw new Error('sequence exceeds safe integer range');
    return Number(sequence);
}

function parseUint(value, fieldName, allowZero = true) {
    if (typeof value === 'number') {
        if (!Number.isSafeInteger(value)) throw new Error(`${fieldName} must be a safe unsigned integer or decimal string`);
        value = String(value);
    }
    if (typeof value !== 'string' || !/^(0|[1-9][0-9]*)$/.test(value)) {
        throw new Error(`${fieldName} must be an unsigned decimal integer`);
    }
    const parsed = BigInt(value);
    if (!allowZero && parsed === 0n) throw new Error(`${fieldName} must be positive`);
    if (parsed > 0xffffffffffffffffn) throw new Error(`${fieldName} exceeds uint64 range`);
    return parsed;
}

function parseAuthInfo(authInfoBytes) {
    const authFields = readFields(authInfoBytes);
    const sequence = parseSequence(authInfoBytes);
    const encodedFees = authFields.filter((field) => field.number === 2 && field.wire === 2);
    if (encodedFees.length > 1) throw new Error('AuthInfo contains multiple fee fields');
    const feeFields = encodedFees.length === 1 ? readFields(encodedFees[0].value) : [];
    const coins = feeFields.filter((field) => field.number === 1 && field.wire === 2);
    if (coins.length > 1) throw new Error('only one fee denomination is supported');
    let denom = '';
    let amount = 0n;
    if (coins.length === 1) {
        const coinFields = readFields(coins[0].value);
        denom = oneBytes(coinFields, 1).toString('utf8');
        amount = parseUint(oneBytes(coinFields, 2).toString('utf8'), 'fee amount');
    }
    const text = (number) => feeFields.find((field) => field.number === number && field.wire === 2)?.value.toString('utf8') || '';
    const gas = feeFields.find((field) => field.number === 2 && field.wire === 0)?.value || 0n;
    return { sequence, fee: { denom, amount, gas, payer: text(3), granter: text(4) } };
}

function bech32Polymod(values) {
    const generators = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
    let checksum = 1;
    for (const value of values) {
        const top = checksum >>> 25;
        checksum = ((checksum & 0x1ffffff) << 5) ^ value;
        for (let i = 0; i < 5; i += 1) if ((top >>> i) & 1) checksum ^= generators[i];
    }
    return checksum >>> 0;
}

function convertBits(data, from, to, pad) {
    let acc = 0, bits = 0;
    const out = [], maxv = (1 << to) - 1;
    for (const value of data) {
        acc = (acc << from) | value;
        bits += from;
        while (bits >= to) { bits -= to; out.push((acc >>> bits) & maxv); }
    }
    if (pad && bits) out.push((acc << (to - bits)) & maxv);
    return out;
}

function knirvAddress(publicKey) {
    const words = convertBits(crypto.createHash('ripemd160').update(crypto.createHash('sha256').update(publicKey).digest()).digest(), 8, 5, true);
    const hrp = 'knirv';
    const expanded = [...hrp].map((c) => c.charCodeAt(0) >> 5).concat([0], [...hrp].map((c) => c.charCodeAt(0) & 31));
    const polymod = bech32Polymod(expanded.concat(words, [0, 0, 0, 0, 0, 0])) ^ 1;
    const checksum = Array.from({ length: 6 }, (_, i) => (polymod >>> (5 * (5 - i))) & 31);
    const alphabet = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
    return `${hrp}1${words.concat(checksum).map((word) => alphabet[word]).join('')}`;
}

function verifyCanonicalTransaction(transaction) {
    const body = Buffer.from(transaction.body_bytes || '', 'base64');
    const auth = Buffer.from(transaction.auth_info_bytes || '', 'base64');
    const signature = Buffer.from(transaction.signatures?.[0] || transaction.signature || '', 'base64');
    const publicKey = Buffer.from(transaction.public_key || transaction.publicKey || '', 'base64');
    const chainId = transaction.chain_id || transaction.chainID;
    const accountNumber = parseUint(transaction.account_number ?? transaction.accountNumber ?? '0', 'account_number');
    if (!body.length || !auth.length || signature.length !== 64 || publicKey.length !== 33 || !chainId) throw new Error('canonical signing fields are incomplete');
    if (chainId !== CHAIN_ID) throw new Error(`wrong chain_id: got ${chainId}, want ${CHAIN_ID}`);
    const signDoc = Buffer.concat([bytesField(1, body), bytesField(2, auth), stringField(3, chainId), uintField(4, accountNumber)]);
    const uncompressed = crypto.ECDH.convertKey(publicKey, 'secp256k1', undefined, undefined, 'uncompressed');
    const spki = Buffer.concat([Buffer.from('3056301006072a8648ce3d020106052b8104000a034200', 'hex'), uncompressed]);
    const key = crypto.createPublicKey({ key: spki, format: 'der', type: 'spki' });
    if (!crypto.verify('sha256', signDoc, { key, dsaEncoding: 'ieee-p1363' }, signature)) throw new Error('signature verification failed');
    const action = parseAction(body);
    if (accountNumber !== 0n) throw new Error('transaction-chain accounts currently require account_number 0');
    const amount = parseUint(transaction.amount ?? transaction.value ?? '0', 'amount');
    const payload = transaction.data == null
        ? Buffer.alloc(0)
        : Buffer.from(typeof transaction.data === 'string' ? transaction.data : JSON.stringify(transaction.data));
    if (!Number.isSafeInteger(transaction.timestamp) || transaction.timestamp <= 0) throw new Error('transaction timestamp is required');
    if (action.sender !== transaction.from || action.recipient !== transaction.to || action.amount !== amount ||
        action.action !== (transaction.type || 'transfer') || action.timestamp !== BigInt(transaction.timestamp) ||
        !action.payload.equals(payload)) throw new Error('signed action does not match transaction');
    const address = knirvAddress(publicKey);
    if (address !== transaction.from) throw new Error('public key does not match sender');
    const txRaw = Buffer.concat([bytesField(1, body), bytesField(2, auth), bytesField(3, signature)]);
    const parsedAuth = parseAuthInfo(auth);
    if (parsedAuth.fee.granter) throw new Error('fee grants are not supported by the transaction chain');
    if (parsedAuth.fee.payer && parsedAuth.fee.payer !== transaction.from) throw new Error('fee payer must be the sender');
    if (parsedAuth.fee.amount > 0n && parsedAuth.fee.denom !== (process.env.KNIRV_FEE_DENOM || 'uknirv')) {
        throw new Error(`unsupported fee denomination ${parsedAuth.fee.denom}`);
    }
    const declaredFee = parseUint(transaction.fee ?? parsedAuth.fee.amount.toString(), 'fee');
    if (declaredFee !== parsedAuth.fee.amount) throw new Error('declared fee does not match signed AuthInfo fee');
    return { hash: crypto.createHash('sha256').update(txRaw).digest('hex').toUpperCase(), action, sequence: parsedAuth.sequence, fee: parsedAuth.fee };
}

function hashTransaction(transaction) {
    if (DEMO_ENABLED && transaction.from === 'TREASURY' && transaction.type === 'credit') {
        return crypto.createHash('sha256').update(JSON.stringify(transaction)).digest('hex').toUpperCase();
    }
    return verifyCanonicalTransaction(transaction).hash;
}

function normalizeTransaction(transaction = {}, txHash, blockIndex = null, latestHeight = null) {
    const amount = parseUint(transaction.amount ?? transaction.value ?? '0', 'amount').toString();
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
        fee: parseUint(transaction.fee ?? '0', 'fee').toString(),
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
    let balance = 0n;

    for (const block of blockchain.chain) {
        for (const transaction of block.data?.transactions || []) {
            const amount = parseUint(transaction.amount ?? transaction.value ?? '0', 'amount');
            if (transaction.to === address) {
                balance += amount;
            }
            if (transaction.from === address) {
                balance -= amount;
                if (!(DEMO_ENABLED && transaction.from === 'TREASURY' && transaction.type === 'credit')) {
                    balance -= parseAuthInfo(Buffer.from(transaction.auth_info_bytes || '', 'base64')).fee.amount;
                }
            }
        }
    }

    return balance.toString();
}

// Transaction Pool
const transactionPool = [];
const transactionMap = {}
const accountSequences = new Map();

function rebuildAccountSequences() {
    accountSequences.clear();
    for (const block of blockchain.chain) {
        for (const transaction of block.data?.transactions || []) {
            if (DEMO_ENABLED && transaction.from === 'TREASURY' && transaction.type === 'credit') continue;
            try {
                const verified = verifyCanonicalTransaction(transaction);
                accountSequences.set(transaction.from, Math.max(accountSequences.get(transaction.from) || 0, verified.sequence + 1));
            } catch (error) {
                throw new Error(`stored transaction failed canonical verification: ${error.message}`);
            }
        }
    }
}

function admitCanonicalTransaction(transaction) {
    const verified = verifyCanonicalTransaction(transaction);
    const expected = accountSequences.get(transaction.from) || 0;
    if (verified.sequence !== expected) throw new Error(`wrong account sequence: got ${verified.sequence}, want ${expected}`);
    const required = verified.action.amount + verified.fee.amount;
    if (verified.action.action === 'transfer' && BigInt(calculateBalance(transaction.from)) < required) {
        throw new Error('insufficient balance for amount and fee');
    }
    return verified;
}

const transactionSubmitMutex = new AsyncMutex();

async function submitCanonicalTransaction(transaction) {
    return transactionSubmitMutex.runExclusive(async () => {
        const verified = admitCanonicalTransaction(transaction);
        transactionPool.push(transaction);
        await mineBlock();
        accountSequences.set(transaction.from, verified.sequence + 1);
        return verified.hash;
    });
}

// Init
const blockchain = new Blockchain();
let index = blockchain.chain.length;

const miningMutex = new AsyncMutex();

async function mineBlock() { // Function to mine a single block
  return miningMutex.runExclusive(async () => {
    let transactions = [];
    try {
        const prevBlock = blockchain.getLatestBlock();
        const previousHash = prevBlock ? prevBlock.hash : "0";
        transactions = [...transactionPool]; //Create a new array with transactions from the transactionPool and assign it to transactions.
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
        transactionPool.unshift(...transactions);
        throw error;
    }
  });
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
    try {
        const transactionHash = await submitCanonicalTransaction(transaction);
        res.status(201).json({ transactionHash, tx_hash: transactionHash });
    } catch (error) {
        res.status(400).json({ error: error.message });
    }
});

app.post('/transaction', async (req, res) => {
    const transaction = req.body;
    try {
        const transactionHash = await submitCanonicalTransaction(transaction);
        res.status(201).json({ transactionHash, tx_hash: transactionHash });
    } catch (error) {
        res.status(400).json({ error: error.message });
    }
});

// /wallet/credit is how KNIRVSERVER's launcher provisions a wallet's initial
// balance (see main.go's creditTransactionChainWallet) — e.g. the root.key
// holder's one-time provisional funding on startup. There is no separate
// ledger entry type for a credit: it is recorded as an ordinary transaction
// from a synthetic treasury address, same as any other transfer, so
// calculateBalance and block explorers see it consistently. Mined and
// awaited immediately so the caller's 2xx response means the credit is
// already on-chain.
app.post('/wallet/credit', async (req, res) => {
	if (!DEMO_ENABLED) {
		res.status(404).json({ error: 'demo wallet credit is disabled' });
		return;
	}
    if (!INTERNAL_AUTH_TOKEN || req.get('Authorization') !== `Bearer ${INTERNAL_AUTH_TOKEN}`) {
        res.status(403).json({ error: 'internal authorization required' });
        return;
    }
    const { address, amount, reason } = req.body || {};
    if (typeof address !== 'string' || !address) {
        res.status(400).json({ error: 'address is required' });
        return;
    }
    let creditAmount;
    try {
        creditAmount = parseUint(amount, 'amount', false).toString();
    } catch (error) {
        res.status(400).json({ error: error.message });
        return;
    }
    const transaction = {
        from: 'TREASURY',
        to: address,
        amount: creditAmount,
        type: 'credit',
        data: { reason: reason || 'wallet credit' },
    };
    const transactionHash = hashTransaction(transaction);
    transactionPool.push(transaction);

    await mineBlock();
    res.status(200).json({ transactionHash, tx_hash: transactionHash, address, amount: creditAmount });
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
    rebuildAccountSequences();
    index = blockchain.chain.length; // Initialize after loading

    // Blocks are only mined on demand (see the /transactions and /transaction
    // handlers calling mineBlock()) — there is no background timer mining
    // empty blocks here.
    if (SOCKET_PATH) {
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
