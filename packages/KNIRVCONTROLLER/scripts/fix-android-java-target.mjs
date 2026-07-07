#!/usr/bin/env node
import { readFileSync, writeFileSync, existsSync, readdirSync } from 'node:fs';
import { join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const rootDir = resolve(scriptDir, '..');
const nodeModulesDir = join(rootDir, 'node_modules');

// Capacitor plugins routinely ship with a Java target ahead of what this
// project's Android toolchain supports (JDK 17) - every plugin release can
// reintroduce this, and `npx cap sync` regenerates android/app/capacitor.build.gradle
// from whatever's currently installed. So rather than hardcoding a plugin
// list (which has already gone stale twice - @capacitor/app, then
// splash-screen/keyboard/status-bar/haptics/preferences, all missing from an
// earlier hardcoded version of this script), discover every installed
// package's android/build.gradle and patch any that declare VERSION_21.
function findPluginBuildGradleFiles(dir) {
  const found = [];
  if (!existsSync(dir)) {
    return found;
  }
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) {
      continue;
    }
    if (entry.name.startsWith('@')) {
      // Scoped packages: node_modules/@scope/pkg/android/build.gradle
      found.push(...findPluginBuildGradleFiles(join(dir, entry.name)));
      continue;
    }
    const candidate = join(dir, entry.name, 'android', 'build.gradle');
    if (existsSync(candidate)) {
      found.push(candidate);
    }
  }
  return found;
}

const targets = [
  join(rootDir, 'android', 'app', 'capacitor.build.gradle'),
  ...findPluginBuildGradleFiles(nodeModulesDir),
];

let changed = 0;

for (const target of targets) {
  if (!existsSync(target)) {
    continue;
  }

  const source = readFileSync(target, 'utf8');
  const updated = source.replace(/VERSION_21/g, 'VERSION_17');
  if (updated !== source) {
    writeFileSync(target, updated, 'utf8');
    changed += 1;
    console.log(`Updated Java target in ${target}`);
  }
}

if (changed === 0) {
  console.log('Android Java target already aligned to 17.');
}
