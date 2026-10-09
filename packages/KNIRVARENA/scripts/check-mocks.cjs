#!/usr/bin/env node
/**
 * CI guard (§4): fail the build if any jest.mock() path does not resolve.
 * Stale mock paths were the cause of several suites failing to even load.
 *
 * Usage: node scripts/check-mocks.cjs [rootDir]
 */

const fs = require('fs');
const path = require('path');

const root = process.argv[2] || '.';
const srcDirs = ['src', 'tests'];
const extensions = new Set(['.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs']);

const mockCallRe = /jest\.mock\(\s*(['"`])([^'"`]+)\1/g;

let checked = 0;
let failed = 0;

function resolveMockPath(fromFile, spec) {
  // jest.mock resolves relative specs against the file, like require().
  if (!spec.startsWith('.') && !path.isAbsolute(spec)) {
    return { ok: true }; // package specifier — let the module resolver judge it
  }
  const base = path.resolve(path.dirname(fromFile), spec);
  const candidates = [
    base,
    ...extensions.has(path.extname(base)) ? [] : [...extensions].map(ext => base + ext),
    ...extensions.has(path.extname(base)) ? [] : [...extensions].map(ext => path.join(base, 'index' + ext))
  ];
  return { ok: candidates.some(candidate => fs.existsSync(candidate)) };
}

function walk(dir) {
  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === 'build') continue;
      walk(full);
    } else if (extensions.has(path.extname(entry.name))) {
      const content = fs.readFileSync(full, 'utf8');
      let match;
      mockCallRe.lastIndex = 0;
      while ((match = mockCallRe.exec(content)) !== null) {
        checked++;
        const spec = match[2];
        const { ok } = resolveMockPath(full, spec);
        if (!ok) {
          failed++;
          const rel = path.relative(root, full);
          console.error(`${rel}: jest.mock('${spec}') does not resolve`);
        }
      }
    }
  }
}

for (const dir of srcDirs) {
  walk(path.join(root, dir));
}

if (failed > 0) {
  console.error(`\n${failed} unresolvable jest.mock() path(s) across ${checked} mock calls`);
  process.exit(1);
}
console.log(`check-mocks: ${checked} jest.mock() paths all resolve`);
