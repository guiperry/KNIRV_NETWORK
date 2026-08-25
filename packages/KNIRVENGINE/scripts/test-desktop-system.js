#!/usr/bin/env node

// Verify that the desktop application's local API and sandbox service respond.
const baseURL = (process.env.KNIRVENGINE_URL || 'http://127.0.0.1:8081').replace(/\/$/, '');

async function getJSON(path) {
  const response = await fetch(`${baseURL}${path}`);
  if (!response.ok) {
    throw new Error(`${path} returned HTTP ${response.status}`);
  }
  return response.json();
}

async function main() {
  const health = await getJSON('/api/v1/health');
  if (health.status !== 'healthy') {
    throw new Error(`unexpected health status: ${health.status}`);
  }

  const sandboxes = await getJSON('/api/v1/sandboxes');
  if (!Array.isArray(sandboxes.sandboxes)) {
    throw new Error('sandbox list response is malformed');
  }

  console.log(`KNIRVENGINE desktop system is ready at ${baseURL}; ${sandboxes.sandboxes.length} sandbox session(s) active.`);
}

main().catch((error) => {
  console.error(`KNIRVENGINE desktop system smoke test failed: ${error.message}`);
  process.exit(1);
});
