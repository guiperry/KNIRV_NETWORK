// Runs the shared knirv:// URI vectors (testvectors/uri/uri_v1.json) against
// the compiled TypeScript twin. The Go and Rust SDKs read the same file.
import { readFile } from "node:fs/promises";
import { runUriVectors } from "../uri/uri.test.js";

const vectors = JSON.parse(await readFile(new URL("../../testvectors/uri/uri_v1.json", import.meta.url), "utf8"));
runUriVectors(vectors);
console.log(`uri vectors: ${vectors.cases.length} cases, ${vectors.derive.length} derivations`);
