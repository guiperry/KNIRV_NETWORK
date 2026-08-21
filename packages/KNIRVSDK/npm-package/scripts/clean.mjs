import { rmSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
rmSync(resolve(fileURLToPath(new URL("..", import.meta.url)), "wasm"), { recursive: true, force: true });
