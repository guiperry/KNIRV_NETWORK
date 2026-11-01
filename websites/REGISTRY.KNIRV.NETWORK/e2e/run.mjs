// End-to-end test for bootnode validation. Runs the REAL registry code in
// workerd (wrangler dev) against mock bootnodes on loopback.
//
// Two things in the build copy are patched because loopback can't satisfy them
// (src/ itself is never modified): isPublicIP accepts any IPv4 literal, and the
// HTTPS health probe goes to http://127.0.0.1:<port>/health, where <port> is
// taken from the fqdn label ("p18081.knirv.com" -> 18081). Everything else --
// register gate, TCP connect via cloudflare:sockets, health identity checks,
// latency, cooldown, cache, listing filter, /validate -- is the shipped code.
//
//   node e2e/run.mjs
import { spawn } from "node:child_process";
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import http from "node:http";
import net from "node:net";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const build = path.join(root, "e2e", ".build");
const REG = "http://127.0.0.1:18788";

// ---- build patched copy ----------------------------------------------------
rmSync(build, { recursive: true, force: true });
mkdirSync(path.join(build, "src"), { recursive: true });
for (const f of ["index.js", "validation.js"]) cpSync(path.join(root, "src", f), path.join(build, "src", f));
cpSync(path.join(root, "wrangler.jsonc"), path.join(build, "wrangler.jsonc"));
function patch(file, from, to) {
  const p = path.join(build, "src", file);
  const s = readFileSync(p, "utf8");
  if (!s.includes(from)) throw new Error(`patch target not found in ${file}: ${from}`);
  writeFileSync(p, s.replace(from, to));
}
patch("validation.js", 'if (typeof ip !== "string") return false;', 'if (typeof ip === "string" && /^\\d+\\.\\d+\\.\\d+\\.\\d+$/.test(ip)) return true; if (typeof ip !== "string") return false;');
patch("validation.js", "`https://${target.fqdn}/health`", "`http://127.0.0.1:${target.fqdn.split(\".\")[0].slice(1)}/health`");

// ---- mock bootnodes --------------------------------------------------------
const servers = [];
const hits = {};
function mockBootnode({ tcpPort, httpPort, chainId, peerId, listenTcp = true }) {
  hits[httpPort] = 0;
  if (listenTcp) servers.push(net.createServer((s) => s.end()).listen(tcpPort, "127.0.0.1"));
  servers.push(http.createServer((req, res) => {
    hits[httpPort]++;
    res.setHeader("content-type", "application/json");
    res.end(JSON.stringify({ status: "ok", chainId, dht: { peer_id: peerId, self_addrs: [`/ip4/127.0.0.1/tcp/${tcpPort}`] } }));
  }).listen(httpPort, "127.0.0.1"));
}
mockBootnode({ tcpPort: 14001, httpPort: 18081, chainId: "chain-good", peerId: "peer-good" });           // passes everything
mockBootnode({ tcpPort: 14002, httpPort: 18082, chainId: "some-other-chain", peerId: "peer-bad" });      // wrong chain
mockBootnode({ tcpPort: 14003, httpPort: 18083, chainId: "chain-closed", peerId: "peer-closed", listenTcp: false }); // P2P port closed
mockBootnode({ tcpPort: 14004, httpPort: 18084, chainId: "chain-peer", peerId: "peer-actual" });         // peer id mismatch

// ---- registry in workerd ---------------------------------------------------
const wrangler = spawn(path.join(root, "node_modules/.bin/wrangler"), ["dev", "--local", "--port", "18788", "--persist-to", path.join(build, ".state")], { cwd: build, stdio: ["ignore", "pipe", "pipe"] });
let wlog = "";
wrangler.stdout.on("data", (d) => (wlog += d));
wrangler.stderr.on("data", (d) => (wlog += d));

let failed = 0, passed = 0;
function check(name, cond, extra) {
  if (cond) { passed++; console.log(`  PASS  ${name}`); } else { failed++; console.log(`  FAIL  ${name}${extra ? "  -> " + JSON.stringify(extra) : ""}`); }
}
const post = async (p, body, headers = {}) => {
  const r = await fetch(REG + p, { method: "POST", headers: { "content-type": "application/json", ...headers }, body: JSON.stringify(body) });
  return { status: r.status, retryAfter: r.headers.get("retry-after"), body: await r.json().catch(() => null) };
};
const get = async (p) => (await fetch(REG + p)).json();
const bootnode = (n) => ({ chainID: n.chainID, type: "bootnode", ip: "127.0.0.1", port: n.port, nodeID: n.peer, fqdn: n.fqdn });
const GOOD = { chainID: "chain-good", port: 14001, peer: "peer-good", fqdn: "p18081.knirv.com" };
const BAD_CHAIN = { chainID: "chain-bad", port: 14002, peer: "peer-bad", fqdn: "p18082.knirv.com" };
const CLOSED = { chainID: "chain-closed", port: 14003, peer: "peer-closed", fqdn: "p18083.knirv.com" };
const BAD_PEER = { chainID: "chain-peer", port: 14004, peer: "peer-registered", fqdn: "p18084.knirv.com" };

async function main() {
  for (let i = 0; i < 60; i++) {
    try { if ((await fetch(REG + "/status")).ok) break; } catch { /* not up yet */ }
    if (wrangler.exitCode !== null) throw new Error("wrangler dev exited:\n" + wlog);
    await new Promise((r) => setTimeout(r, 500));
    if (i === 59) throw new Error("registry did not start:\n" + wlog);
  }
  console.log("registry up (workerd)\n");

  console.log("1. valid bootnode registers");
  let r = await post("/register", bootnode(GOOD));
  check("201 registered", r.status === 201, r);
  const listed = JSON.stringify(await get("/bootnodes"));
  check("listed in /bootnodes", listed.includes("chain-good"), listed);
  check("listed in /nodes", JSON.stringify(await get("/nodes")).includes("chain-good"));
  const status = await get("/status");
  check("/status shows it validated", JSON.stringify(status).includes("chain-good"), status);

  console.log("2. validation is cached on re-registration");
  const before = hits[18081];
  r = await post("/register", bootnode(GOOD));
  check("201 registered", r.status === 201, r);
  check("no new health probes", hits[18081] === before, { before, after: hits[18081] });

  console.log("3. wrong chainId in /health is rejected");
  r = await post("/register", bootnode(BAD_CHAIN));
  check("422", r.status === 422, r);
  check("health check failed, connectivity passed", r.body?.validation?.checks?.health === false && r.body?.validation?.checks?.connectivity === true, r.body);
  check("not listed", !JSON.stringify(await get("/bootnodes")).includes("chain-bad"));

  console.log("4. closed P2P port is rejected");
  r = await post("/register", bootnode(CLOSED));
  check("422", r.status === 422, r);
  check("connectivity failed", r.body?.validation?.checks?.connectivity === false, r.body);
  console.log("5. failed node is rate limited");
  r = await post("/register", bootnode(CLOSED));
  check("429 with Retry-After", r.status === 429 && Number(r.retryAfter) > 0, r);

  console.log("6. peer id mismatch is rejected");
  r = await post("/register", bootnode(BAD_PEER));
  check("422", r.status === 422 && r.body?.validation?.checks?.health === false, r);

  console.log("7. bootnode without fqdn is rejected");
  r = await post("/register", { ...bootnode({ chainID: "chain-nofqdn", port: 14001, peer: "x" }), fqdn: undefined });
  check("422", r.status === 422, r);

  console.log("8. clients are unaffected");
  r = await post("/register", { chainID: "chain-good", type: "client", ip: "198.51.100.7", port: 5000, nodeID: "client-1" });
  check("client registers without validation (201)", r.status === 201, r);

  console.log("9. POST /validate (onboarding contract)");
  r = await post("/validate", { registrationId: "reg-1", fqdn: GOOD.fqdn, publicIps: ["127.0.0.1"], expectedPorts: [14001] });
  const c = r.body?.checks || {};
  check("all four checks pass", r.status === 200 && c.connectivity && c.health && c.security && c.performance, r);
  r = await post("/validate", { registrationId: "reg-2", fqdn: CLOSED.fqdn, publicIps: ["127.0.0.1"], expectedPorts: [14003] });
  check("closed port fails connectivity", r.body?.checks?.connectivity === false, r);

  console.log(`\n${passed} passed, ${failed} failed`);
}

let code = 1;
try { await main(); code = failed ? 1 : 0; } catch (e) { console.error(e); }
finally {
  wrangler.kill("SIGTERM");
  servers.forEach((s) => s.close());
  await new Promise((r) => setTimeout(r, 500));
  rmSync(build, { recursive: true, force: true });
  process.exit(code);
}
