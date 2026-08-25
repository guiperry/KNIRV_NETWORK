import React, { useEffect, useRef, useState } from 'react';
import { Box, Check, Clipboard, FolderOpen, Play, Square } from 'lucide-react';
import { useSandbox } from '../../SandboxContext';

interface Bind {
  id: number;
  mode: 'ro-bind' | 'bind' | 'tmpfs';
  src: string;
  dst: string;
}

type ElectronWindow = Window & {
  electronAPI?: { selectSandboxBinary?: (defaultPath?: string) => Promise<string | null> };
};

const Bubblewrap: React.FC = () => {
  const { launch, stop, status, log, targetLabel, projectPath, projectTargetPath, error, deps, depsInstalling, installDeps } = useSandbox();
  const running = status === 'running' || status === 'provisioning';
  const missingDeps = (deps ?? []).filter((d) => !d.present);

  const [binds, setBinds] = useState<Bind[]>([
    { id: 1, mode: 'ro-bind', src: '/usr', dst: '/usr' },
    { id: 2, mode: 'ro-bind', src: '/lib', dst: '/lib' },
    { id: 3, mode: 'ro-bind', src: '/lib64', dst: '/lib64' },
    { id: 4, mode: 'tmpfs', src: '-', dst: '/tmp' },
  ]);
  const [unshareAll, setUnshareAll] = useState(true);
  const [shareNet, setShareNet] = useState(true);
  const [dieWithParent, setDieWithParent] = useState(true);
  const [display, setDisplay] = useState(':99');
  const [target, setTarget] = useState('');
  const [executionMode, setExecutionMode] = useState<'binary' | 'command'>('binary');
  const [runtime, setRuntime] = useState('node');
  const [customRuntime, setCustomRuntime] = useState('');
  const [scriptPath, setScriptPath] = useState('');
  const [commandArguments, setCommandArguments] = useState('');
  const [selectedTargetDirectory, setSelectedTargetDirectory] = useState('');
  const [copyStatus, setCopyStatus] = useState<'idle' | 'copied' | 'failed'>('idle');
  const [pickerMessage, setPickerMessage] = useState('');
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const targetCommand = executionMode === 'command' ? (runtime === 'custom' ? customRuntime : runtime) : target;
  const targetArgs = executionMode === 'command'
    ? [scriptPath, ...commandArguments.split('\n').map((arg) => arg.trim()).filter(Boolean)]
    : [];
  const automaticMounts = [
    projectPath,
    selectedTargetDirectory,
  ].filter((path, index, paths) => Boolean(path) && paths.indexOf(path) === index);
  const launchBinds = automaticMounts.reduce<Bind[]>(
    (configuredBinds, directory) => configuredBinds.some((bind) => bind.src === directory)
      ? configuredBinds
      : [...configuredBinds, { id: -configuredBinds.length - 1, mode: 'ro-bind', src: directory, dst: directory }],
    binds,
  );
  const command = [
    'bwrap',
    ...launchBinds.flatMap(b => b.mode === 'tmpfs' ? [`--tmpfs ${b.dst}`] : [`--${b.mode} ${b.src} ${b.dst}`]),
    '--proc /proc',
    '--dev /dev',
    unshareAll ? '--unshare-all' : '',
    shareNet ? '--share-net' : '',
    dieWithParent ? '--die-with-parent' : '',
    `--setenv DISPLAY ${display}`,
    '--',
    targetCommand,
    ...targetArgs,
  ].filter(Boolean).join(' \\\n  ');

  useEffect(() => {
    if (!projectTargetPath) return;
    if (executionMode === 'command') setScriptPath(projectTargetPath);
    else setTarget(projectTargetPath);
    const separator = Math.max(projectTargetPath.lastIndexOf('/'), projectTargetPath.lastIndexOf('\\'));
    setSelectedTargetDirectory(separator > 0 ? projectTargetPath.slice(0, separator) : '');
    setPickerMessage('The dashboard-selected file and its project directory are mounted read-only automatically.');
  }, [executionMode, projectTargetPath]);

  const launchSandbox = async () => {
    try {
      await launch({
        targetLabel: targetLabel || (executionMode === 'command' ? scriptPath : target).split('/').pop() || 'sandbox-target',
        targetCommand,
        targetArgs,
        display,
        binds: launchBinds,
        unshareAll,
        shareNet,
        dieWithParent,
      });
    } catch {
      // The context surfaces the backend error in the namespace log panel.
    }
  };

  const stopSandbox = async () => {
    try {
      await stop();
    } catch {
      /* best effort */
    }
  };

  const selectTargetBinary = async () => {
    const electronAPI = (window as ElectronWindow).electronAPI;
    if (electronAPI?.selectSandboxBinary) {
      const selectedPath = await electronAPI.selectSandboxBinary(projectPath || undefined);
      if (selectedPath) {
        if (executionMode === 'command') setScriptPath(selectedPath);
        else setTarget(selectedPath);
        setSelectedTargetDirectory(selectedPath.slice(0, selectedPath.lastIndexOf('/')));
        setPickerMessage('Selected file directory will be mounted read-only automatically.');
      }
      return;
    }
    fileInputRef.current?.click();
  };

  const handleBrowserFileSelection = (event: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0];
    if (!selectedFile) return;
    const path = (selectedFile as File & { path?: string }).path;
    if (path) {
      if (executionMode === 'command') setScriptPath(path);
      else setTarget(path);
      setSelectedTargetDirectory(path.slice(0, path.lastIndexOf('/')));
      setPickerMessage('Selected file directory will be mounted read-only automatically.');
    } else {
      setPickerMessage('Browsers do not reveal absolute local paths. Run KNIRVENGINE in Electron to select a launchable binary.');
    }
    event.target.value = '';
  };

  const copyLog = async () => {
    const text = [error, ...log].filter((line): line is string => Boolean(line)).join('\n');
    if (!text) return;

    try {
      await navigator.clipboard.writeText(text);
      setCopyStatus('copied');
    } catch {
      // Electron and non-secure browser contexts may not expose Clipboard API.
      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      const copied = document.execCommand('copy');
      textarea.remove();
      setCopyStatus(copied ? 'copied' : 'failed');
    }
  };

  const addBind = () => setBinds(prev => [...prev, { id: Date.now(), mode: 'ro-bind', src: '', dst: '' }]);
  const removeBind = (id: number) => setBinds(prev => prev.filter(b => b.id !== id));

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-fuchsia-500/20 rounded-lg">
            <Box className="w-6 h-6 text-fuchsia-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Bubblewrap</h1>
            <p className="text-slate-400 text-sm font-mono">unprivileged namespace sandbox</p>
          </div>
        </div>
        <button
          onClick={running ? stopSandbox : launchSandbox}
          className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-fuchsia-500/20 border border-fuchsia-500/40 text-fuchsia-300 hover:bg-fuchsia-500/30"
        >
          {running ? <Square className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          <span>{running ? 'Stop' : 'Launch'}</span>
        </button>
      </div>

      {missingDeps.length > 0 && (
        <div className="bg-amber-500/10 border border-amber-500/40 rounded-lg p-4 space-y-2">
          {depsInstalling ? (
            <div className="flex items-center space-x-2 text-amber-300 text-sm">
              <span className="animate-pulse">Installing sandbox dependencies…</span>
            </div>
          ) : (
            <>
              <div className="text-amber-300 text-sm font-medium">
                Some sandbox dependencies are missing. KNIRVENGINE can install them automatically:
              </div>
              <ul className="text-xs font-mono text-amber-200/80 space-y-1">
                {missingDeps.map((d) => (
                  <li key={d.binary}>
                    {d.binary}
                    {d.installCommand && (
                      <span className="text-amber-400/60"> — {d.installCommand}</span>
                    )}
                    {d.error && <span className="text-red-400"> — {d.error}</span>}
                  </li>
                ))}
              </ul>
              <button
                onClick={installDeps}
                className="mt-1 px-3 py-1.5 rounded-lg text-sm font-medium bg-amber-500/20 border border-amber-500/40 text-amber-300 hover:bg-amber-500/30"
              >
                Install dependencies
              </button>
            </>
          )}
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 space-y-4">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500 uppercase">Bind mounts</span>
              <button onClick={addBind} className="text-xs text-fuchsia-400 hover:text-fuchsia-300">+ add</button>
            </div>
            <div className="space-y-1.5">
              {binds.map(b => (
                <div key={b.id} className="flex items-center space-x-2 font-mono text-xs">
                  <select
                    value={b.mode}
                    onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, mode: e.target.value as Bind['mode'] } : bb))}
                    className="bg-slate-900/60 border border-slate-700/50 rounded px-1.5 py-1 text-slate-300"
                  >
                    <option value="ro-bind">ro-bind</option>
                    <option value="bind">bind</option>
                    <option value="tmpfs">tmpfs</option>
                  </select>
                  {b.mode !== 'tmpfs' && (
                    <input
                      value={b.src}
                      onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, src: e.target.value } : bb))}
                      placeholder="host path"
                      className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                    />
                  )}
                  <input
                    value={b.dst}
                    onChange={e => setBinds(prev => prev.map(bb => bb.id === b.id ? { ...bb, dst: e.target.value } : bb))}
                    placeholder="sandbox path"
                    className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1 text-slate-300"
                  />
                  <button onClick={() => removeBind(b.id)} className="text-slate-600 hover:text-red-400">×</button>
                </div>
              ))}
            </div>
            {automaticMounts.length > 0 && (
              <div className="mt-3 space-y-1 border-t border-slate-700/50 pt-2 text-xs font-mono">
                <span className="text-slate-500">automatic read-only mounts</span>
                {automaticMounts.map((directory) => (
                  <div key={directory} className="flex items-center gap-2 text-fuchsia-300">
                    <span className="rounded bg-fuchsia-500/10 px-1.5 py-0.5">ro-bind</span>
                    <span className="truncate">{directory} → {directory}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-4 text-xs font-mono">
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={unshareAll} onChange={e => setUnshareAll(e.target.checked)} className="accent-fuchsia-500" />
              <span>--unshare-all</span>
            </label>
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={shareNet} onChange={e => setShareNet(e.target.checked)} className="accent-fuchsia-500" />
              <span>--share-net</span>
            </label>
            <label className="flex items-center space-x-2 text-slate-300">
              <input type="checkbox" checked={dieWithParent} onChange={e => setDieWithParent(e.target.checked)} className="accent-fuchsia-500" />
              <span>--die-with-parent</span>
            </label>
          </div>

          <div className="space-y-2 text-xs font-mono">
            <div className="flex items-center space-x-2">
              <span className="text-slate-500">mode</span>
              <select aria-label="Execution mode" value={executionMode} onChange={(event) => setExecutionMode(event.target.value as 'binary' | 'command')} className="rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300">
                <option value="binary">native binary</option>
                <option value="command">Node / Python command</option>
              </select>
              <span className="text-slate-500">DISPLAY</span>
              <input value={display} onChange={e => setDisplay(e.target.value)} className="w-16 rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300" />
            </div>
            <div className="flex items-center space-x-2">
              {executionMode === 'binary' ? <>
                <span className="text-slate-500">target</span>
                <input aria-label="Target binary" value={target} onChange={e => setTarget(e.target.value)} placeholder="Select a file from Dashboard or Browse" className="flex-1 rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300" />
              </> : <>
                <span className="text-slate-500">runtime</span>
                <select aria-label="Command runtime" value={runtime} onChange={(event) => setRuntime(event.target.value)} className="rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300">
                  <option value="node">node</option><option value="python3">python3</option><option value="python">python</option><option value="custom">custom</option>
                </select>
                {runtime === 'custom' && <input aria-label="Custom runtime" value={customRuntime} onChange={(event) => setCustomRuntime(event.target.value)} placeholder="runtime path" className="w-32 rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300" />}
                <span className="text-slate-500">script</span>
                <input aria-label="Script path" value={scriptPath} onChange={(event) => setScriptPath(event.target.value)} placeholder="Select a file from Dashboard or Browse" className="flex-1 rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300" />
              </>}
              <input ref={fileInputRef} type="file" className="hidden" onChange={handleBrowserFileSelection} />
              <button type="button" onClick={selectTargetBinary} className="flex items-center gap-1 rounded border border-fuchsia-500/40 bg-fuchsia-500/10 px-2 py-1 text-fuchsia-300 hover:bg-fuchsia-500/20" title="Select target file" aria-label="Select target file">
                <FolderOpen className="h-3.5 w-3.5" /><span>Browse</span>
              </button>
            </div>
            {executionMode === 'command' && <textarea aria-label="Command arguments" value={commandArguments} onChange={(event) => setCommandArguments(event.target.value)} placeholder="Additional arguments, one per line" rows={2} className="w-full resize-y rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1 text-slate-300" />}
          </div>
          {pickerMessage && <p className="text-xs text-slate-400">{pickerMessage}</p>}
          {targetLabel && (
            <div className="text-xs text-slate-500 font-mono">
              target label: <span className="text-fuchsia-300">{targetLabel}</span>
            </div>
          )}
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 flex flex-col">
          <div className="text-xs text-slate-500 uppercase mb-2">Generated command</div>
          <pre className="text-xs font-mono text-fuchsia-300 bg-slate-900/60 rounded p-3 overflow-x-auto mb-4">{command}</pre>

          <div className="mb-2 flex items-center justify-between">
            <div className="text-xs uppercase text-slate-500">Namespace log</div>
            <button
              type="button"
              onClick={copyLog}
              disabled={!error && log.length === 0}
              className="flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-700/50 hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
              aria-label="Copy namespace log"
            >
              {copyStatus === 'copied' ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Clipboard className="h-3.5 w-3.5" />}
              <span>{copyStatus === 'copied' ? 'Copied' : copyStatus === 'failed' ? 'Copy failed' : 'Copy all'}</span>
            </button>
          </div>
          <div className="min-h-[140px] flex-1 select-text overflow-y-auto rounded bg-slate-900/60 p-2 font-mono text-xs text-slate-400">
            {error && <div className="text-red-400">{error}</div>}
            {log.length === 0 ? (
              <span className="text-slate-700">launch to provision the namespace</span>
            ) : (
              log.map((line, index) => <div key={`${index}-${line}`} className="whitespace-pre-wrap text-green-400">{line}</div>)
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Bubblewrap;
