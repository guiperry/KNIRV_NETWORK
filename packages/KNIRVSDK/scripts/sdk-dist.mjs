#!/usr/bin/env node
// Release bookkeeping for the SDK core. Subcommands:
//   modules   build the WASM modules; write dist/<version>/modules/{<id>.wasm,manifest.json}
//   stamp-modules  copy that manifest and bytes into core/wasm-modules (refuses to replace
//             tracked bytes unless they match, or --force / KNIRV_SDK_ALLOW_MODULE_CHANGE=1)
//   manifest  write dist/<version>/{knirv_core.wasm.sha256,manifest.json} and dist/latest/manifest.json
//   stamp     write the version and digests into the Go package and the npm package
//   check     fail unless Cargo.toml, the manifest, the Go stamp and the npm stamp agree
//   repro     clean-target rebuild of modules + both cores; compare the released digests
//
// Two builds of the one Rust core exist until the artifacts are unified: the
// WASI module (Go, published to R2) and the wasm-bindgen module (npm, inlined).
// Both are recorded, under the same version.
//
// Reproducibility note: Cargo derives each package's `-C metadata` (the crate
// disambiguator used for symbol mangling) from its absolute source path. Path
// dependencies therefore produce different bytes when the checkout is moved,
// even with --remap-path-prefix (which only rewrites file/debug path strings).
// `repro` verifies the released bytes from the current checkout and explains
// the mismatch when run from any other path.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { homedir, tmpdir } from "node:os";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const p = (...parts) => resolve(root, ...parts);
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");

function capture(cmd, args) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return "unavailable";
  }
}

function lockVersion(lockPath, name) {
  try {
    return new RegExp(`name = "${name}"\\nversion = "([^"]+)"`).exec(readFileSync(lockPath, "utf8"))?.[1] ?? "unknown";
  } catch {
    return "unknown";
  }
}

function toolchain() {
  // The web build's wasm-bindgen CLI is the one wasm-pack installs for the
  // locked crate version, not necessarily the `wasm-bindgen` on PATH.
  const webLock = p("npm-package/wasm-core/Cargo.lock");
  const wasmOptDisabled = /wasm-opt\s*=\s*false/.test(readFileSync(p("npm-package/wasm-core/Cargo.toml"), "utf8"));
  return {
    rustc: capture("rustc", ["--version"]),
    cargo: capture("cargo", ["--version"]),
    wasm_pack: capture("wasm-pack", ["--version"]),
    wasm_bindgen: lockVersion(webLock, "wasm-bindgen"),
    wasm_bindgen_path_cli: capture("wasm-bindgen", ["--version"]),
    wasm_opt: wasmOptDisabled ? "disabled (wasm-opt = false in npm-package/wasm-core/Cargo.toml)" : capture("wasm-opt", ["--version"]),
    node: process.version,
  };
}

function moduleRustflags() {
  // Remap the paths that do appear in the output (debug/panic file strings).
  // This makes them stable but cannot make the crate disambiguator stable.
  const cargoHome = process.env.CARGO_HOME ?? resolve(homedir(), ".cargo");
  return [process.env.RUSTFLAGS ?? "", `--remap-path-prefix=${root}=/knirv-sdk`, `--remap-path-prefix=${cargoHome}=/cargo`].join(" ").trim();
}

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
  // Remap absolute paths so the debug/panic strings do not depend on where the
  // checkout or the cargo registry lives (the crate disambiguator still does).
  const RUSTFLAGS = moduleRustflags();
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
  writeFileSync(resolve(distModules, "manifest.json"), `${JSON.stringify({ version: 1, source_digest: moduleSourceDigest(), modules: entries }, null, 2)}\n`);
  console.log(`modules: ${entries.map((e) => `${e.id}=${e.sha256.slice(0, 12)}`).join(" ")}`);
}

function stampModules(replaceBytes = false) {
  const manifest = JSON.parse(readFileSync(resolve(distModules, "manifest.json"), "utf8"));
  const committedPath = resolve(modulesDir, "manifest.json");
  const allowChange = replaceBytes || process.argv.includes("--force") || process.env.KNIRV_SDK_ALLOW_MODULE_CHANGE === "1";
  if (!allowChange && existsSync(committedPath)) {
    const committed = JSON.parse(readFileSync(committedPath, "utf8"));
    const diffs = [];
    for (const b of manifest.modules) {
      const c = (committed.modules ?? []).find((m) => m.id === b.id);
      if (!c) diffs.push(`${b.id}: built, not in the committed manifest`);
      else if (c.sha256 !== b.sha256) diffs.push(`${b.id}: built ${b.sha256.slice(0, 12)}, committed ${c.sha256.slice(0, 12)}`);
    }
    for (const c of committed.modules ?? []) if (!manifest.modules.some((b) => b.id === c.id)) diffs.push(`${c.id}: committed, not built`);
    if (diffs.length) {
      console.error(`stamp-modules refused: the rebuilt modules differ from core/wasm-modules/manifest.json\n  ${diffs.join("\n  ")}\n\nCargo's crate metadata includes each package's absolute source path, so a rebuild from a different checkout path produces different bytes even with --remap-path-prefix. Reproduce the released bytes from the release checkout (the release manifest, dist/${version}/manifest.json, records build_root when present) with the recorded toolchain, or pass --force / KNIRV_SDK_ALLOW_MODULE_CHANGE=1 to replace the tracked module bytes on purpose.`);
      process.exit(1);
    }
  }
  mkdirSync(resolve(modulesDir, "assets"), { recursive: true });
  for (const m of manifest.modules) copyFileSync(resolve(distModules, `${m.id}.wasm`), resolve(modulesDir, "assets", `${m.id}.wasm`));
  writeFileSync(committedPath, `${JSON.stringify(manifest, null, 2)}\n`);
  console.log("stamp-modules: core/wasm-modules/{manifest.json,assets}");
}

// Digest of everything that determines the module bytes: the module list, each
// crate's manifest, lockfile and sources (target/ excluded), and the rustc
// that compiles them. A stamp records this; `check` fails when it no longer
// matches, so the tracked bytes can only drift together with a recorded change.
// The module crates depend on the parent knirv-sdk crate by path (core/), so
// their inputs are not all under their own directory. rustc's dep-info file
// (written by each build) lists every file a module's compile actually read,
// including the path-dependency sources and include_str! data. We hash those
// files, plus both Cargo manifests and lockfiles, so any input change moves
// the digest. Requires a built module (`modules` or `sync-modules`).
function moduleDepInfoPath(m) {
  const artifact = m.artifact.replace(/\.wasm$/, ".d").replace(/-/g, "_");
  return resolve(modulesDir, m.crate, "target/wasm32-unknown-unknown/release", artifact);
}

function depInfoFiles(depPath) {
  if (!existsSync(depPath)) {
    throw new Error(`missing dep-info ${depPath}: build the modules first (node scripts/sdk-dist.mjs modules)`);
  }
  const text = readFileSync(depPath, "utf8").replace(/\\\r?\n/g, " ");
  const body = text.slice(text.indexOf(": ") + 2);
  return body.trim().split(/(?<!\\)\s+/).map((s) => s.replace(/\\ /g, " ")).filter(Boolean);
}

function moduleSourceDigest() {
  const { modules } = JSON.parse(readFileSync(p("core/wasm-modules/modules.json"), "utf8"));
  const files = new Set([p("core/Cargo.toml"), p("core/Cargo.lock"), p("core/wasm-modules/modules.json")]);
  for (const m of modules) {
    files.add(resolve(modulesDir, m.crate, "Cargo.toml"));
    files.add(resolve(modulesDir, m.crate, "Cargo.lock"));
    for (const f of depInfoFiles(moduleDepInfoPath(m))) files.add(f);
  }
  const h = createHash("sha256");
  for (const f of [...files].sort()) {
    const rel = f.startsWith(`${root}/`) ? f.slice(root.length + 1) : f;
    h.update(`${rel}\0`).update(readFileSync(f)).update("\0");
  }
  h.update(`rustc=${capture("rustc", ["--version"])}`);
  return h.digest("hex");
}

function artifact(path) {
  let buf;
  try {
    buf = readFileSync(path);
  } catch (err) {
    if (err.code === "ENOENT") {
      throw new Error(`missing artifact: ${path}\n  run \`make build-wasm\` (or \`npm run build\` in npm-package) to generate the SDK wasm output before running \`check\``);
    }
    throw err;
  }
  return { path, sha256: createHash("sha256").update(buf).digest("hex"), size: buf.byteLength };
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
    // Where this release was built. Cargo's crate disambiguator includes this
    // absolute path, so reproducing these exact bytes requires this checkout.
    build_root: root,
    toolchain: toolchain(),
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
  const current = moduleSourceDigest();
  if (committed.source_digest !== current) {
    problems.push(`core/wasm-modules: the module sources or rustc changed since the modules were stamped (stamped ${committed.source_digest ? committed.source_digest.slice(0, 12) : "none"}, current ${current.slice(0, 12)}). Run: make -C packages/KNIRVSDK sync-modules`);
  }
  if (JSON.stringify(committed) !== JSON.stringify(built)) {
    const diffs = (built.modules ?? []).filter((b) => (committed.modules ?? []).find((c) => c.id === b.id)?.sha256 !== b.sha256).map((b) => b.id);
    problems.push(`core/wasm-modules/manifest.json != dist modules manifest${diffs.length ? ` (${diffs.join(", ")})` : ""}: the tracked bytes are not what this build produces. Rebuild, then restamp with KNIRV_SDK_ALLOW_MODULE_CHANGE=1 or find the original build`);
  }
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

function repro() {
  const released = JSON.parse(readFileSync(p("dist", version, "manifest.json"), "utf8"));
  const committed = JSON.parse(readFileSync(resolve(modulesDir, "manifest.json"), "utf8"));
  const { modules } = JSON.parse(readFileSync(p("core/wasm-modules/modules.json"), "utf8"));
  const tmp = mkdtempSync(resolve(tmpdir(), "knirv-sdk-repro-"));
  const keep = process.argv.includes("--keep");
  const rows = [];
  try {
    // 1. Modules: same command and flags as `modules`.
    for (const m of modules) {
      const targetDir = resolve(tmp, `target-${m.id}`);
      execFileSync("cargo", ["build", "--release", "--locked", "--target", "wasm32-unknown-unknown", "--manifest-path", resolve(modulesDir, m.crate, "Cargo.toml")], {
        stdio: ["ignore", "ignore", "inherit"],
        env: { ...process.env, RUSTFLAGS: moduleRustflags(), CARGO_TARGET_DIR: targetDir },
      });
      rows.push({
        what: `module ${m.id}`,
        got: artifact(resolve(targetDir, "wasm32-unknown-unknown/release", m.artifact)).sha256,
        want: (committed.modules ?? []).find((x) => x.id === m.id)?.sha256,
      });
    }
    // 2. WASI core: same command as the Makefile (no path remap).
    const wasiEnv = { ...process.env, CARGO_TARGET_DIR: resolve(tmp, "target-wasi") };
    delete wasiEnv.RUSTFLAGS;
    execFileSync("cargo", ["build", "--release", "--locked", "--target", "wasm32-wasip1", "--manifest-path", p("bindings/wasi/Cargo.toml")], {
      stdio: ["ignore", "ignore", "inherit"], env: wasiEnv,
    });
    rows.push({ what: "wasi core", got: artifact(resolve(tmp, "target-wasi", "wasm32-wasip1/release/knirv_core.wasm")).sha256, want: released.sha256 });
    // 3. Web core: same command as npm-package/scripts/build.mjs.
    const webOut = resolve(tmp, "web");
    mkdirSync(webOut, { recursive: true });
    const webEnv = { ...process.env, CARGO_TARGET_DIR: resolve(tmp, "target-web") };
    delete webEnv.RUSTFLAGS;
    execFileSync("wasm-pack", ["build", "wasm-core", "--target", "web", "--release", "--out-dir", webOut, "--out-name", "knirv_sdk_core"], {
      cwd: p("npm-package"), stdio: ["ignore", "ignore", "inherit"], env: webEnv,
    });
    rows.push({ what: "web core", got: artifact(resolve(webOut, "knirv_sdk_core_bg.wasm")).sha256, want: released.web.sha256 });
  } finally {
    if (!keep) rmSync(tmp, { recursive: true, force: true });
    else console.log(`repro: kept scratch build at ${tmp}`);
  }
  const failed = rows.filter((r) => r.got !== r.want);
  for (const r of rows) {
    const ok = r.got === r.want;
    console.log(`${ok ? "ok  " : "FAIL"} ${r.what.padEnd(24)} ${r.got}${ok ? "" : ` (pinned ${r.want ?? "unknown"})`}`);
  }
  if (failed.length) {
    console.error(`\nreproduce FAILED for: ${failed.map((r) => r.what).join(", ")}`);
    console.error(`\nThe Rust build is not relocatable: Cargo's crate metadata (the crate disambiguator used in symbol mangling) includes each package's absolute source path. --remap-path-prefix only rewrites file/debug path strings, not that disambiguator, so a clean rebuild reproduces the released bytes only from the checkout the release was built at (dist/${version}/manifest.json build_root${released.build_root ? ` = ${released.build_root}` : " (not recorded)"}) with the recorded toolchain and lockfiles.`);
    console.error(`This checkout is: ${root}`);
    process.exit(1);
  }
  console.log(`\nreproduce: all ${rows.length} artifacts match the released digests from ${root}`);
}

// sync-modules: rebuild the modules and replace the tracked bytes, but only when
// the module sources or rustc changed since the last stamp (the source digest is
// the reason the bytes may change). With unchanged sources, a byte difference is
// not something a rebuild can explain, so this refuses and leaves the tree alone.
function syncModules() {
  buildModules();
  const built = JSON.parse(readFileSync(resolve(distModules, "manifest.json"), "utf8"));
  const committedPath = resolve(modulesDir, "manifest.json");
  const committed = existsSync(committedPath) ? JSON.parse(readFileSync(committedPath, "utf8")) : {};
  const bytesMatch = (committed.modules ?? []).length === built.modules.length
    && built.modules.every((b) => (committed.modules ?? []).find((c) => c.id === b.id)?.sha256 === b.sha256);
  if (committed.source_digest === built.source_digest && bytesMatch) {
    console.log("sync-modules: tracked module bytes already match the sources");
    return;
  }
  if (committed.source_digest === built.source_digest) {
    // Same sources, different bytes: not a source change, so do not replace.
    console.error("sync-modules refused: the module sources are unchanged since the last stamp, but this build produces different bytes. A rebuild cannot explain that; investigate the build inputs before replacing the tracked bytes.");
    process.exit(1);
  }
  console.log(`sync-modules: sources changed (${String(committed.source_digest ?? "none").slice(0, 12)} -> ${built.source_digest.slice(0, 12)}); replacing the tracked module bytes`);
  stampModules(true);
}

const commands = { modules: buildModules, "stamp-modules": stampModules, "sync-modules": syncModules, manifest: writeManifest, stamp, check, repro };
const run = commands[process.argv[2]];
if (!run) {
  console.error(`usage: sdk-dist.mjs ${Object.keys(commands).join("|")}`);
  process.exit(2);
}
try {
  run();
} catch (err) {
  console.error(`error: ${err && err.message ? err.message : err}`);
  process.exit(1);
}
