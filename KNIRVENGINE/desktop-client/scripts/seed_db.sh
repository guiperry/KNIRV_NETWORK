#!/bin/bash

DB_PATH="./inference.db"
ADMIN_USER="admin"
DEFAULT_PASS="ChangeMe123!"
ADMIN_PASS=${ADMIN_PASSWORD:-$DEFAULT_PASS}

# Check if sqlite3 and htpasswd are installed
if ! command -v sqlite3 &> /dev/null || ! command -v htpasswd &> /dev/null; then
  echo "Error: Required tools not found. Please install:"
  echo "  sqlite3 and apache2-utils (for htpasswd)"
  exit 1
fi

# Create users table if not exists
sqlite3 "$DB_PATH" <<EOF
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at DATETIME NOT NULL
);
EOF

# Hash password using bcrypt
HASHED_PASS=$(htpasswd -bnBC 10 "" "$ADMIN_PASS" | tr -d ':\n' | sed 's/$2y/$2a/')

# Check if admin exists
ADMIN_EXISTS=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM users WHERE username='$ADMIN_USER';" 2>/dev/null || echo 0)

if [ "$ADMIN_EXISTS" -eq 0 ]; then
  echo "Creating admin user..."
  sqlite3 "$DB_PATH" <<EOF
    INSERT INTO users (username, email, password_hash, role, created_at)
    VALUES (
      '$ADMIN_USER',
      'admin@example.com',
      '$HASHED_PASS',
      'admin',
      datetime('now')
    );
EOF
  if [ $? -eq 0 ]; then
    echo "Admin user created successfully!"
    echo "Username: $ADMIN_USER"
    echo "Password: $ADMIN_PASS"
  else
    echo "Failed to create admin user!" >&2
    exit 1
  fi
else
  echo "Admin user already exists"
fi