const Database = require('better-sqlite3');
const path = require('path');
const fs = require('fs');

// Import table names from schema
const TABLES = {
  USERS: 'users',
  SETTINGS: 'settings',
  CHAT_SESSIONS: 'chat_sessions',
  CHAT_MESSAGES: 'chat_messages',
  PROMPTS: 'prompts',
  NOTES: 'notes'
};

// Ensure data directory exists
const dataDir = path.resolve('./data');
if (!fs.existsSync(dataDir)) {
  fs.mkdirSync(dataDir, { recursive: true });
  console.log('Created data directory');
}

// Initialize SQLite database
const sqlite = new Database(path.resolve('./data/my-chat-brain.db'));

// Enable foreign key constraints
sqlite.pragma('foreign_keys = ON');

// Create tables if they don't exist
function initDb() {
  console.log('Initializing database...');

  try {
    // Create users table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.USERS} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
      );
    `);
    console.log(`${TABLES.USERS} table created or already exists`);

    // Create settings table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.SETTINGS} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        key TEXT NOT NULL,
        value TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES ${TABLES.USERS} (id)
      );
    `);
    console.log(`${TABLES.SETTINGS} table created or already exists`);

    // Create chat_sessions table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.CHAT_SESSIONS} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        title TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES ${TABLES.USERS} (id)
      );
    `);
    console.log(`${TABLES.CHAT_SESSIONS} table created or already exists`);

    // Create chat_messages table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.CHAT_MESSAGES} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id INTEGER NOT NULL,
        content TEXT NOT NULL,
        role TEXT NOT NULL,
        timestamp TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (session_id) REFERENCES ${TABLES.CHAT_SESSIONS} (id)
      );
    `);
    console.log(`${TABLES.CHAT_MESSAGES} table created or already exists`);

    // Create prompts table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.PROMPTS} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        content TEXT NOT NULL,
        title TEXT,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES ${TABLES.USERS} (id)
      );
    `);
    console.log(`${TABLES.PROMPTS} table created or already exists`);

    // Create notes table
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS ${TABLES.NOTES} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES ${TABLES.USERS} (id)
      );
    `);
    console.log(`${TABLES.NOTES} table created or already exists`);

    console.log('Database initialization completed');
  } catch (error) {
    console.error('Error initializing database:', error);
  } finally {
    sqlite.close();
  }
}

initDb();