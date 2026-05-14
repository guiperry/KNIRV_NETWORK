import React, { useState, useEffect } from 'react';
import './VerifierOverlay.css';
import { RewardAnchor, useKnirvana } from './stores/useKnirvana';

interface VerifierOverlayProps {
  anchor: RewardAnchor;
  onClose: () => void;
  updateRewardAnchor: (id: string, updates: Partial<RewardAnchor>) => void;
  removeRewardAnchor: (id: string) => void;
}

// ── TRL dataset templates keyed by error node type ────────────────────────

interface DatasetTemplate {
  format: string;
  trlTrainer: string;
  description: string;
  example: Record<string, unknown>;
}

const DATASET_TEMPLATES: Record<string, DatasetTemplate> = {
  'Memory Leak': {
    format: 'Prompt-Completion',
    trlTrainer: 'SFTTrainer',
    description: 'Supervised fine-tuning on correct memory-safe completions.',
    example: {
      prompt: "Fix the memory leak:\n```python\ndef load_files(paths):\n    results = []\n    for p in paths:\n        results.append(open(p).read())\n    return results\n```",
      completion: "```python\ndef load_files(paths):\n    results = []\n    for p in paths:\n        with open(p) as fh:\n            results.append(fh.read())\n    return results\n```"
    }
  },
  'Buffer Overflow': {
    format: 'Prompt-Completion',
    trlTrainer: 'SFTTrainer',
    description: 'Supervised fine-tuning on bounds-checked completions.',
    example: {
      prompt: "Add bounds checking:\n```c\nvoid copy(char* dst, char* src) {\n    while (*dst++ = *src++);\n}\n```",
      completion: "```c\nvoid safe_copy(char* dst, const char* src, size_t n) {\n    strncpy(dst, src, n - 1);\n    dst[n - 1] = '\\0';\n}\n```"
    }
  },
  'Logic Error': {
    format: 'Preference',
    trlTrainer: 'DPOTrainer',
    description: 'Direct preference optimization between correct and incorrect logic.',
    example: {
      prompt: "Sort a list of objects by their `.value` field.",
      chosen: "return sorted(items, key=lambda x: x.value)",
      rejected: "items.sort(); return items"
    }
  },
  'Race Condition': {
    format: 'Conversational Preference',
    trlTrainer: 'DPOTrainer',
    description: 'Chat-format preference training for concurrency best-practices.',
    example: {
      prompt: [{ role: 'user', content: 'How do I safely increment a shared counter in Python?' }],
      chosen: [{ role: 'assistant', content: 'Use a threading.Lock:\n```python\nwith lock:\n    counter += 1\n```' }],
      rejected: [{ role: 'assistant', content: 'Just do `counter += 1` directly.' }]
    }
  },
  'API Timeout': {
    format: 'Prompt-Only',
    trlTrainer: 'SFTTrainer (prompt-only)',
    description: 'Language-model completion training — model learns to generate retry/backoff solutions.',
    example: {
      prompt: "Implement exponential backoff (max 3 retries, base 1 s) for this API call:\n```python\ndef call_api(url):\n    return requests.get(url)\n```"
    }
  },
};

const DEFAULT_TEMPLATE: DatasetTemplate = {
  format: 'Language Modeling',
  trlTrainer: 'SFTTrainer',
  description: 'Plain text format for general language model fine-tuning.',
  example: {
    text: "The agent resolved the error by applying the following fix: <solution>"
  }
};

// ── Component ─────────────────────────────────────────────────────────────

export default function VerifierOverlay({ anchor, onClose, updateRewardAnchor, removeRewardAnchor }: VerifierOverlayProps) {
  const [weights, setWeights] = useState({
    w_c: anchor.weights.w_c,
    w_l: anchor.weights.w_l,
    w_s: anchor.weights.w_s,
  });

  const [constraints, setConstraints] = useState(
    anchor.constraints && anchor.constraints !== '// Define constraints here'
      ? anchor.constraints
      : `// DEFINE ADVERSARIAL CONSTRAINTS\nverifier.addConstraint("memory_leak_check", (agent_res) => {\n  return agent_res.usage < 128; // Max 128MB\n});\n\nverifier.setTargetOutput([104729, 104743]);`
  );

  const [copiedTemplate, setCopiedTemplate] = useState(false);
  const [activeTab, setActiveTab] = useState<'config' | 'dataset'>('config');

  const linkedNode = useKnirvana(s =>
    anchor.linkedErrorNode ? s.errorNodes.find(n => n.id === anchor.linkedErrorNode) : null
  );

  const isFullyStraightened = anchor.isCommitted && anchor.isHorizontal === false;
  const template = linkedNode?.type
    ? (DATASET_TEMPLATES[linkedNode.type] ?? DEFAULT_TEMPLATE)
    : DEFAULT_TEMPLATE;

  // Sync local state when a different anchor is selected
  useEffect(() => {
    setWeights({
      w_c: anchor.weights.w_c,
      w_l: anchor.weights.w_l,
      w_s: anchor.weights.w_s,
    });
    setConstraints(
      anchor.constraints && anchor.constraints !== '// Define constraints here'
        ? anchor.constraints
        : `// DEFINE ADVERSARIAL CONSTRAINTS\nverifier.addConstraint("memory_leak_check", (agent_res) => {\n  return agent_res.usage < 128; // Max 128MB\n});\n\nverifier.setTargetOutput([104729, 104743]);`
    );
    setCopiedTemplate(false);
  }, [anchor.id]);

  // Auto-switch to dataset tab once anchor is fully straightened
  useEffect(() => {
    if (isFullyStraightened) setActiveTab('dataset');
  }, [isFullyStraightened]);

  const updateWeights = (weightType: 'w_c' | 'w_l' | 'w_s', value: number) => {
    setWeights(prev => ({ ...prev, [weightType]: value }));
  };

  const handleSetAnchor = () => {
    updateRewardAnchor(anchor.id, { weights, constraints, isSet: true });
    onClose();
  };

  const handleCopyTemplate = () => {
    navigator.clipboard.writeText(JSON.stringify(template.example, null, 2));
    setCopiedTemplate(true);
    setTimeout(() => setCopiedTemplate(false), 2000);
  };

  return (
    <div id="verifier-overlay" className="cyber-modal">
      <div className="header">
        <h2>ANCHOR_CONFIG // ERROR_CONTEXT_DATA</h2>
        <button className="close-btn" onClick={onClose}>[X]</button>
      </div>

      {/* Anchor info */}
      <div className="anchor-info" style={{ padding: '0.5rem 1rem', borderBottom: '1px solid #333', fontSize: '0.85rem', color: '#888', display: 'flex', gap: '1.5rem', flexWrap: 'wrap' }}>
        <span>Anchor: <span style={{ color: anchor.isSet ? '#4488ff' : '#ffcc00' }}>{anchor.id}</span></span>
        <span>Position: ({anchor.position.x.toFixed(2)}, {anchor.position.z.toFixed(2)})</span>
        {anchor.linkedErrorNode && (
          <span style={{ color: '#ff6666' }}>Linked: {anchor.linkedErrorNode}</span>
        )}
        {linkedNode && (
          <span style={{ color: '#aaaaff' }}>Type: {linkedNode.type}</span>
        )}
        <span style={{ marginLeft: 'auto', color: anchor.isSet ? '#4488ff' : '#888' }}>
          {anchor.isSet ? '● SET' : '○ UNSET'}
        </span>
        {isFullyStraightened && (
          <span style={{ color: '#00ff88' }}>✓ STRAIGHTENED</span>
        )}
      </div>

      {/* Tab bar — dataset tab appears once fully straightened */}
      <div style={{ display: 'flex', borderBottom: '1px solid #333', background: '#0a0a0a' }}>
        <button
          onClick={() => setActiveTab('config')}
          style={{
            padding: '0.5rem 1.25rem',
            background: activeTab === 'config' ? '#111' : 'transparent',
            color: activeTab === 'config' ? '#00ffcc' : '#666',
            border: 'none',
            borderBottom: activeTab === 'config' ? '2px solid #00ffcc' : '2px solid transparent',
            cursor: 'pointer',
            fontSize: '0.8rem',
            fontFamily: 'monospace',
            letterSpacing: '0.05em',
          }}
        >
          ANCHOR_CONFIG
        </button>
        <button
          onClick={() => setActiveTab('dataset')}
          style={{
            padding: '0.5rem 1.25rem',
            background: activeTab === 'dataset' ? '#111' : 'transparent',
            color: isFullyStraightened
              ? (activeTab === 'dataset' ? '#00ff88' : '#44aa66')
              : '#333',
            border: 'none',
            borderBottom: activeTab === 'dataset' ? '2px solid #00ff88' : '2px solid transparent',
            cursor: isFullyStraightened ? 'pointer' : 'default',
            fontSize: '0.8rem',
            fontFamily: 'monospace',
            letterSpacing: '0.05em',
            display: 'flex',
            alignItems: 'center',
            gap: '0.4rem',
          }}
          disabled={!isFullyStraightened}
        >
          DATASET_TEMPLATE
          {!isFullyStraightened && (
            <span style={{ fontSize: '0.65rem', color: '#555' }}>(await straightening)</span>
          )}
          {isFullyStraightened && activeTab !== 'dataset' && (
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#00ff88', display: 'inline-block', marginLeft: 4 }} />
          )}
        </button>
      </div>

      {/* ── ANCHOR CONFIG tab ─────────────────────────────────────────────── */}
      {activeTab === 'config' && (
        <div className="content-grid">
          <section className="weight-shaping">
            <h3>REWARD_WEIGHTS</h3>
            <div className="slider-group">
              <label>Correctness (w_c): <span>{weights.w_c.toFixed(2)}</span></label>
              <input
                type="range"
                min="0" max="1" step="0.05"
                value={weights.w_c}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_c', parseFloat(e.target.value))}
              />
            </div>
            <div className="slider-group">
              <label>Latency (w_l): <span>{weights.w_l.toFixed(2)}</span></label>
              <input
                type="range"
                min="0" max="1" step="0.05"
                value={weights.w_l}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_l', parseFloat(e.target.value))}
              />
            </div>
            <div className="slider-group">
              <label>Simplicity (w_s): <span>{weights.w_s.toFixed(2)}</span></label>
              <input
                type="range"
                min="0" max="1" step="0.05"
                value={weights.w_s}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateWeights('w_s', parseFloat(e.target.value))}
              />
            </div>
            <p className="formula-preview">Score = (C × w_c) + (L_norm × w_l) + (S × w_s)</p>
            <button
              className="remove-anchor-btn"
              onClick={() => { removeRewardAnchor(anchor.id); onClose(); }}
            >
              REMOVE_ANCHOR
            </button>
          </section>

          <section className="constraint-editor">
            <h3>UNIT_TEST_INJECTION</h3>
            <div className="editor-container">
              <textarea
                id="test-editor"
                spellCheck={false}
                value={constraints}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setConstraints(e.target.value)}
              />
            </div>
            <button className="deploy-logic-btn" onClick={handleSetAnchor}>
              SET_ANCHOR
            </button>
          </section>
        </div>
      )}

      {/* ── DATASET TEMPLATE tab ─────────────────────────────────────────── */}
      {activeTab === 'dataset' && isFullyStraightened && (
        <div style={{ padding: '1rem', fontFamily: 'monospace', overflowY: 'auto', maxHeight: '420px' }}>
          {/* Header */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1rem' }}>
            <div>
              <div style={{ color: '#00ff88', fontSize: '0.75rem', marginBottom: '0.25rem' }}>
                ● TRL FORMAT — {template.format.toUpperCase()}
              </div>
              <div style={{ color: '#888', fontSize: '0.75rem' }}>
                Trainer: <span style={{ color: '#aaffcc' }}>{template.trlTrainer}</span>
              </div>
              <div style={{ color: '#666', fontSize: '0.7rem', marginTop: '0.25rem' }}>
                {template.description}
              </div>
            </div>
            <button
              onClick={handleCopyTemplate}
              style={{
                background: copiedTemplate ? '#004422' : '#1a1a2e',
                border: `1px solid ${copiedTemplate ? '#00ff88' : '#334'}`,
                color: copiedTemplate ? '#00ff88' : '#aaa',
                padding: '0.35rem 0.75rem',
                borderRadius: 4,
                cursor: 'pointer',
                fontSize: '0.72rem',
                fontFamily: 'monospace',
              }}
            >
              {copiedTemplate ? 'COPIED ✓' : 'COPY_JSON'}
            </button>
          </div>

          {/* All available formats */}
          <div style={{ marginBottom: '0.75rem' }}>
            <div style={{ color: '#555', fontSize: '0.65rem', marginBottom: '0.5rem', letterSpacing: '0.08em' }}>
              ALL TRL DATASET FORMATS (huggingface.co/docs/trl/dataset_formats)
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem' }}>
              {Object.entries(DATASET_TEMPLATES).map(([type, tmpl]) => (
                <span
                  key={type}
                  style={{
                    padding: '0.2rem 0.5rem',
                    borderRadius: 3,
                    fontSize: '0.65rem',
                    border: `1px solid ${linkedNode?.type === type ? '#00ff88' : '#333'}`,
                    color: linkedNode?.type === type ? '#00ff88' : '#666',
                    background: linkedNode?.type === type ? '#001a0d' : 'transparent',
                  }}
                >
                  {type} → {tmpl.format}
                </span>
              ))}
            </div>
          </div>

          {/* Example JSON */}
          <div style={{ background: '#050505', border: '1px solid #1a2a1a', borderRadius: 4, padding: '0.75rem', position: 'relative' }}>
            <div style={{ color: '#334433', fontSize: '0.65rem', marginBottom: '0.5rem', letterSpacing: '0.1em' }}>
              EXAMPLE_RECORD // {linkedNode?.type ?? 'GENERIC'}
            </div>
            <pre style={{
              color: '#aaffcc',
              fontSize: '0.72rem',
              margin: 0,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              lineHeight: 1.5,
            }}>
              {JSON.stringify(template.example, null, 2)}
            </pre>
          </div>

          {/* Format description */}
          <div style={{ marginTop: '0.75rem', padding: '0.75rem', background: '#0a0a1a', border: '1px solid #22224a', borderRadius: 4 }}>
            <div style={{ color: '#4488ff', fontSize: '0.7rem', marginBottom: '0.4rem', letterSpacing: '0.05em' }}>
              FORMAT_NOTES
            </div>
            {template.format === 'Prompt-Completion' && (
              <p style={{ color: '#668', fontSize: '0.68rem', margin: 0, lineHeight: 1.5 }}>
                Used with <span style={{ color: '#aaffcc' }}>SFTTrainer</span>. The model learns to predict{' '}
                <code style={{ color: '#ffcc44' }}>completion</code> given <code style={{ color: '#ffcc44' }}>prompt</code>.{' '}
                Ideal for code-fix and fill-in-the-middle tasks.
              </p>
            )}
            {template.format === 'Preference' && (
              <p style={{ color: '#668', fontSize: '0.68rem', margin: 0, lineHeight: 1.5 }}>
                Used with <span style={{ color: '#aaffcc' }}>DPOTrainer / ORPOTrainer</span>. Each sample pairs a{' '}
                <code style={{ color: '#00ff88' }}>chosen</code> and{' '}
                <code style={{ color: '#ff4444' }}>rejected</code> completion for the same prompt.
                The model learns to prefer correct solutions.
              </p>
            )}
            {template.format === 'Conversational Preference' && (
              <p style={{ color: '#668', fontSize: '0.68rem', margin: 0, lineHeight: 1.5 }}>
                Used with <span style={{ color: '#aaffcc' }}>DPOTrainer</span> in chat mode.{' '}
                Messages use the <code style={{ color: '#ffcc44' }}>{'[{role, content}]'}</code> schema.{' '}
                Applies the chat template before training.
              </p>
            )}
            {template.format === 'Prompt-Only' && (
              <p style={{ color: '#668', fontSize: '0.68rem', margin: 0, lineHeight: 1.5 }}>
                Used with <span style={{ color: '#aaffcc' }}>SFTTrainer</span> in completion mode.{' '}
                Only the <code style={{ color: '#ffcc44' }}>prompt</code> field is provided; the model generates the rest.{' '}
                Good for open-ended solution generation.
              </p>
            )}
            {template.format === 'Language Modeling' && (
              <p style={{ color: '#668', fontSize: '0.68rem', margin: 0, lineHeight: 1.5 }}>
                Used with <span style={{ color: '#aaffcc' }}>SFTTrainer</span> on raw{' '}
                <code style={{ color: '#ffcc44' }}>text</code> fields.{' '}
                The model trains next-token prediction on the full sequence.
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
