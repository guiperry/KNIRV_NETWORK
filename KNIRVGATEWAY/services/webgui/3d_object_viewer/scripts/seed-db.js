const Database = require('better-sqlite3');
const bcrypt = require('bcryptjs');
const path = require('path');
const fs = require('fs');

// Ensure data directory exists
const dataDir = path.resolve('./data');
if (!fs.existsSync(dataDir)) {
  fs.mkdirSync(dataDir, { recursive: true });
  console.log('Created data directory');
}

// Initialize SQLite database
const sqlite = new Database(path.resolve('./data/my-chat-brain.db'));

// Hash password function
async function hashPassword(password) {
  return await bcrypt.hash(password, 10);
}

async function seed() {
  console.log('Seeding database...');

  // Create admin user
  const hashedPassword = await hashPassword('admin123');
  
  try {
    // Check if admin user already exists
    const existingUser = sqlite.prepare('SELECT * FROM users WHERE username = ?').get('admin');
    
    if (!existingUser) {
      // Insert admin user
      const result = sqlite.prepare(`
        INSERT INTO users (username, email, password, created_at, updated_at)
        VALUES (?, ?, ?, datetime('now'), datetime('now'))
      `).run('admin', 'admin@example.com', hashedPassword);
      
      console.log('Created admin user with ID:', result.lastInsertRowid);
    } else {
      console.log('Admin user already exists');
    }
  } catch (error) {
    console.error('Error creating admin user:', error);
  }

  console.log('Database seeding completed');
}

seed().catch(console.error);