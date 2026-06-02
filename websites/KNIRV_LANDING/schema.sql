-- D1 Schema for KNIRV Landing Page Leads
-- Run: wrangler d1 execute knirv-leads --local --file=schema.sql

CREATE TABLE IF NOT EXISTS leads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    company TEXT,
    role TEXT,
    phone TEXT,
    source TEXT DEFAULT 'landing-page',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(email);
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON leads(created_at);
