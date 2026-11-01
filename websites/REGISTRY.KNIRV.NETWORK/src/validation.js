// Bootnode validation for the registry. A bootnode must pass four checks
// before it is listed: connectivity (TCP to its registered ip:port), health
// (its gateway /health answers and names this chain), security (the gateway
// serves a valid TLS certificate for its hostname) and performance (median
// latency under a limit). The same checks back POST /validate, which is the
// validation-service contract that ONBOARDING.KNIRV.COM's run-tests.ts calls.

export const VALIDATION_DEFAULTS = {
  ttlSeconds: 3600,
  failCooldownSeconds: 60,
  maxLatencyMs: 1500,
  timeoutMs: 5000,
  samples: 3,
  fqdnSuffixes: [".knirv.com"],
};

export function validationMode(env) {
  const mode = String(env?.BOOTNODE_VALIDATION ?? "enforce").trim().toLowerCase();
  return mode === "monitor" || mode === "off" ? mode : "enforce";
}

export function validationSettings(env) {
  const int = (value, fallback) => {
    const parsed = Number.parseInt(value, 10);
    return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
  };
  const suffixes = env?.VALIDATION_FQDN_SUFFIXES === undefined
    ? VALIDATION_DEFAULTS.fqdnSuffixes
    : String(env.VALIDATION_FQDN_SUFFIXES).split(",").map((s) => s.trim().toLowerCase()).filter(Boolean);
  return {
    ttlMs: int(env?.VALIDATION_TTL_SECONDS, VALIDATION_DEFAULTS.ttlSeconds) * 1000,
    failCooldownMs: int(env?.VALIDATION_FAIL_COOLDOWN_SECONDS, VALIDATION_DEFAULTS.failCooldownSeconds) * 1000,
    maxLatencyMs: int(env?.VALIDATION_MAX_LATENCY_MS, VALIDATION_DEFAULTS.maxLatencyMs),
    timeoutMs: VALIDATION_DEFAULTS.timeoutMs,
    samples: VALIDATION_DEFAULTS.samples,
    fqdnSuffixes: suffixes,
  };
}

function isPrivateIPv4(parts) {
  const [a, b] = parts;
  return a === 0 || a === 10 || a === 127 || (a === 100 && b >= 64 && b <= 127) || (a === 169 && b === 254) ||
    (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) || (a === 192 && b === 0) || (a === 198 && (b === 18 || b === 19)) || a >= 224;
}

// True only for a syntactically valid, publicly routable IP literal.
export function isPublicIP(ip) {
  if (typeof ip !== "string") return false;
  const v4 = ip.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/);
  if (v4) {
    const parts = v4.slice(1).map(Number);
    return parts.every((n) => n >= 0 && n <= 255) && !isPrivateIPv4(parts);
  }
  if (ip.includes(":") && /^[0-9a-fA-F:.]+$/.test(ip)) {
    const lower = ip.toLowerCase();
    if (lower === "::" || lower === "::1" || lower.startsWith("fe8") || lower.startsWith("fe9") || lower.startsWith("fea") || lower.startsWith("feb") ||
      lower.startsWith("fc") || lower.startsWith("fd") || lower.startsWith("ff") || lower.startsWith("::ffff:")) return false;
    return true;
  }
  return false;
}

function isAllowedHostname(host, suffixes) {
  if (typeof host !== "string" || host.length > 253) return false;
  const lower = host.toLowerCase();
  if (!/^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/.test(lower)) return false;
  return suffixes.length === 0 || suffixes.some((suffix) => lower.endsWith(suffix));
}

// Builds the validation target from a registration or /validate request.
// Returns { error } when the target cannot be probed safely.
export function buildTarget({ chainID, ip, port, fqdn, peerID }, settings) {
  if (!isPublicIP(ip)) return { error: "ip must be a public IP address" };
  if (!Number.isInteger(port) || port < 1 || port > 65535) return { error: "port must be between 1 and 65535" };
  const host = typeof fqdn === "string" && fqdn.trim() ? fqdn.trim().toLowerCase() : "";
  if (host && !isAllowedHostname(host, settings.fqdnSuffixes)) return { error: "fqdn is not an allowed hostname" };
  return { target: { chainID: chainID || "", ip, port, fqdn: host, peerID: peerID || "" } };
}

export function validationKey(target) {
  return `${target.ip}:${target.port}:${target.fqdn}`;
}

function timeoutSignal(ms) {
  return AbortSignal.timeout(ms);
}

export const defaultProbes = {
  async tcp(ip, port, timeoutMs) {
    const { connect } = await import("cloudflare:sockets");
    const started = Date.now();
    const socket = connect({ hostname: ip, port });
    try {
      await Promise.race([
        socket.opened,
        new Promise((_, reject) => setTimeout(() => reject(new Error("timeout")), timeoutMs)),
      ]);
      return { ok: true, ms: Date.now() - started };
    } catch (err) {
      return { ok: false, ms: Date.now() - started, error: String(err?.message || err) };
    } finally {
      try { await socket.close(); } catch { /* already closed */ }
    }
  },
  async https(url, timeoutMs) {
    const started = Date.now();
    try {
      const response = await fetch(url, { redirect: "manual", signal: timeoutSignal(timeoutMs), headers: { "user-agent": "KNIRV-Registry-Validator/1.0" } });
      let body = null;
      if (response.ok) body = await response.json().catch(() => null);
      return { reached: true, status: response.status, body, ms: Date.now() - started };
    } catch (err) {
      return { reached: false, ms: Date.now() - started, error: String(err?.message || err) };
    }
  },
};

function median(values) {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted.length ? sorted[Math.floor(sorted.length / 2)] : Infinity;
}

// Runs all four checks. Never throws; a probe failure is a failed check.
export async function validateNode(target, settings, probes = defaultProbes) {
  const checks = { connectivity: false, health: false, security: false, performance: false };
  const details = {};

  const tcp = await probes.tcp(target.ip, target.port, settings.timeoutMs);
  checks.connectivity = Boolean(tcp.ok);
  details.connectivity = tcp.ok ? { latencyMs: tcp.ms } : { error: tcp.error || "connection failed" };

  if (!target.fqdn) {
    details.health = { error: "fqdn is required for health, security and performance checks" };
    details.security = details.health;
    details.performance = details.health;
  } else {
    const samples = [];
    for (let i = 0; i < settings.samples; i++) {
      samples.push(await probes.https(`https://${target.fqdn}/health`, settings.timeoutMs));
    }
    const reached = samples.filter((s) => s.reached);
    // fetch only resolves over HTTPS when the certificate chain and hostname verify.
    checks.security = reached.length === samples.length;
    details.security = checks.security ? { tls: "verified", fqdn: target.fqdn } : { error: samples.find((s) => !s.reached)?.error || "TLS handshake failed" };

    const last = reached[reached.length - 1];
    const body = last?.body;
    const healthy = reached.length > 0 && last.status === 200 && body && typeof body === "object";
    let identityError = "";
    if (healthy && target.chainID && body.chainId !== target.chainID) identityError = "health chainId does not match the registered chainID";
    const peerID = body?.dht?.peer_id;
    if (healthy && !identityError && target.peerID && peerID && peerID !== target.peerID) identityError = "health peer_id does not match the registered peer";
    const addrs = Array.isArray(body?.dht?.self_addrs) ? body.dht.self_addrs : [];
    const ipBound = addrs.length > 0;
    if (healthy && !identityError && ipBound && !addrs.some((a) => String(a).includes(target.ip))) identityError = "health self_addrs do not include the registered ip";
    checks.health = Boolean(healthy) && !identityError;
    details.health = checks.health
      ? { status: last.status, ipBound, identityChecked: Boolean(target.chainID) }
      : { error: identityError || (reached.length ? `unexpected response (HTTP ${last.status})` : "health endpoint unreachable") };

    const latency = median([...(tcp.ok ? [tcp.ms] : []), ...reached.map((s) => s.ms)]);
    checks.performance = checks.connectivity && checks.security && latency <= settings.maxLatencyMs;
    details.performance = { medianLatencyMs: Number.isFinite(latency) ? latency : null, maxLatencyMs: settings.maxLatencyMs };
  }

  const passed = Object.values(checks).every(Boolean);
  return { passed, checks, details, key: validationKey(target), validatedAt: Date.now() };
}

// Constant-time string comparison for the service token.
export function tokensMatch(provided, expected) {
  if (typeof provided !== "string" || typeof expected !== "string" || !expected) return false;
  const a = new TextEncoder().encode(provided);
  const b = new TextEncoder().encode(expected);
  let diff = a.length ^ b.length;
  for (let i = 0; i < Math.max(a.length, b.length); i++) diff |= (a[i] || 0) ^ (b[i] || 0);
  return diff === 0;
}
