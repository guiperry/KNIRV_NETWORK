#!/usr/bin/env node
/**
 * CI guard (§3.5): fail the build if the produced arena bundle contains
 * provider key material. VITE_* keys are compiled into the bundle by
 * Vite, so this catches any that slip past the vite.config.ts guard.
 *
 * Usage: node scripts/check-bundle-secrets.cjs [distDir]
 */

const fs = require('fs');
const path = require('path');

const dist = process.argv[2] || path.join(__dirname, '..', 'dist');

// Known provider key prefixes (Google, OpenAI, Anthropic) plus generic
// long bearer-shaped tokens.
const patterns = [
  /AIza[0-9A-Za-z_-]{20,}/, // Google API key
  /sk-ant-[0-9A-Za-z_-]{20,}/, // Anthropic
  /sk-[0-9A-Za-z_-]{20,}/, // OpenAI (and others)
  /xox[baprs]-[0-9A-Za-z-]{10,}/, // Slack
  // Any non-empty secret-named VITE_ variable, whatever the key format
  // (e.g. Adaline). This is how a whole inlined `import.meta.env` leaks.
  /VITE_[A-Z0-9_]*(?:KEY|SECRET|TOKEN|PASSWORD)[A-Z0-9_]*["']?\s*:\s*["'][^"']+["']/
];

let files = 0;
let hits = 0;

function walk(dir) {
  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    console.error(`check-bundle-secrets: no dist directory at ${dist} — run the build first`);
    process.exit(1);
  }
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
    } else if (entry.name.endsWith('.js') || entry.name.endsWith('.html')) {
      files++;
      const content = fs.readFileSync(full, 'utf8');
      for (const pattern of patterns) {
        const match = content.match(pattern);
        if (match) {
          hits++;
          console.error(`${path.relative(dist, full)}: possible key material (${match[0].slice(0, 12)}…)`);
        }
      }
    }
  }
}

walk(dist);

if (hits > 0) {
  console.error(`check-bundle-secrets: ${hits} key-material match(es) in ${files} bundle files`);
  process.exit(1);
}
console.log(`check-bundle-secrets: no key material in ${files} bundle files`);
