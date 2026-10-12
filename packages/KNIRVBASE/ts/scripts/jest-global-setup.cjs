// Build the Go conformance helper once for the whole run. Without this every
// conformance suite `go run`s it, and concurrent cold compiles make them flaky.
const { execFileSync, spawnSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

module.exports = async () => {
  if (process.env.NRVCONFORM_BIN) return; // caller supplied a binary
  if (spawnSync('go', ['version']).status !== 0) return; // suites skip (or fail under NRVCONFORM_REQUIRED)
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nrvconform-'));
  const bin = path.join(dir, 'nrvconform');
  execFileSync('go', ['build', '-o', bin, './cmd/nrvconform'], { cwd: path.resolve(__dirname, '../../go'), stdio: 'inherit' });
  process.env.NRVCONFORM_BIN = bin;
  global.__NRVCONFORM_DIR__ = dir;
};
