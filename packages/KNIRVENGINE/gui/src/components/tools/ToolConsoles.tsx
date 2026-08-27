import { useCallback, useState } from 'react';
import { Download, Loader2, Play, Square } from 'lucide-react';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import { useToolAttach } from '../../hooks/useToolAttach';
import { useToolScan } from '../../hooks/useToolScan';
import { useToolStream } from '../../hooks/useToolStream';
import { runToolAnalysis } from '../../services/sandboxToolService';
import { addToolReport } from '../../services/toolReports';

const outputClass = 'min-h-[20rem] max-h-[32rem] overflow-auto whitespace-pre-wrap rounded border border-slate-700/50 bg-slate-950/60 p-3 font-mono text-xs leading-relaxed text-slate-300';

function SessionNotice({ ready }: { ready: boolean }) {
  return ready ? null : <div className="rounded border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-200">Start Bubble Wrap before using this tool.</div>;
}

// Tool acquisition happens lazily on the first run, so show an immediate,
// explicit status rather than leaving the operator with a disabled button.
// It is intentionally shared by every lane to keep messages consistent.
export function ToolPreparationNotice({ tool, active }: { tool: string; active: boolean }) {
  if (!active) return null;
  return <div role="status" aria-live="polite" className="flex items-center gap-2 rounded border border-cyan-500/30 bg-cyan-500/10 p-3 text-sm text-cyan-100"><Loader2 className="h-4 w-4 shrink-0 animate-spin" />Preparing {tool} — installing required components if they are not already available…</div>;
}

function downloadMarkdownReport(filenamePrefix: string, report: string) {
  const blob = new Blob([report], { type: 'text/markdown;charset=utf-8' });
  const href = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  const stamp = new Date().toISOString().replace(/[:.]/g, '-');
  anchor.href = href;
  anchor.download = `${filenamePrefix}-${stamp}.md`;
  anchor.click();
  URL.revokeObjectURL(href);
}

function createToolOutputMarkdownReport(title: string, tool: string, rawOutput: string, targetPath: string): string {
  return [
    `# ${title} Analysis Report`,
    '',
    `- Generated: ${new Date().toISOString()}`,
    `- Tool: ${tool}`,
    `- Target: ${targetPath || 'Launched sandbox target'}`,
    '',
    '## Tool output',
    '',
    '```text',
    rawOutput,
    '```',
    '',
  ].join('\n');
}

export function ToolScanConsole({ title, tool, native = false, token = false, nativeInputKey = 'source' }: { title: string; tool: string; native?: boolean; token?: boolean; nativeInputKey?: string }) {
  const { session, isReady } = useSandboxSession();
  const { rawOutput, structured, running, error, run } = useToolScan({ sessionID: session?.id ?? '', tool, useNative: native });
  const [targetPath, setTargetPath] = useState('');
  const [value, setValue] = useState('');
  const start = useCallback(() => {
    const args: Record<string, unknown> = targetPath ? { targetPath } : {};
    if (token) args.token = value;
    if (native && value) args[nativeInputKey] = value;
    run(args);
  }, [native, nativeInputKey, run, targetPath, token, value]);
  const output = rawOutput || (structured ? JSON.stringify(structured, null, 2) : 'No result yet.');
	const supportsMarkdownExport = tool === 'ilspy' || tool === 'jadx';
	const canExport = supportsMarkdownExport && !running && output !== 'No result yet.';

  return <div className="h-full bg-slate-900 p-6 space-y-4">
    <header className="flex items-center justify-between gap-4"><div><h1 className="text-2xl font-bold text-white">{title}</h1><p className="text-sm font-mono text-slate-400">{native ? 'native sandbox analysis' : `sandbox batch scan · ${tool}`}</p></div>
      <button onClick={start} disabled={!isReady || running || (token && !value.trim())} className="flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/20 px-3 py-2 text-sm text-emerald-300 disabled:opacity-50"><Play className="h-4 w-4" />{running ? 'Running…' : 'Run'}</button>
    </header>
    <SessionNotice ready={isReady} />
	<ToolPreparationNotice tool={title} active={running} />
    {token ? <textarea value={value} onChange={e => setValue(e.target.value)} placeholder="JWT token" className="h-24 w-full resize-none rounded border border-slate-700/50 bg-slate-950/60 p-3 font-mono text-xs text-slate-200" /> : native ? <textarea value={value} onChange={e => setValue(e.target.value)} placeholder="Source or SAML XML (leave blank to analyze the selected target)" className="h-32 w-full resize-none rounded border border-slate-700/50 bg-slate-950/60 p-3 font-mono text-xs text-slate-200" /> : <input value={targetPath} onChange={e => setTargetPath(e.target.value)} placeholder="Target path (defaults to the dashboard project mount)" className="w-full rounded border border-slate-700/50 bg-slate-950/60 px-3 py-2 font-mono text-xs text-slate-200" />}
		{supportsMarkdownExport && <button onClick={() => downloadMarkdownReport(`${tool}-analysis`, createToolOutputMarkdownReport(title, tool, output, targetPath))} disabled={!canExport} aria-label="Export Markdown report" title={canExport ? 'Export Markdown report' : 'Export becomes available after analysis'} className="inline-flex max-w-full items-center gap-2 rounded-lg border border-sky-500/40 bg-sky-500/20 px-3 py-2 text-sm text-sky-200 disabled:cursor-not-allowed disabled:opacity-50"><Download className="h-4 w-4 shrink-0" /><span className="truncate">Export report</span></button>}
    {error && <div className="rounded border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error}</div>}
    <pre className={outputClass}>{output}</pre>
  </div>;
}

export function ToolStreamConsole({ title, tool, inputLabel = 'Interface', defaultValue = 'any', inputKey = 'interface' }: { title: string; tool: string; inputLabel?: string; defaultValue?: string; inputKey?: string }) {
  const { session, isReady } = useSandboxSession();
  const { events, starting, running, error, start, stop, clearEvents } = useToolStream({ sessionID: session?.id ?? '', tool });
  const [value, setValue] = useState(defaultValue);
  const run = useCallback(() => start({ [inputKey]: value }), [inputKey, start, value]);
  return <div className="h-full bg-slate-900 p-6 space-y-4"><header className="flex items-center justify-between gap-4"><div><h1 className="text-2xl font-bold text-white">{title}</h1><p className="text-sm font-mono text-slate-400">namespace-joined live stream · {tool}</p></div><div className="flex gap-2"><button onClick={run} disabled={!isReady || running} className="flex items-center gap-2 rounded-lg border border-cyan-500/40 bg-cyan-500/20 px-3 py-2 text-sm text-cyan-200 disabled:opacity-50"><Play className="h-4 w-4" />Start</button><button onClick={stop} disabled={!running} className="flex items-center gap-2 rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 disabled:opacity-50"><Square className="h-4 w-4" />Stop</button></div></header><SessionNotice ready={isReady} /><ToolPreparationNotice tool={title} active={starting} /><label className="block text-xs font-mono text-slate-400">{inputLabel}<input value={value} onChange={e => setValue(e.target.value)} className="mt-1 w-full rounded border border-slate-700/50 bg-slate-950/60 px-3 py-2 text-slate-200" /></label>{error && <div className="rounded border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error}</div>}<div className="flex justify-end"><button onClick={clearEvents} className="text-xs text-slate-400 hover:text-white">Clear</button></div><pre className={outputClass}>{events.length ? events.map(e => `[${new Date(e.timestamp).toLocaleTimeString()}] ${e.rawLine || JSON.stringify(e.payload)}`).join('\n') : running ? 'Waiting for events…' : 'No events recorded.'}</pre></div>;
}

export function ToolAttachConsole({ title, tool }: { title: string; tool: string }) {
  const { session, isReady } = useSandboxSession();
  const { attached, attaching, pid, log, error, attach, detach, send } = useToolAttach({ sessionID: session?.id ?? '', tool });
  const [pidInput, setPidInput] = useState('');
  const [script, setScript] = useState('');
  const connect = useCallback(() => attach(Number(pidInput) || 0, script ? { script } : {}), [attach, pidInput, script]);
  return <div className="h-full bg-slate-900 p-6 space-y-4"><header className="flex items-center justify-between"><div><h1 className="text-2xl font-bold text-white">{title}</h1><p className="text-sm font-mono text-slate-400">namespace PID attach · {attached ? `attached to ${pid}` : 'detached'}</p></div><button onClick={attached ? detach : connect} disabled={!isReady || attaching} className="rounded-lg border border-cyan-500/40 bg-cyan-500/20 px-3 py-2 text-sm text-cyan-200 disabled:opacity-50">{attached ? 'Detach' : attaching ? 'Attaching…' : 'Attach'}</button></header><SessionNotice ready={isReady} /><ToolPreparationNotice tool={title} active={attaching} /><input value={pidInput} onChange={e => setPidInput(e.target.value)} placeholder="Target PID (leave empty to choose the sandbox target)" className="w-full rounded border border-slate-700/50 bg-slate-950/60 px-3 py-2 font-mono text-xs text-slate-200" /><textarea value={script} onChange={e => setScript(e.target.value)} placeholder="Frida script path inside the mounted project (optional)" className="h-36 w-full resize-none rounded border border-slate-700/50 bg-slate-950/60 p-3 font-mono text-xs text-slate-200" />{attached && <button onClick={() => send('reload', { script })} className="rounded border border-slate-600 px-2 py-1 text-xs text-slate-300">Send reload</button>}{error && <div className="rounded border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error}</div>}<pre className={outputClass}>{log.join('\n') || 'No bridge output yet.'}</pre></div>;
}

type CutterFunction = {
  name?: unknown;
  offset?: unknown;
  addr?: unknown;
  size?: unknown;
  signature?: unknown;
};

const markdownCell = (value: unknown): string => String(value ?? '—').replace(/\|/g, '\\|').replace(/\r?\n/g, ' ');

export function createCutterMarkdownReport(rawOutput: string, binaryPath: string): string {
  const createdAt = new Date().toISOString();
  let functions: CutterFunction[] = [];
  let parseNote = '';
  try {
    const parsed: unknown = JSON.parse(rawOutput);
    if (Array.isArray(parsed)) functions = parsed.filter((entry): entry is CutterFunction => typeof entry === 'object' && entry !== null);
    else parseNote = 'Rizin returned JSON that is not a function list; the raw output is included below.';
  } catch {
    parseNote = 'The Rizin output was not valid JSON; the raw output is included below.';
  }

  const rows = functions.map((fn) => {
    const address = fn.offset ?? fn.addr;
    return `| ${markdownCell(fn.name)} | ${markdownCell(address)} | ${markdownCell(fn.size)} | ${markdownCell(fn.signature)} |`;
  });
  return [
    '# Cutter Analysis Report',
    '',
    `- Generated: ${createdAt}`,
    `- Target: ${binaryPath || 'Launched sandbox target'}`,
    `- Functions discovered: ${functions.length}`,
    '',
    parseNote,
    parseNote ? '' : '',
    '## Function index',
    '',
    '| Name | Address | Size (bytes) | Signature |',
    '| --- | ---: | ---: | --- |',
    ...(rows.length ? rows : ['| No functions available | — | — | — |']),
    '',
    '## Raw Rizin output',
    '',
    '```json',
    rawOutput,
    '```',
    '',
  ].filter((line, index, lines) => line !== '' || index === 0 || lines[index - 1] !== '').join('\n');
}

function downloadCutterMarkdownReport(rawOutput: string, binaryPath: string) {
  const report = createCutterMarkdownReport(rawOutput, binaryPath);
  downloadMarkdownReport('cutter-analysis', report);
}

export function ToolHeadlessConsole({ title, tool }: { title: string; tool: string }) {
  const { session, isReady } = useSandboxSession();
  const [binaryPath, setBinaryPath] = useState('');
  const [output, setOutput] = useState('No analysis yet.');
  const [error, setError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const analyze = useCallback(async () => {
    if (!session) return;
    const startedAt = new Date().toISOString();
    const startedAtMs = Date.now();
    const args = binaryPath ? { binaryPath } : {};
    setRunning(true); setError(null); setOutput('Analyzing…');
    try {
      const result = await runToolAnalysis(session.id, tool, args);
      const reportOutput = result.rawOutput || (result.structured ? JSON.stringify(result.structured, null, 2) : 'Analysis completed with no output.');
      setOutput(reportOutput);
      addToolReport({ tool, execution: 'analysis', status: 'completed', sessionID: session.id, startedAt: result.startedAt || startedAt, completedAt: new Date().toISOString(), durationMs: result.durationMs ?? Date.now() - startedAtMs, args, output: reportOutput });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      addToolReport({ tool, execution: 'analysis', status: 'failed', sessionID: session.id, startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - startedAtMs, args, output: '', error: message });
    } finally { setRunning(false); }
  }, [binaryPath, session, tool]);
  const canExport = tool === 'cutter' && !running && output !== 'No analysis yet.' && output !== 'Analyzing…' && output !== 'Analysis completed with no output.';
  return <div className="h-full bg-slate-900 p-6 space-y-4"><header className="flex items-center justify-between"><div><h1 className="text-2xl font-bold text-white">{title}</h1><p className="text-sm font-mono text-slate-400">headless native analysis · {tool}</p></div><button onClick={analyze} disabled={!isReady || running} className="flex items-center gap-2 rounded-lg border border-red-500/40 bg-red-500/20 px-3 py-2 text-sm text-red-200 disabled:opacity-50"><Play className="h-4 w-4" />{running ? 'Analyzing…' : 'Analyze'}</button></header><SessionNotice ready={isReady} /><ToolPreparationNotice tool={title} active={running} /><input value={binaryPath} onChange={e => setBinaryPath(e.target.value)} placeholder="Binary path (defaults to the launched target)" className="w-full rounded border border-slate-700/50 bg-slate-950/60 px-3 py-2 font-mono text-xs text-slate-200" /><button onClick={() => downloadCutterMarkdownReport(output, binaryPath || session?.targetCommand || '')} disabled={!canExport} aria-label="Export Markdown report" title={canExport ? 'Export Markdown report' : 'Export becomes available after analysis'} className="inline-flex max-w-full items-center gap-2 rounded-lg border border-sky-500/40 bg-sky-500/20 px-3 py-2 text-sm text-sky-200 disabled:cursor-not-allowed disabled:opacity-50"><Download className="h-4 w-4 shrink-0" /><span className="truncate">Export report</span></button>{error && <div className="rounded border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error}</div>}<pre className={outputClass}>{output}</pre></div>;
}
