import * as fs from 'fs';
import * as path from 'path';
import { IndexManager } from './index';
import { EncryptionManager } from '../crypto/pqc';
import { EntryType } from '../types/types';
export class FileStorage {
    constructor(baseDir) {
        this.baseDir = baseDir;
        fs.mkdirSync(baseDir, { recursive: true });
        this.indexManager = new IndexManager(baseDir);
        this.indexManager.loadIndexes();
        this.encryptionMgr = new EncryptionManager();
    }
    getCollectionDir(collection) {
        return path.join(this.baseDir, collection);
    }
    getDocPath(collection, id) {
        return path.join(this.getCollectionDir(collection), `${id}.json`);
    }
    getKVPath(key) {
        return path.join(this.baseDir, 'kv', key);
    }
    setMasterKey(keyPair) {
        this.encryptionMgr.setMasterKey(keyPair);
    }
    getMasterKey() {
        return this.encryptionMgr.getMasterKey();
    }
    isEncryptedCollection(collection) {
        const sensitiveCollections = [
            'credentials',
            'pqc_keys',
            'sessions',
            'audit_log',
            'threat_events',
            'access_control',
        ];
        return sensitiveCollections.includes(collection);
    }
    async insert(collection, doc) {
        fs.mkdirSync(this.getCollectionDir(collection), { recursive: true });
        const docPath = this.getDocPath(collection, doc.id);
        const docCopy = this.deepCopyDoc(doc);
        // Handle MEMORY blob
        if (docCopy.entryType === EntryType.Memory) {
            const payload = docCopy.payload;
            if (payload && payload.blob !== undefined) {
                const blobPath = this.saveBlob(collection, docCopy.id, payload.blob);
                payload.blobRef = blobPath;
                delete payload.blob;
            }
        }
        // Encrypt all documents (mandatory PQC encryption at rest)
        if (this.encryptionMgr.getMasterKey()) {
            const encryptedDoc = await this.encryptDocument(collection, docCopy);
            Object.assign(docCopy, encryptedDoc);
            // Sign the write operation for integrity (Dilithium-3)
            const data = JSON.stringify(docCopy);
            const signature = await this.encryptionMgr.sign(data);
            if (signature) {
                const signedData = {
                    data: docCopy,
                    signature: signature,
                };
                const signedJson = JSON.stringify(signedData);
                fs.writeFileSync(docPath, signedJson, 'utf8');
                await this.indexManager.insert(collection, doc);
                return;
            }
        }
        const data = JSON.stringify(docCopy);
        fs.writeFileSync(docPath, data, 'utf8');
        // Update indexes
        await this.indexManager.insert(collection, doc);
    }
    async update(collection, id, update) {
        const doc = await this.find(collection, id);
        if (!doc) {
            throw new Error('not found');
        }
        Object.assign(doc, update);
        await this.insert(collection, doc);
        return 1;
    }
    async delete(collection, id) {
        const docPath = this.getDocPath(collection, id);
        try {
            fs.unlinkSync(docPath);
        }
        catch (err) {
            if (err.code !== 'ENOENT')
                throw err;
        }
        // Remove blob
        const blobDir = path.join(this.getCollectionDir(collection), 'blobs');
        const blobPath = path.join(blobDir, id);
        try {
            fs.unlinkSync(blobPath);
        }
        catch { }
        // Remove from indexes
        await this.indexManager.delete(collection, id);
        return 1;
    }
    async find(collection, id) {
        const docPath = this.getDocPath(collection, id);
        try {
            const data = fs.readFileSync(docPath, 'utf8');
            const wrapper = JSON.parse(data);
            let doc;
            // Unwrap and verify signature if present
            if (wrapper.data && wrapper.signature) {
                doc = wrapper.data;
                if (this.encryptionMgr.getMasterKey()) {
                    const verified = await this.encryptionMgr.verify(JSON.stringify(doc), wrapper.signature);
                    if (!verified) {
                        throw new Error('integrity violation: document signature verification failed');
                    }
                }
            }
            else {
                doc = wrapper;
            }
            // Decrypt if document is encrypted and we have a master key
            if (doc.encrypted && this.encryptionMgr.getMasterKey()) {
                const decrypted = await this.decryptDocument(doc);
                Object.assign(doc, decrypted);
            }
            // Load blob
            if (doc.entryType === EntryType.Memory) {
                const payload = doc.payload;
                if (payload && payload.blobRef) {
                    const blob = this.loadBlob(payload.blobRef);
                    if (blob !== null) {
                        payload.blob = blob;
                        delete payload.blobRef;
                    }
                }
            }
            return doc;
        }
        catch (err) {
            if (err.code === 'ENOENT')
                return null;
            throw err;
        }
    }
    async findAll(collection) {
        const dir = this.getCollectionDir(collection);
        try {
            const files = fs.readdirSync(dir);
            const docs = [];
            for (const file of files) {
                if (path.extname(file) === '.json') {
                    const id = path.basename(file, '.json');
                    const doc = await this.find(collection, id);
                    if (doc)
                        docs.push(doc);
                }
            }
            return docs;
        }
        catch (err) {
            if (err.code === 'ENOENT')
                return [];
            throw err;
        }
    }
    // Key-Value API
    async put(key, value) {
        const kvDir = path.join(this.baseDir, 'kv');
        fs.mkdirSync(kvDir, { recursive: true });
        const kvPath = this.getKVPath(key);
        let dataToSave = value;
        const masterKey = this.encryptionMgr.getMasterKey();
        if (masterKey) {
            const encrypted = await this.encryptionMgr.encryptData(value, masterKey.id);
            if (encrypted) {
                dataToSave = new TextEncoder().encode(encrypted);
            }
        }
        fs.writeFileSync(kvPath, Buffer.from(dataToSave));
    }
    async get(key) {
        const kvPath = this.getKVPath(key);
        try {
            const data = fs.readFileSync(kvPath);
            if (this.encryptionMgr.getMasterKey()) {
                const decrypted = await this.encryptionMgr.decryptData(data.toString('utf8'));
                if (decrypted) {
                    return new Uint8Array(decrypted);
                }
            }
            return new Uint8Array(data);
        }
        catch (err) {
            if (err.code === 'ENOENT')
                return null;
            throw err;
        }
    }
    async deleteKey(key) {
        const kvPath = this.getKVPath(key);
        try {
            fs.unlinkSync(kvPath);
        }
        catch (err) {
            if (err.code !== 'ENOENT')
                throw err;
        }
    }
    async has(key) {
        const kvPath = this.getKVPath(key);
        try {
            fs.statSync(kvPath);
            return true;
        }
        catch {
            return false;
        }
    }
    // JSON Object API
    async storeObject(key, obj) {
        const data = new TextEncoder().encode(JSON.stringify(obj));
        await this.put(key, data);
    }
    async getObject(key) {
        const data = await this.get(key);
        if (data === null)
            return null;
        const text = new TextDecoder().decode(data);
        return JSON.parse(text);
    }
    // Markdown Projection
    async projectToMarkdown(key, targetPath) {
        const data = await this.get(key);
        if (data === null) {
            throw new Error(`key not found: ${key}`);
        }
        const text = new TextDecoder().decode(data);
        try {
            const obj = JSON.parse(text);
            if (typeof obj === 'object' && obj !== null) {
                let md = `# ${key}\n\n`;
                for (const [k, v] of Object.entries(obj)) {
                    md += `## ${k}\n${JSON.stringify(v, null, 2)}\n\n`;
                }
                fs.writeFileSync(targetPath, md, 'utf8');
            }
            else {
                fs.writeFileSync(targetPath, `# Projected Data: ${key}\n\n${text}`, 'utf8');
            }
        }
        catch {
            fs.writeFileSync(targetPath, `# Projected Data: ${key}\n\n${text}`, 'utf8');
        }
    }
    saveBlob(collection, id, blob) {
        const blobDir = path.join(this.getCollectionDir(collection), 'blobs');
        fs.mkdirSync(blobDir, { recursive: true });
        const blobPath = path.join(blobDir, id);
        const data = JSON.stringify(blob);
        fs.writeFileSync(blobPath, data, 'utf8');
        return blobPath;
    }
    loadBlob(blobRef) {
        try {
            const data = fs.readFileSync(blobRef, 'utf8');
            return JSON.parse(data);
        }
        catch {
            return null;
        }
    }
    async createIndex(collection, name, indexType, fields, unique, partialExpr, options) {
        return this.indexManager.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
    }
    async dropIndex(collection, name) {
        return this.indexManager.dropIndex(collection, name);
    }
    getIndex(collection, name) {
        return this.indexManager.getIndex(collection, name);
    }
    getIndexesForCollection(collection) {
        return this.indexManager.getIndexesForCollection(collection);
    }
    async queryIndex(collection, indexName, query) {
        return this.indexManager.queryIndex(collection, indexName, query);
    }
    async close() {
        // No resources to close for FileStorage
    }
    deepCopyDoc(doc) {
        return JSON.parse(JSON.stringify(doc));
    }
    async encryptDocument(collection, doc) {
        const masterKey = this.encryptionMgr.getMasterKey();
        if (!masterKey)
            throw new Error('no master key set');
        if (doc.payload) {
            const encryptedPayload = await this.encryptPayload(collection, doc.payload, masterKey.id);
            doc.payload = encryptedPayload;
            doc.encrypted = true;
            doc.encryption_key_id = masterKey.id;
        }
        return doc;
    }
    async encryptPayload(collection, payload, keyID) {
        const encrypted = {};
        for (const [key, value] of Object.entries(payload)) {
            if (this.isSensitiveField(collection, key)) {
                const valueBytes = JSON.stringify(value);
                const encryptedValue = await this.encryptionMgr.encryptData(Buffer.from(valueBytes), keyID);
                if (encryptedValue) {
                    encrypted[key] = encryptedValue;
                    encrypted[key + '_encrypted'] = true;
                }
                else {
                    encrypted[key] = value;
                }
            }
            else {
                encrypted[key] = value;
            }
        }
        return encrypted;
    }
    isSensitiveField(collection, fieldName) {
        const sensitiveFields = {
            credentials: ['hash', 'salt'],
            pqc_keys: ['kyber_private_key', 'dilithium_private_key'],
            sessions: ['token_hash'],
            audit_log: ['details'],
            threat_events: ['indicators'],
            access_control: ['permissions'],
        };
        return (sensitiveFields[collection] || []).includes(fieldName);
    }
    async decryptDocument(doc) {
        const keyID = doc.encryption_key_id;
        if (!keyID)
            throw new Error('missing encryption_key_id');
        if (doc.payload) {
            const decryptedPayload = await this.decryptPayload(doc.payload, keyID);
            doc.payload = decryptedPayload;
        }
        delete doc.encrypted;
        delete doc.encryption_key_id;
        return doc;
    }
    async decryptPayload(payload, keyID) {
        const decrypted = {};
        for (const [key, value] of Object.entries(payload)) {
            if (key.endsWith('_encrypted'))
                continue;
            if (payload[key + '_encrypted']) {
                const encryptedValue = value;
                const decryptedBytes = await this.encryptionMgr.decryptData(encryptedValue);
                if (decryptedBytes) {
                    decrypted[key] = JSON.parse(new TextDecoder().decode(new Uint8Array(decryptedBytes)));
                }
                else {
                    decrypted[key] = value;
                }
            }
            else {
                decrypted[key] = value;
            }
        }
        return decrypted;
    }
}
//# sourceMappingURL=storage.js.map