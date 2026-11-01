// Packs the package and checks every relative import of every shipped .js file
// resolves inside the tarball. The `files` allowlist names entry files one by
// one, so a new module that is imported but not listed ships a broken package.
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, posix } from "node:path";
import assert from "node:assert/strict";

const dir = mkdtempSync(join(tmpdir(), "knirv-pack-"));
try {
  const out = execFileSync("npm", ["pack", "--json", "--ignore-scripts", "--pack-destination", dir], { encoding: "utf8", cwd: new URL("..", import.meta.url) });
  const files = new Set(JSON.parse(out)[0].files.map((f) => f.path));
  const missing = [];
  for (const file of files) {
    if (!file.endsWith(".js")) continue;
    const source = readFileSync(new URL(`../${file}`, import.meta.url), "utf8");
    for (const match of source.matchAll(/(?:from\s+|import\s*\(\s*|import\s+)["'](\.{1,2}\/[^"']+)["']/g)) {
      const target = posix.normalize(posix.join(posix.dirname(file), match[1]));
      if (!files.has(target)) missing.push(`${file} imports ${match[1]} -> ${target}`);
    }
  }
  assert.deepEqual(missing, [], `published files import modules that are not in the package:\n${missing.join("\n")}`);
  console.log(`package contents: ${files.size} files, all relative imports resolve`);
} finally {
  rmSync(dir, { recursive: true, force: true });
}
