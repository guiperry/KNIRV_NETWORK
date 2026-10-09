// Provider keys (arena_fixes.md §3.5). Keys are stored in KNIRVSERVER secret
// storage and are write-only from here: the page only ever learns which
// providers have a key, never the key itself.
import React, { useCallback, useEffect, useState } from 'react';
import {
  DEFAULT_MODELS,
  authToken,
  listProviderKeys,
  removeProviderKey,
  saveProviderKey,
  type KeyedProvider,
} from '../services/serverLLM';

const PROVIDER_LABELS: Record<KeyedProvider, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Google Gemini',
  deepseek: 'DeepSeek',
  cerebras: 'Cerebras',
};

const PROVIDERS = Object.keys(DEFAULT_MODELS) as KeyedProvider[];

const errorText = (err: unknown) => (err instanceof Error ? err.message : String(err));

export const ProviderKeysPanel: React.FC = () => {
  const [stored, setStored] = useState<Set<KeyedProvider>>(new Set());
  const [drafts, setDrafts] = useState<Partial<Record<KeyedProvider, string>>>({});
  const [busy, setBusy] = useState<KeyedProvider | 'list' | null>(null);
  const [error, setError] = useState<string | null>(null);
  const signedIn = Boolean(authToken());

  const refresh = useCallback(async () => {
    if (!authToken()) return;
    setBusy('list');
    setError(null);
    try {
      setStored(new Set(await listProviderKeys()));
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(null);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  const save = async (provider: KeyedProvider) => {
    setBusy(provider);
    setError(null);
    try {
      await saveProviderKey(provider, drafts[provider] ?? '');
      setDrafts((d) => ({ ...d, [provider]: '' }));
      setStored((s) => new Set(s).add(provider));
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(null);
    }
  };

  const remove = async (provider: KeyedProvider) => {
    setBusy(provider);
    setError(null);
    try {
      await removeProviderKey(provider);
      setStored((s) => { const next = new Set(s); next.delete(provider); return next; });
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(null);
    }
  };

  if (!signedIn) {
    return <p className="p-4 text-sm text-gray-400">Sign in to KNIRV to manage provider keys.</p>;
  }

  return (
    <div className="p-4 space-y-4">
      <p className="text-xs text-gray-400">
        Keys are stored encrypted on your KNIRVSERVER and used only for your requests. They can't be read back here.
        The local model needs no key.
      </p>
      {error && <p role="alert" className="text-sm text-red-400">{error}</p>}
      {PROVIDERS.map((provider) => {
        const has = stored.has(provider);
        const draft = drafts[provider] ?? '';
        return (
          <form
            key={provider}
            className="bg-gray-800 rounded-lg p-3 space-y-2"
            onSubmit={(e) => { e.preventDefault(); void save(provider); }}
          >
            <div className="flex items-center justify-between">
              <label htmlFor={`provider-key-${provider}`} className="text-sm font-medium text-white">
                {PROVIDER_LABELS[provider]}
              </label>
              <span className={`text-xs ${has ? 'text-green-400' : 'text-gray-500'}`}>{has ? 'Key saved' : 'No key'}</span>
            </div>
            <input
              id={`provider-key-${provider}`}
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={draft}
              placeholder={has ? 'Enter a new key to replace it' : 'Paste API key'}
              onChange={(e) => setDrafts((d) => ({ ...d, [provider]: e.target.value }))}
              className="w-full px-2 py-1 rounded bg-gray-900 border border-gray-700 text-sm text-white"
            />
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={busy !== null || !draft.trim()}
                className="px-3 py-1 text-xs rounded bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white"
              >
                {busy === provider ? 'Saving…' : has ? 'Replace' : 'Save'}
              </button>
              {has && (
                <button
                  type="button"
                  disabled={busy !== null}
                  onClick={() => void remove(provider)}
                  className="px-3 py-1 text-xs rounded bg-gray-700 hover:bg-gray-600 disabled:opacity-50 text-white"
                >
                  Remove
                </button>
              )}
            </div>
          </form>
        );
      })}
    </div>
  );
};

export default ProviderKeysPanel;
