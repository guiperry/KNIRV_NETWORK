const test = require('node:test');
const assert = require('node:assert/strict');
const { detectPlatform, installerFileName, trimTrailingSlash } = require('../scripts/install');

test('maps supported Node platforms to published installer names', () => {
  assert.equal(installerFileName(detectPlatform('linux', 'x64')), 'knirvengine-installer-linux-amd64');
  assert.equal(installerFileName(detectPlatform('darwin', 'arm64')), 'knirvengine-installer-macos-arm64');
  assert.equal(installerFileName(detectPlatform('win32', 'x64')), 'knirvengine-installer-windows-amd64.exe');
});

test('rejects unsupported installer platforms', () => {
  assert.throws(() => detectPlatform('win32', 'arm64'), /Unsupported platform/);
  assert.throws(() => detectPlatform('freebsd', 'x64'), /Unsupported platform/);
});

test('trims release URL suffix slashes', () => {
  assert.equal(trimTrailingSlash('https://releases.knirv.com/engine/installer///'), 'https://releases.knirv.com/engine/installer');
});
