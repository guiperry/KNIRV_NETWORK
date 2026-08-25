#!/usr/bin/env node

// Smoke-test the local API used by the KNIRVENGINE desktop application.
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

  console.log(`KNIRVENGINE desktop API is healthy at ${baseURL} (version ${health.version}).`);
}

main().catch((error) => {
  console.error(`KNIRVENGINE desktop API smoke test failed: ${error.message}`);
  process.exit(1);
});
