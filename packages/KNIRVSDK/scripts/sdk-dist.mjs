#!/usr/bin/env node
// Release bookkeeping for the SDK core. Subcommands:
//   modules   build the WASM modules; write dist/<version>/modules/{<id>.wasm,manifest.json}
//   stamp-modules  write that manifest and the module bytes into core/wasm-modules (read by Rust and npm)
//   manifest  write dist/<version>/{knirv_core.wasm.sha256,manifest.json} and dist/latest/manifest.json
//   stamp     write the version and digests into the Go package and the npm package
//   check     fail unless Cargo.toml, the manifest, the Go stamp and the npm stamp agree
//
// Two builds of the one Rust core exist until the artifacts are unified: the
// WASI module (Go, published to R2) and the wasm-bindgen module (npm, inlined).
// Both are recorded, under the same version.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const p = (...parts) => resolve(root, ...parts);
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");

const cargo = readFileSync(p("core/Cargo.toml"), "utf8");
const version = /^version\s*=\s*"([^"]+)"/m.exec(cargo)?.[1];
if (!version) throw new Error("no version in core/Cargo.toml");

const wasiPath = p("dist", version, "knirv_core.wasm");
const webPath = p("npm-package/wasm/knirv_sdk_core_bg.wasm");
const goStamp = p("go-package/internal/core/stamp.go");
const npmStamp = p("npm-package/core.json");

const modulesDir = p("core/wasm-modules");
const distModules = p("dist", version, "modules");

function buildModules() {
  const { modules } = JSON.parse(readFileSync(p("core/wasm-modules/modules.json"), "utf8"));
  // Remap absolute paths so the bytes do not depend on where this checkout or
  // the cargo registry lives.
  const cargoHome = process.env.CARGO_HOME ?? resolve(homedir(), ".cargo");
  const RUSTFLAGS = [process.env.RUSTFLAGS ?? "", `--remap-path-prefix=${root}=/knirv-sdk`, `--remap-path-prefix=${cargoHome}=/cargo`].join(" ").trim();
  mkdirSync(distModules, { recursive: true });
  const entries = modules.map((m) => {
    const manifestPath = resolve(modulesDir, m.crate, "Cargo.toml");
    execFileSync("cargo", ["build", "--release", "--locked", "--target", "wasm32-unknown-unknown", "--manifest-path", manifestPath], { stdio: "inherit", env: { ...process.env, RUSTFLAGS } });
    const built = resolve(modulesDir, m.crate, "target/wasm32-unknown-unknown/release", m.artifact);
    const target = resolve(distModules, `${m.id}.wasm`);
    copyFileSync(built, target);
    const { sha256: digest, size } = artifact(target);
    return { id: m.id, artifact_version: m.artifact_version, module_kind: m.module_kind, abi_version: m.abi_version, byte_length: size, sha256: digest, capabilities: m.capabilities };
  });
  writeFileSync(resolve(distModules, "manifest.json"), `${JSON.stringify({ version: 1, modules: entries }, null, 2)}\n`);
  console.log(`modules: ${entries.map((e) => `${e.id}=${e.sha256.slice(0, 12)}`).join(" ")}`);
}

function stampModules() {
  const manifest = JSON.parse(readFileSync(resolve(distModules, "manifest.json"), "utf8"));
  mkdirSync(resolve(modulesDir, "assets"), { recursive: true });
  for (const m of manifest.modules) copyFileSync(resolve(distModules, `${m.id}.wasm`), resolve(modulesDir, "assets", `${m.id}.wasm`));
  writeFileSync(resolve(modulesDir, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
  console.log("stamp-modules: core/wasm-modules/{manifest.json,assets}");
}

function artifact(path) {
  const bytes = readFileSync(path);
  return { sha256: sha256(bytes), size: bytes.length };
}

function manifest() {
  let revision = "unknown";
  try {
    revision = execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
  } catch {}
  return {
    version,
    sha256: artifact(wasiPath).sha256, // the R2 object: knirv_core.wasm
    size: artifact(wasiPath).size,
    web: artifact(webPath), // the inlined npm module
    git_revision: revision,
    built_at: new Date().toISOString(),
  };
}

function writeManifest() {
  const m = manifest();
  const dir = p("dist", version);
  mkdirSync(dir, { recursive: true });
  writeFileSync(p("dist", version, "knirv_core.wasm.sha256"), `${m.sha256}  knirv_core.wasm\n`);
  writeFileSync(p("dist", version, "manifest.json"), `${JSON.stringify(m, null, 2)}\n`);
  // latest is a pointer only: it names a version and digest, never holds bytes.
  mkdirSync(p("dist/latest"), { recursive: true });
  writeFileSync(p("dist/latest/manifest.json"), `${JSON.stringify(m, null, 2)}\n`);
  console.log(`manifest: ${version} wasi=${m.sha256} web=${m.web.sha256}`);
}

function stamp() {
  const m = JSON.parse(readFileSync(p("dist", version, "manifest.json"), "utf8"));
  writeFileSync(
    goStamp,
    `// Code generated by \`make stamp\` in KNIRVSDK; DO NOT EDIT.

package core

const (
	// Version is the core/Cargo.toml version this package is pinned to.
	Version = "${m.version}"
	// Digest is the SHA-256 of that version's knirv_core.wasm.
	Digest = "${m.sha256}"
)
`,
  );
  writeFileSync(npmStamp, `${JSON.stringify({ version: m.version, sha256: m.web.sha256 }, null, 2)}\n`);
  console.log(`stamp: go + npm pinned to ${m.version}`);
}

function check() {
  const problems = [];
  const m = JSON.parse(readFileSync(p("dist", version, "manifest.json"), "utf8"));
  if (m.version !== version) problems.push(`manifest version ${m.version} != Cargo.toml ${version}`);
  if (m.sha256 !== artifact(wasiPath).sha256) problems.push("manifest digest != dist/knirv_core.wasm");
  const go = readFileSync(goStamp, "utf8");
  if (!go.includes(`Version = "${version}"`)) problems.push("Go stamp version != Cargo.toml");
  if (!go.includes(`Digest = "${m.sha256}"`)) problems.push("Go stamp digest != manifest");
  const committed = JSON.parse(readFileSync(resolve(modulesDir, "manifest.json"), "utf8"));
  const built = JSON.parse(readFileSync(resolve(distModules, "manifest.json"), "utf8"));
  if (JSON.stringify(committed) !== JSON.stringify(built)) problems.push("core/wasm-modules/manifest.json != dist modules manifest");
  for (const m of committed.modules) {
    const a = artifact(resolve(modulesDir, "assets", `${m.id}.wasm`));
    if (a.sha256 !== m.sha256 || a.size !== m.byte_length) problems.push(`core asset ${m.id} != manifest`);
    const n = artifact(p("npm-package/modules", `${m.id}.wasm`));
    if (n.sha256 !== m.sha256) problems.push(`npm module ${m.id} != manifest`);
  }
  const npm = JSON.parse(readFileSync(npmStamp, "utf8"));
  if (npm.version !== version) problems.push("npm core.json version != Cargo.toml");
  if (npm.sha256 !== m.web.sha256) problems.push("npm core.json digest != manifest");
  if (npm.sha256 !== artifact(webPath).sha256) problems.push("npm core.json digest != the module npm inlines");
  if (problems.length) {
    console.error(`check failed:\n  ${problems.join("\n  ")}`);
    process.exit(1);
  }
  console.log(`check: Cargo.toml, manifest, Go and npm all at ${version}`);
}

const commands = { modules: buildModules, "stamp-modules": stampModules, manifest: writeManifest, stamp, check };
const run = commands[process.argv[2]];
if (!run) {
  console.error(`usage: sdk-dist.mjs ${Object.keys(commands).join("|")}`);
  process.exit(2);
}
run();
