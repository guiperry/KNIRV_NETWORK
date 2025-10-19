// scripts/migrate.js
import Database from 'better-sqlite3';
import fs from 'fs';
import path from 'path';

const sqlite = new Database('./data/my-chat-brain.db');

async function migrate() {
  console.log('Running database migrations...');

  try {
    // Get all migration files in order
    const migrationsDir = path.resolve('./migrations');
    const files = fs.readdirSync(migrationsDir)
      .filter(file => file.endsWith('.sql'))
      .sort();

    // Execute each migration in order
    for (const file of files) {
      const filePath = path.join(migrationsDir, file);
      const sql = fs.readFileSync(filePath, 'utf8');
      
      console.log(`Applying migration: ${file}`);
      sqlite.exec(sql);
    }

    console.log('Database migrations completed successfully');
  } catch (error) {
    console.error('Error running database migrations:', error);
    process.exit(1); // Exit with an error code
  } finally {
    sqlite.close();
  }
}

migrate().catch(console.error);
