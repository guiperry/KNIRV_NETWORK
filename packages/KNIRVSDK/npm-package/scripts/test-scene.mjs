// Verifies the shared knirv.scenes/scene-manifest golden vector from the SDK
// TypeScript signing module. Go and Rust read the same JSON.
import { readFile } from "node:fs/promises";
import { runSceneVectors } from "../scene.test.js";

const vectors = JSON.parse(await readFile(new URL("../../testvectors/models/scene_manifest_v1.json", import.meta.url), "utf8"));
await runSceneVectors(vectors);
console.log(`scene manifest vector: ${vectors.signed.address} verified`);
