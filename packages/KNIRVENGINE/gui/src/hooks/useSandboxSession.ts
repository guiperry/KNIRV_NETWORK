import { useSandbox } from '../components/SandboxContext';
import type { SandboxSession, SandboxSessionStatus } from '../services/sandboxService';
import type { SandboxProxyFlow } from '../services/sandboxService';

/**
 * ## Sandbox Integration Contract
 *
 * This is the narrow seam every downstream tool integration must consume
 * instead of reaching into `SandboxContext` directly. Future Proxy / Frida /
 * Wireshark / etc. PRs should import `useSandboxSession()` and rely ONLY on the
 * fields below:
 *
 * - `session`            — the live `SandboxSession`, or `null` when none exists.
 * - `session.id`         — backend-assigned uuid of the active session.
 * - `session.status`     — `'running'` is the only "safe to operate" state.
 * - `session.targetLabel`— human name of the loaded target (Dashboard project).
 * - `session.targetCommand` — host path/command the sandbox execs.
 * - `session.netnsId`    — OPAQUE handle other tools use to mean "operate on
 *                          THIS sandbox's network/PID namespace". Today it equals
 *                          `session.id` (single-session phase); treat it as the
 *                          canonical reference and never derive a network identity
 *                          from `targetLabel` or `targetCommand`.
 * - `session.display`    — the Xvfb `DISPLAY` (e.g. `:99`).
 * - `session.vncWsPath`  — path to the Go-native VNC bridge WebSocket (docked RFB).
 * - `isReady`            — convenience: `session?.status === 'running'`.
 *
 * Do NOT block on `session` being non-null in ways that break the UI; render a
 * locked/empty state (see `RequireSandbox`) when `!isReady`.
 */
export interface SandboxSessionContract {
  session: SandboxSession | null;
  status: SandboxSessionStatus | null;
  isReady: boolean;
  proxyFlows: SandboxProxyFlow[];
}

export const useSandboxSession = (): SandboxSessionContract => {
  const { session, status, isReady, proxyFlows } = useSandbox();
  return { session, status, isReady, proxyFlows };
};

export default useSandboxSession;
