// Submit TRL dataset records for a registered error node (arena_fixes.md
// §3.2). KNIRVSERVER validates the records, encodes them to NRV brackets and
// stores them in KNIRVBASE under the network error-node id.
import React, { useMemo, useState } from 'react';
import { DATASET_TEMPLATES, DEFAULT_TEMPLATE } from './game/VerifierOverlay';
import type { ArenaDatasetFormat, ArenaDatasetSubmission } from '../services/KNIRVSERVERClient';

export const DATASET_FORMATS: Array<{ id: ArenaDatasetFormat; label: string; trainer: string }> = [
  { id: 'prompt-completion', label: 'Prompt-Completion', trainer: 'SFTTrainer' },
  { id: 'preference', label: 'Preference', trainer: 'DPOTrainer' },
  { id: 'conversational-preference', label: 'Conversational Preference', trainer: 'DPOTrainer' },
  { id: 'prompt-only', label: 'Prompt-Only', trainer: 'SFTTrainer (prompt-only)' },
  { id: 'language-modeling', label: 'Language Modeling', trainer: 'SFTTrainer' },
];

/** Must match the server's per-submission limit. */
export const MAX_DATASET_RECORDS = 50;

const exampleFor = (format: ArenaDatasetFormat): string => {
  const label = DATASET_FORMATS.find((f) => f.id === format)?.label;
  const template = Object.values(DATASET_TEMPLATES).find((t) => t.format === label) ?? DEFAULT_TEMPLATE;
  return JSON.stringify([template.example], null, 2);
};

export interface DatasetTarget {
  /** Network KNIRVGRAPH error-node id. */
  errorNodeId: string;
  label: string;
}

interface Props {
  targets: DatasetTarget[];
  onSubmit: (errorNodeId: string, format: ArenaDatasetFormat, records: unknown[]) => Promise<ArenaDatasetSubmission>;
}

/** Parses the editor text; returns the records or a message. */
export function parseRecords(text: string): { records?: unknown[]; error?: string } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (err) {
    return { error: `Records are not valid JSON: ${err instanceof Error ? err.message : String(err)}` };
  }
  const records = Array.isArray(parsed) ? parsed : [parsed];
  if (records.length === 0) return { error: 'Add at least one record.' };
  if (records.length > MAX_DATASET_RECORDS) return { error: `Send at most ${MAX_DATASET_RECORDS} records at a time.` };
  if (records.some((r) => typeof r !== 'object' || r === null || Array.isArray(r))) {
    return { error: 'Each record must be a JSON object.' };
  }
  return { records };
}

export const DatasetSubmissionForm: React.FC<Props> = ({ targets, onSubmit }) => {
  const [target, setTarget] = useState(targets[0]?.errorNodeId ?? '');
  const [format, setFormat] = useState<ArenaDatasetFormat>('prompt-completion');
  const [text, setText] = useState(() => exampleFor('prompt-completion'));
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const parsed = useMemo(() => parseRecords(text), [text]);

  if (targets.length === 0) {
    return (
      <p className="p-2 text-sm text-gray-400">
        Datasets attach to error nodes registered on KNIRVGRAPH. Submit an error first (menu → Submit Error).
      </p>
    );
  }

  const changeFormat = (next: ArenaDatasetFormat) => {
    setFormat(next);
    setText(exampleFor(next));
    setMessage(null);
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!parsed.records || !target) return;
    setBusy(true);
    setMessage(null);
    try {
      const result = await onSubmit(target, format, parsed.records);
      const brackets = result.datasets.reduce((n, d) => n + d.brackets, 0);
      setMessage({ ok: true, text: `Stored ${result.records} record${result.records === 1 ? '' : 's'} as ${brackets} brackets (${result.names.join(', ')}).` });
    } catch (err) {
      const data = (err as { response?: { data?: { error?: unknown } } })?.response?.data;
      setMessage({ ok: false, text: typeof data?.error === 'string' ? data.error : err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  };

  const fieldClass = 'w-full px-2 py-1 rounded bg-gray-900 border border-gray-700 text-sm text-white';
  const trainer = DATASET_FORMATS.find((f) => f.id === format)?.trainer;

  return (
    <form onSubmit={submit} className="space-y-3 p-1">
      <label className="block text-xs text-gray-300">
        Error node
        <select className={fieldClass} value={target} onChange={(e) => setTarget(e.target.value)}>
          {targets.map((t) => <option key={t.errorNodeId} value={t.errorNodeId}>{t.label}</option>)}
        </select>
      </label>
      <label className="block text-xs text-gray-300">
        Format <span className="text-gray-500">({trainer})</span>
        <select className={fieldClass} value={format} onChange={(e) => changeFormat(e.target.value as ArenaDatasetFormat)}>
          {DATASET_FORMATS.map((f) => <option key={f.id} value={f.id}>{f.label}</option>)}
        </select>
      </label>
      <label className="block text-xs text-gray-300">
        Records (JSON array, up to {MAX_DATASET_RECORDS})
        <textarea className={`${fieldClass} h-48 font-mono text-xs`} value={text} spellCheck={false}
          onChange={(e) => { setText(e.target.value); setMessage(null); }} />
      </label>
      {parsed.error && <p className="text-xs text-amber-400">{parsed.error}</p>}
      {message && <p role={message.ok ? 'status' : 'alert'} className={`text-sm ${message.ok ? 'text-green-400' : 'text-red-400'}`}>{message.text}</p>}
      <button type="submit" disabled={busy || !parsed.records}
        className="w-full px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm">
        {busy ? 'Encoding and storing…' : 'Submit dataset'}
      </button>
    </form>
  );
};

export default DatasetSubmissionForm;
