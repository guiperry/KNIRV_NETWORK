// Submit an Error, Context or Idea node to the network KNIRVGRAPH (§3.1, §3.6).
// Types and the description limit mirror KNIRVGRAPH's validation
// (internal/network/context_idea_rpc.go) so bad input fails here, not there.
import React, { useState } from 'react';

export type NodeKind = 'error' | 'context' | 'idea';

export const CONTEXT_TYPES = ['mcp_server', 'api_endpoint', 'tool'] as const;
export const IDEA_TYPES = ['innovation', 'improvement', 'feature', 'asset', 'characteristic', 'attribute'] as const;
export type ContextType = (typeof CONTEXT_TYPES)[number];
export type IdeaType = (typeof IDEA_TYPES)[number];

export const MAX_NODE_DESCRIPTION = 4000;

export type NodeSubmission =
  | { kind: 'error'; errorType: string; description: string; severity: 1 | 2 | 3 | 4 | 5 }
  | { kind: 'context'; contextType: ContextType; description: string }
  | { kind: 'idea'; ideaType: IdeaType; description: string };

const TYPE_LABELS: Record<string, string> = {
  mcp_server: 'MCP server', api_endpoint: 'API endpoint', tool: 'Tool',
  innovation: 'Innovation', improvement: 'Improvement', feature: 'Feature',
  asset: 'Asset', characteristic: 'Characteristic', attribute: 'Attribute',
};

const PROMPTS: Record<NodeKind, string> = {
  error: 'What went wrong? Include the error message and what you were doing.',
  context: 'What does this server, endpoint or tool provide, and how is it reached?',
  idea: 'Describe the idea and why it is worth building.',
};

interface Props {
  kind: NodeKind;
  onSubmit: (submission: NodeSubmission) => Promise<void>;
  onDone: () => void;
}

export const NodeSubmissionForm: React.FC<Props> = ({ kind, onSubmit, onDone }) => {
  const [description, setDescription] = useState('');
  const [errorType, setErrorType] = useState('');
  const [severity, setSeverity] = useState<1 | 2 | 3 | 4 | 5>(3);
  const [contextType, setContextType] = useState<ContextType>('mcp_server');
  const [ideaType, setIdeaType] = useState<IdeaType>('innovation');
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  const trimmed = description.trim();
  const valid = trimmed.length > 0 && trimmed.length <= MAX_NODE_DESCRIPTION && (kind !== 'error' || errorType.trim().length > 0);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    setBusy(true);
    setFailure(null);
    try {
      if (kind === 'error') await onSubmit({ kind, errorType: errorType.trim(), description: trimmed, severity });
      else if (kind === 'context') await onSubmit({ kind, contextType, description: trimmed });
      else await onSubmit({ kind, ideaType, description: trimmed });
      onDone();
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const fieldClass = 'w-full px-2 py-1 rounded bg-gray-900 border border-gray-700 text-sm text-white';

  return (
    <form onSubmit={submit} className="space-y-3 p-1">
      {kind === 'error' && (
        <div className="grid grid-cols-3 gap-2">
          <label className="col-span-2 text-xs text-gray-300">
            Error type
            <input className={fieldClass} value={errorType} maxLength={120} placeholder="e.g. TypeError, build failure"
              onChange={(e) => setErrorType(e.target.value)} />
          </label>
          <label className="text-xs text-gray-300">
            Severity
            <select className={fieldClass} value={severity} onChange={(e) => setSeverity(Number(e.target.value) as 1 | 2 | 3 | 4 | 5)}>
              {[1, 2, 3, 4, 5].map((n) => <option key={n} value={n}>{n}</option>)}
            </select>
          </label>
        </div>
      )}
      {kind === 'context' && (
        <label className="block text-xs text-gray-300">
          Context type
          <select className={fieldClass} value={contextType} onChange={(e) => setContextType(e.target.value as ContextType)}>
            {CONTEXT_TYPES.map((t) => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
          </select>
        </label>
      )}
      {kind === 'idea' && (
        <label className="block text-xs text-gray-300">
          Idea type
          <select className={fieldClass} value={ideaType} onChange={(e) => setIdeaType(e.target.value as IdeaType)}>
            {IDEA_TYPES.map((t) => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
          </select>
        </label>
      )}
      <label className="block text-xs text-gray-300">
        Description
        <textarea className={`${fieldClass} h-32`} value={description} maxLength={MAX_NODE_DESCRIPTION}
          placeholder={PROMPTS[kind]} onChange={(e) => setDescription(e.target.value)} />
        <span className="block text-right text-gray-500">{trimmed.length}/{MAX_NODE_DESCRIPTION}</span>
      </label>
      {failure && <p role="alert" className="text-sm text-red-400">{failure}</p>}
      <button type="submit" disabled={!valid || busy}
        className="w-full px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm">
        {busy ? 'Registering on KNIRVGRAPH…' : 'Submit'}
      </button>
    </form>
  );
};

export default NodeSubmissionForm;
