import { useSyncExternalStore } from 'react';

const STORAGE_KEY = 'knirvengine.tool-reports.v1';
const MAX_REPORTS = 200;

export type ToolReportStatus = 'completed' | 'failed';

export interface ToolReport {
  id: string;
  tool: string;
  execution: 'scan' | 'stream' | 'attach' | 'analysis';
  status: ToolReportStatus;
  sessionID: string;
  startedAt: string;
  completedAt: string;
  durationMs?: number;
  args?: Record<string, unknown>;
  output: string;
  error?: string;
}

let reports: ToolReport[] | null = null;
const listeners = new Set<() => void>();

const getStorage = (): Storage | null => {
  if (typeof window === 'undefined') return null;
  try {
    return window.localStorage;
  } catch {
    return null;
  }
};

const loadReports = (): ToolReport[] => {
  if (reports) return reports;
  const storage = getStorage();
  if (!storage) return (reports = []);
  try {
    const parsed: unknown = JSON.parse(storage.getItem(STORAGE_KEY) || '[]');
    reports = Array.isArray(parsed) ? parsed.filter(isToolReport).slice(0, MAX_REPORTS) : [];
  } catch {
    reports = [];
  }
  return reports;
};

const isToolReport = (value: unknown): value is ToolReport => {
  if (!value || typeof value !== 'object') return false;
  const report = value as Partial<ToolReport>;
  return typeof report.id === 'string' && typeof report.tool === 'string' &&
    typeof report.execution === 'string' && typeof report.status === 'string' &&
    typeof report.startedAt === 'string' && typeof report.completedAt === 'string' &&
    typeof report.output === 'string';
};

const persist = () => {
  const storage = getStorage();
  if (!storage) return;
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(reports));
  } catch {
    // A full or unavailable storage area must not block tool execution.
  }
};

const notify = () => listeners.forEach((listener) => listener());

const redactArgs = (args?: Record<string, unknown>): Record<string, unknown> | undefined => {
  if (!args) return undefined;
  return Object.fromEntries(Object.entries(args).map(([key, value]) => [
    key,
    /token|password|secret|credential/i.test(key) ? '[redacted]' : value,
  ]));
};

export const addToolReport = (report: Omit<ToolReport, 'id' | 'args'> & { args?: Record<string, unknown> }): ToolReport => {
  const entry: ToolReport = {
    ...report,
    id: typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : `${report.tool}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    args: redactArgs(report.args),
  };
  reports = [entry, ...loadReports()].slice(0, MAX_REPORTS);
  persist();
  notify();
  return entry;
};

export const clearToolReports = () => {
  reports = [];
  persist();
  notify();
};

export const getToolReports = () => loadReports();

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => listeners.delete(listener);
};

export const useToolReports = () => useSyncExternalStore(subscribe, getToolReports, () => []);

export const createCombinedToolReport = (entries: ToolReport[]): string => [
  '# KNIRVENGINE Tool Reports',
  '',
  `- Generated: ${new Date().toISOString()}`,
  `- Reports included: ${entries.length}`,
  '',
  ...entries.flatMap((entry) => [
    `## ${entry.tool} — ${entry.status}`,
    '',
    `- Execution: ${entry.execution}`,
    `- Started: ${entry.startedAt}`,
    `- Completed: ${entry.completedAt}`,
    `- Duration: ${entry.durationMs ?? 'unknown'} ms`,
    entry.sessionID ? `- Sandbox session: ${entry.sessionID}` : '',
    entry.error ? `- Error: ${entry.error}` : '',
    '',
    '### Output',
    '',
    '```text',
    entry.output,
    '```',
    '',
  ].filter(Boolean)),
].join('\n');
