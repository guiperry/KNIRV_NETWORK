import Database from 'better-sqlite3';
import path from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Initialize SQLite database
const dbPath = path.resolve(__dirname, '../data/my-chat-brain.db');
console.log(`Checking database at: ${dbPath}`);

try {
  const db = new Database(dbPath);
  
  // Enable foreign keys
  db.pragma('foreign_keys = ON');
  
  // Check if tables exist
  const tables = db.prepare(`
    SELECT name FROM sqlite_master 
    WHERE type='table' AND name NOT LIKE 'sqlite_%'
    ORDER BY name;
  `).all();
  
  console.log('\n=== TABLES IN DATABASE ===');
  if (tables.length === 0) {
    console.log('No tables found in the database.');
  } else {
    tables.forEach(table => {
      console.log(`- ${table.name}`);
    });
  }
  
  // Expected tables
  const expectedTables = [
    'users',
    'settings',
    'chat_sessions',
    'chat_messages',
    'prompts',
    'notes'
  ];
  
  console.log('\n=== CHECKING EXPECTED TABLES ===');
  expectedTables.forEach(tableName => {
    const tableExists = tables.some(t => t.name === tableName);
    console.log(`- ${tableName}: ${tableExists ? '✅ EXISTS' : '❌ MISSING'}`);
    
    if (tableExists) {
      // Check schema of the table
      const schema = db.prepare(`PRAGMA table_info(${tableName});`).all();
      console.log(`  Schema for ${tableName}:`);
      schema.forEach(col => {
        console.log(`  - ${col.name} (${col.type}${col.notnull ? ', NOT NULL' : ''}${col.pk ? ', PRIMARY KEY' : ''})`);
      });
      
      // Check row count
      const rowCount = db.prepare(`SELECT COUNT(*) as count FROM ${tableName}`).get();
      console.log(`  Row count: ${rowCount.count}`);
      
      // Show sample data (up to 3 rows) if there are any rows
      if (rowCount.count > 0) {
        const sampleRows = db.prepare(`SELECT * FROM ${tableName} LIMIT 3`).all();
        console.log(`  Sample data (up to 3 rows):`);
        sampleRows.forEach((row, index) => {
          console.log(`  Row ${index + 1}:`, row);
        });
      }
    }
  });
  
  // Check foreign key constraints
  console.log('\n=== FOREIGN KEY CONSTRAINTS ===');
  const foreignKeys = db.prepare(`
    SELECT m.name as table_name, p."from" as column_name, 
           p."table" as referenced_table, p."to" as referenced_column
    FROM sqlite_master m
    JOIN pragma_foreign_key_list(m.name) p
    ON m.name != p."table"
    WHERE m.type = 'table'
    ORDER BY m.name;
  `).all();
  
  if (foreignKeys.length === 0) {
    console.log('No foreign key constraints found.');
  } else {
    foreignKeys.forEach(fk => {
      console.log(`- ${fk.table_name}.${fk.column_name} -> ${fk.referenced_table}.${fk.referenced_column}`);
    });
  }
  
  // Check indexes
  console.log('\n=== INDEXES ===');
  const indexes = db.prepare(`
    SELECT m.name as table_name, i.name as index_name
    FROM sqlite_master m, pragma_index_list(m.name) i
    WHERE m.type = 'table'
    ORDER BY m.name, i.name;
  `).all();
  
  if (indexes.length === 0) {
    console.log('No indexes found.');
  } else {
    indexes.forEach(idx => {
      console.log(`- ${idx.table_name}: ${idx.index_name}`);
      
      // Get columns in this index
      const indexColumns = db.prepare(`
        SELECT name FROM pragma_index_info('${idx.index_name}');
      `).all();
      
      console.log(`  Columns: ${indexColumns.map(c => c.name).join(', ')}`);
    });
  }
  
  // Database file size
  console.log('\n=== DATABASE INFORMATION ===');
  const dbInfo = db.prepare('PRAGMA database_list;').get();
  console.log(`- Database file: ${dbInfo.file}`);
  
  const pageSize = db.prepare('PRAGMA page_size;').get().page_size;
  const pageCount = db.prepare('PRAGMA page_count;').get().page_count;
  const sizeInBytes = pageSize * pageCount;
  console.log(`- Size: ${(sizeInBytes / 1024).toFixed(2)} KB (${sizeInBytes} bytes)`);
  console.log(`- Page size: ${pageSize} bytes`);
  console.log(`- Page count: ${pageCount}`);
  
  // Close the database connection
  db.close();
  console.log('\nDatabase check completed successfully.');
} catch (error) {
  console.error('Error checking database:', error);
}