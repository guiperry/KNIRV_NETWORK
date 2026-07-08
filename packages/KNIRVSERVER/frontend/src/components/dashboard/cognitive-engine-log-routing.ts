export interface RoutedLogEntry {
  timestamp?: string;
  level?: string;
  message?: string;
  module?: string;
  source?: string;
}

export interface ProcessingActivity {
  id: string;
  timestamp: string;
  type: string;
  title: string;
  description: string;
  status: 'active' | 'completed';
}

const COGNITIVE_ENGINE_PATTERNS = [
  /cognitive engine/i,
  /cognitive-engine/i,
  /cognitive_engine/i,
  /knirv cognitive engine/i,
];

const normalizeValue = (value?: string) => (value || '').trim();

const hashString = (value: string) => {
  let hash = 0;
  for (let i = 0; i < value.length; i += 1) {
    hash = ((hash << 5) - hash) + value.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash).toString(36);
};

export function isCognitiveEngineLog(entry: RoutedLogEntry) {
  const message = normalizeValue(entry.message);
  const source = normalizeValue(entry.source);
  const moduleName = normalizeValue(entry.module);

  if (!message) {
    return false;
  }

  const haystack = `${message} ${source} ${moduleName}`;
  return COGNITIVE_ENGINE_PATTERNS.some(pattern => pattern.test(haystack));
}

export function createLogKey(entry: RoutedLogEntry) {
  return `${normalizeValue(entry.timestamp)}|${normalizeValue(entry.level).toLowerCase()}|${normalizeValue(entry.message)}`;
}

export function formatLogTimestamp(timestamp?: string) {
  return timestamp
    ? new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export function createProcessingActivity(entry: RoutedLogEntry): ProcessingActivity {
  const level = normalizeValue(entry.level).toLowerCase();
  const message = normalizeValue(entry.message);
  const timestamp = entry.timestamp ? new Date(entry.timestamp).toISOString() : new Date().toISOString();

  const title =
    level === 'error' ? 'Cognitive Engine Error' :
    level === 'warn' ? 'Cognitive Engine Warning' :
    'Cognitive Engine Activity';

  return {
    id: `processing-${hashString(`${timestamp}|${level}|${message}`)}`,
    timestamp,
    type: 'cognitive_engine_log',
    title,
    description: message,
    status: 'active',
  };
}
