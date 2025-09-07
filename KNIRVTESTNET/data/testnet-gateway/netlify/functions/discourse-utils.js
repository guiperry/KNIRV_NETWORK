const fs = require('fs').promises;
const path = require('path');
const config = require('./config-loader');

// Database abstraction layer
class DatabaseManager {
    constructor(appType = null) {
        this.dbType = config.get('DB_TYPE', 'json');

        // Determine data directory based on application type or request context
        if (appType === 'forum') {
            this.dataDir = path.join(__dirname, '..', 'forum', 'data');
        } else if (appType === 'support-desk') {
            this.dataDir = path.join(__dirname, '..', 'support-desk', 'data');
        } else {
            // Auto-detect based on calling context or default to forum
            this.dataDir = this.detectDataDirectory();
        }

        this.supabaseClient = null;

        if (this.dbType === 'supabase') {
            try {
                const { createClient } = require('@supabase/supabase-js');
                this.supabaseClient = createClient(
                    config.get('SUPABASE_URL'),
                    config.get('SUPABASE_ANON_KEY')
                );
            } catch (error) {
                console.warn('Supabase client not available, falling back to JSON database');
                this.dbType = 'json';
                this.supabaseClient = null;
            }
        }
    }

    detectDataDirectory() {
        // Try to detect which application is calling based on the request context
        // Check if we're being called from a support desk context
        const stack = new Error().stack;

        if (stack && (stack.includes('hesk') || stack.includes('support'))) {
            return path.join(__dirname, '..', 'support-desk', 'data');
        }

        // Default to forum data directory
        return path.join(__dirname, '..', 'forum', 'data');
    }

    async ensureDataDirectory() {
        try {
            await fs.access(this.dataDir);
        } catch {
            await fs.mkdir(this.dataDir, { recursive: true });
        }
    }

    async readJsonTable(tableName) {
        if (this.dbType === 'supabase') {
            const { data, error } = await this.supabaseClient
                .from(tableName)
                .select('*');
            if (error) throw error;
            return data || [];
        }

        await this.ensureDataDirectory();
        const filePath = path.join(this.dataDir, `${tableName}.json`);

        try {
            const data = await fs.readFile(filePath, 'utf8');
            return JSON.parse(data);
        } catch (error) {
            if (error.code === 'ENOENT') {
                await this.writeJsonTable(tableName, []);
                return [];
            }
            throw error;
        }
    }

    async writeJsonTable(tableName, data) {
        if (this.dbType === 'supabase') {
            return data;
        }

        await this.ensureDataDirectory();
        const filePath = path.join(this.dataDir, `${tableName}.json`);
        await fs.writeFile(filePath, JSON.stringify(data, null, 2));
        return data;
    }

    async insertRecord(tableName, record) {
        if (this.dbType === 'supabase') {
            const { data, error } = await this.supabaseClient
                .from(tableName)
                .insert([record])
                .select();
            if (error) throw error;
            return data[0];
        }

        const records = await this.readJsonTable(tableName);

        if (!record.id) {
            const maxId = records.length > 0 ? Math.max(...records.map(r => r.id || 0)) : 0;
            record.id = maxId + 1;
        }

        record.created_at = record.created_at || new Date().toISOString();
        record.updated_at = new Date().toISOString();

        records.push(record);
        await this.writeJsonTable(tableName, records);
        return record;
    }

    async updateRecord(tableName, id, updates) {
        if (this.dbType === 'supabase') {
            const { data, error } = await this.supabaseClient
                .from(tableName)
                .update({ ...updates, updated_at: new Date().toISOString() })
                .eq('id', id)
                .select();
            if (error) throw error;
            return data[0];
        }

        const records = await this.readJsonTable(tableName);
        const index = records.findIndex(r => r.id === id);

        if (index === -1) {
            throw new Error(`Record with id ${id} not found in ${tableName}`);
        }

        records[index] = {
            ...records[index],
            ...updates,
            updated_at: new Date().toISOString()
        };

        await this.writeJsonTable(tableName, records);
        return records[index];
    }

    async findRecords(tableName, query = {}) {
        if (this.dbType === 'supabase') {
            let supabaseQuery = this.supabaseClient.from(tableName).select('*');

            Object.entries(query).forEach(([key, value]) => {
                if (key === 'limit') {
                    supabaseQuery = supabaseQuery.limit(value);
                } else if (key === 'offset') {
                    supabaseQuery = supabaseQuery.range(value, value + 19);
                } else if (key === 'orderBy') {
                    supabaseQuery = supabaseQuery.order(value.column, { ascending: value.ascending !== false });
                } else if (key === '$or') {
                    value.forEach(condition => {
                        Object.entries(condition).forEach(([k, v]) => {
                            supabaseQuery = supabaseQuery.or(`${k}.eq.${v}`);
                        });
                    });
                } else {
                    supabaseQuery = supabaseQuery.eq(key, value);
                }
            });

            const { data, error } = await supabaseQuery;
            if (error) throw error;
            return data || [];
        }

        let records = await this.readJsonTable(tableName);

        // Apply filters
        Object.entries(query).forEach(([key, value]) => {
            if (key === 'limit' || key === 'offset' || key === 'orderBy') return;

            if (key === '$or') {
                records = records.filter(record => {
                    return value.some(condition => {
                        return Object.entries(condition).every(([k, v]) => {
                            if (typeof v === 'string' && v.includes('%')) {
                                const pattern = v.replace(/%/g, '.*');
                                const regex = new RegExp(pattern, 'i');
                                return regex.test(record[k]);
                            }
                            return record[k] === v;
                        });
                    });
                });
            } else {
                records = records.filter(record => {
                    if (typeof value === 'string' && value.includes('%')) {
                        const pattern = value.replace(/%/g, '.*');
                        const regex = new RegExp(pattern, 'i');
                        return regex.test(record[key]);
                    }
                    return record[key] === value;
                });
            }
        });

        // Apply ordering
        if (query.orderBy) {
            const { column, ascending = true } = query.orderBy;
            records.sort((a, b) => {
                const aVal = a[column];
                const bVal = b[column];
                const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
                return ascending ? comparison : -comparison;
            });
        }

        // Apply pagination
        if (query.offset) {
            records = records.slice(query.offset);
        }
        if (query.limit) {
            records = records.slice(0, query.limit);
        }

        return records;
    }

    async deleteRecord(tableName, id) {
        if (this.dbType === 'supabase') {
            const { error } = await this.supabaseClient
                .from(tableName)
                .delete()
                .eq('id', id);
            if (error) throw error;
            return true;
        }

        const records = await this.readJsonTable(tableName);
        const filteredRecords = records.filter(r => r.id !== id);

        if (filteredRecords.length === records.length) {
            throw new Error(`Record with id ${id} not found in ${tableName}`);
        }

        await this.writeJsonTable(tableName, filteredRecords);
        return true;
    }
}

// Create global database instance
const db = new DatabaseManager();

// Token verification
function verifyToken(token) {
    try {
        const decoded = JSON.parse(Buffer.from(token, 'base64').toString());

        if (decoded.exp < Date.now()) {
            return null;
        }

        return decoded;
    } catch (error) {
        return null;
    }
}

// HTML sanitization
function sanitizeHtml(html) {
    return html
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#x27;')
        .replace(/\//g, '&#x2F;');
}

// Email validation
function isValidEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

// Generate slug from title
function generateSlug(title) {
    return title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-|-$/g, '');
}

module.exports = {
    db,
    verifyToken,
    sanitizeHtml,
    isValidEmail,
    generateSlug
};