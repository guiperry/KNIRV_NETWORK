/**
 * Tool capability registry — frontend mirror of the backend lane
 * classifications. Used to select the correct hook for a tool.
 */

// Lane 6 tools run natively in the Go backend (no subprocess).
const lane6Tools = new Set([
  'tree-sitter',
  'saml-raider',
]);

// Lane 1 tools spawn a subprocess for batch scan.
const lane1Tools = new Set([
  'semgrep',
  'jadx',
  'ilspy',
  'jwt-tool',
]);

// Lane 2 tools are streaming daemons.
const lane2Tools = new Set([
  'bpftrace',
  'tshark',
  'zeek',
  'afl-fuzz',
]);

// Lane 3 tools use RPC attach.
const lane3Tools = new Set([
  'frida',
]);

// Lane 4 tools modify the sandbox launch.
const lane4Tools = new Set([
  'proxychains',
]);

// Lane 5 tools run headless analysis.
const lane5Tools = new Set([
  'cutter',
]);

export type ToolLane = 1 | 2 | 3 | 4 | 5 | 6;

export function getToolLane(tool: string): ToolLane {
  if (lane6Tools.has(tool)) return 6;
  if (lane1Tools.has(tool)) return 1;
  if (lane2Tools.has(tool)) return 2;
  if (lane3Tools.has(tool)) return 3;
  if (lane4Tools.has(tool)) return 4;
  if (lane5Tools.has(tool)) return 5;
  return 1; // default to batch scan
}

export function isLane1Tool(tool: string): boolean {
  return lane1Tools.has(tool);
}

export function isLane2Tool(tool: string): boolean {
  return lane2Tools.has(tool);
}

export function isLane3Tool(tool: string): boolean {
  return lane3Tools.has(tool);
}

export function isLane4Tool(tool: string): boolean {
  return lane4Tools.has(tool);
}

export function isLane5Tool(tool: string): boolean {
  return lane5Tools.has(tool);
}

export function isLane6Tool(tool: string): boolean {
  return lane6Tools.has(tool);
}
