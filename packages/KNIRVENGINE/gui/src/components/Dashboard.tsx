import React, { ChangeEvent, useRef, useState } from 'react';
import { ChevronDown, ChevronRight, File as FileIcon, FileCode2, Folder, FolderOpen, Plus, RefreshCw, Save, X } from 'lucide-react';
import { useSandbox, type SandboxProjectFile } from './SandboxContext';

type ProjectFile = SandboxProjectFile;
type FileTreeNode = { name: string; path: string; file?: File; handle?: FileHandle; nativePath?: string; nativeSize?: number; children: FileTreeNode[] };
type FileHandle = { kind: 'file'; name: string; getFile(): Promise<File>; createWritable?: () => Promise<FileWriter> };
type FileWriter = { write(contents: string): Promise<void>; close(): Promise<void> };
type DirectoryHandle = { name: string; values(): AsyncIterableIterator<DirectoryHandle | FileHandle> };
type DirectoryPickerWindow = Window & {
  showDirectoryPicker?: () => Promise<DirectoryHandle>;
  electronAPI?: {
    getPathForFile?: (file: File) => string;
    selectSandboxProject?: () => Promise<string | null>;
    listSandboxProjectFiles?: () => Promise<Array<{ name: string; path: string; size: number }>>;
    readSandboxProjectFile?: (relativePath: string) => Promise<string>;
  };
};

const TEXT_FILE_LIMIT = 1_000_000;
const IGNORED_DIRECTORIES = new Set(['.git', '.next', 'build', 'coverage', 'dist', 'node_modules']);
const isPreviewable = (file: File) => file.type.startsWith('image/') || file.type === 'application/pdf';
const readText = (file: File) => {
  if (typeof file.text === 'function') return file.text();
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ''));
    reader.onerror = () => reject(reader.error);
    reader.readAsText(file);
  });
};

const createTree = (files: ProjectFile[]): FileTreeNode[] => {
  const root: FileTreeNode = { name: '', path: '', children: [] };
  files.forEach(({ path, file, handle, nativePath, nativeSize }) => {
    const parts = path.split('/').filter(Boolean);
    let node = root;
    parts.forEach((part, index) => {
      const nextPath = parts.slice(0, index + 1).join('/');
      let child = node.children.find((candidate) => candidate.name === part);
      if (!child) {
        child = { name: part, path: nextPath, children: [] };
        node.children.push(child);
      }
      if (index === parts.length - 1) {
        child.file = file;
        child.handle = handle;
        child.nativePath = nativePath;
        child.nativeSize = nativeSize;
      }
      node = child;
    });
  });
  const sortTree = (nodes: FileTreeNode[]) => {
    nodes.sort((a, b) => Boolean(a.file) === Boolean(b.file) ? a.name.localeCompare(b.name) : a.file ? 1 : -1);
    nodes.forEach((node) => sortTree(node.children));
  };
  sortTree(root.children);
  return root.children;
};

const collectDirectoryFiles = async (directory: DirectoryHandle, prefix = ''): Promise<ProjectFile[]> => {
  const files: ProjectFile[] = [];
  for await (const entry of directory.values()) {
    const path = prefix ? `${prefix}/${entry.name}` : entry.name;
    if ('kind' in entry && entry.kind === 'file') files.push({ name: entry.name, path, file: await entry.getFile(), handle: entry });
    else if (!IGNORED_DIRECTORIES.has(entry.name)) files.push(...await collectDirectoryFiles(entry as DirectoryHandle, path));
  }
  return files;
};

const FileTree = ({ nodes, depth = 0, onSelect, selectedPath }: { nodes: FileTreeNode[]; depth?: number; onSelect: (node: FileTreeNode) => void; selectedPath?: string }) => (
  <ul className="space-y-0.5" role={depth === 0 ? 'tree' : 'group'}>
    {nodes.map((node) => <TreeItem key={node.path} node={node} depth={depth} onSelect={onSelect} selectedPath={selectedPath} />)}
  </ul>
);

const TreeItem = ({ node, depth, onSelect, selectedPath }: { node: FileTreeNode; depth: number; onSelect: (node: FileTreeNode) => void; selectedPath?: string }) => {
  const [expanded, setExpanded] = useState(depth < 1);
  if (!node.file) return <li>
    <button type="button" className="flex w-full items-center gap-1.5 rounded px-2 py-1.5 text-left text-sm text-slate-300 hover:bg-slate-800 hover:text-white" style={{ paddingLeft: `${depth * 12 + 8}px` }} onClick={() => setExpanded((current) => !current)} aria-expanded={expanded}>
      {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
      {expanded ? <FolderOpen className="h-4 w-4 text-amber-400" /> : <Folder className="h-4 w-4 text-amber-400" />}
      <span className="truncate">{node.name}</span>
    </button>
    {expanded ? <FileTree nodes={node.children} depth={depth + 1} onSelect={onSelect} selectedPath={selectedPath} /> : null}
  </li>;
  const selected = selectedPath === node.path;
  return <li><button type="button" className={`flex w-full items-center gap-2 rounded py-1.5 pr-2 text-left text-sm transition-colors ${selected ? 'bg-violet-500/20 text-violet-200' : 'text-slate-400 hover:bg-slate-800 hover:text-white'}`} style={{ paddingLeft: `${depth * 12 + 28}px` }} onClick={() => onSelect(node)} aria-current={selected ? 'page' : undefined}>
    <FileCode2 className="h-3.5 w-3.5 shrink-0 text-slate-500" /><span className="truncate">{node.name}</span>
  </button></li>;
};

export const Dashboard = () => {
  const { targetLabel, projectPath, projectFiles, setTargetLabel, setProjectPath, setProjectFiles, setProjectTargetPath } = useSandbox();
  const inputRef = useRef<HTMLInputElement>(null);
  const [targetName, setTargetName] = useState(targetLabel);
  const [tree, setTree] = useState<FileTreeNode[]>(() => createTree(projectFiles));
  const [selected, setSelected] = useState<FileTreeNode>();
  const [content, setContent] = useState('');
  const [draft, setDraft] = useState('');
  const [savedContent, setSavedContent] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');
  const [loadingTarget, setLoadingTarget] = useState(false);
  const [loadingFile, setLoadingFile] = useState(false);
  const [loadError, setLoadError] = useState('');

  const loadTarget = (name: string, files: ProjectFile[], projectPath = '') => {
    // Electron resolves an absolute path only for the files the user selected;
    // browser-only launches retain the label without host filesystem access.
    setTargetLabel(name);
    setProjectPath(projectPath);
    setProjectFiles(files);
    setProjectTargetPath('');
    setTargetName(name); setTree(createTree(files)); setSelected(undefined); setContent(''); setDraft(''); setSavedContent(''); setSaveMessage('');
    setLoadError(files.length || projectPath ? '' : 'This folder does not contain any files.');
  };
  const chooseTarget = async () => {
    const picker = window as DirectoryPickerWindow;
    if (picker.electronAPI?.selectSandboxProject) {
      const projectPath = await picker.electronAPI.selectSandboxProject();
      if (!projectPath) return;
      const name = projectPath.split('/').filter(Boolean).pop() || 'Selected project';
      setLoadingTarget(true);
      try {
        const nativeFiles = await picker.electronAPI.listSandboxProjectFiles?.() ?? [];
        const files = nativeFiles.map((nativeFile) => ({
          name: nativeFile.name,
          path: nativeFile.path,
          file: new globalThis.File([], nativeFile.name),
          nativePath: nativeFile.path,
          nativeSize: nativeFile.size,
        }));
        loadTarget(name, files, projectPath);
      } catch {
        setLoadError('Unable to read that project folder. Please try again.');
      } finally {
        setLoadingTarget(false);
      }
      return;
    }
    if (!picker.showDirectoryPicker) { inputRef.current?.click(); return; }
    try {
      const directory = await picker.showDirectoryPicker();
      // Reveal the workspace immediately. Large projects may take a moment to scan.
      setTargetName(directory.name); setTree([]); setSelected(undefined); setContent(''); setDraft(''); setSavedContent(''); setSaveMessage(''); setLoadError(''); setLoadingTarget(true);
      const files = await collectDirectoryFiles(directory);
      const firstPath = files[0]?.file && picker.electronAPI?.getPathForFile?.(files[0].file);
      const firstRelativePath = files[0]?.path;
      const projectPath = firstPath && firstRelativePath ? firstPath.slice(0, Math.max(0, firstPath.length - firstRelativePath.length)).replace(/\/$/, '') : '';
      loadTarget(directory.name, files, projectPath);
    } catch (error) {
      if ((error as DOMException).name !== 'AbortError') setLoadError('Unable to read that folder. Please try again.');
    } finally { setLoadingTarget(false); }
  };
  const handleInput = (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    const rootPath = files[0]?.webkitRelativePath || '';
    const name = rootPath.split('/')[0] || 'Selected project';
    setLoadingTarget(true);
    const projectFiles = files.map((file) => {
      const relativePath = file.webkitRelativePath || file.name;
      return { name: file.name, path: relativePath.split('/').slice(1).join('/') || file.name, file };
    });
    const picker = window as DirectoryPickerWindow;
    const firstPath = projectFiles[0]?.file && picker.electronAPI?.getPathForFile?.(projectFiles[0].file);
    const firstRelativePath = projectFiles[0]?.path;
    const projectPath = firstPath && firstRelativePath ? firstPath.slice(0, Math.max(0, firstPath.length - firstRelativePath.length)).replace(/\/$/, '') : '';
    loadTarget(name, projectFiles, projectPath);
    setLoadingTarget(false);
    event.target.value = '';
  };
  const selectFile = async (node: FileTreeNode) => {
    if (!node.file) return;
    if (node.nativePath && projectPath) {
      setProjectTargetPath(`${projectPath.replace(/[\\/]$/, '')}/${node.nativePath.replace(/\\/g, '/')}`);
    }
    setSelected(node); setLoadingFile(true); setLoadError(''); setSaveMessage(''); setDraft(''); setSavedContent('');
    try {
      const fileSize = node.nativeSize ?? node.file.size;
      if (fileSize > TEXT_FILE_LIMIT && !isPreviewable(node.file)) setContent('This file is too large to display in the page explorer.');
      else if (node.nativePath && (window as DirectoryPickerWindow).electronAPI?.readSandboxProjectFile) {
        const text = await (window as DirectoryPickerWindow).electronAPI.readSandboxProjectFile(node.nativePath);
        setContent(text); setDraft(text); setSavedContent(text);
      }
      else if (isPreviewable(node.file)) setContent(URL.createObjectURL(node.file));
      else {
        const text = await readText(node.file);
        setContent(text); setDraft(text); setSavedContent(text);
      }
    } catch { setLoadError('This file could not be read.'); }
    finally { setLoadingFile(false); }
  };
  const saveFile = async () => {
    if (!selected?.file || isPreviewable(selected.file)) return;
    setSaving(true); setSaveMessage('');
    try {
      if (selected.handle?.createWritable) {
        const writer = await selected.handle.createWritable();
        await writer.write(draft);
        await writer.close();
        selected.file = new File([draft], selected.name, { type: selected.file.type, lastModified: Date.now() });
        setSavedContent(draft); setSaveMessage('Saved to project.');
      } else {
        const url = URL.createObjectURL(new Blob([draft], { type: selected.file.type || 'text/plain' }));
        const link = document.createElement('a');
        link.href = url; link.download = selected.name; link.click(); URL.revokeObjectURL(url);
        setSavedContent(draft); setSaveMessage('Downloaded changed copy.');
      }
    } catch { setSaveMessage('Unable to save this file.'); }
    finally { setSaving(false); }
  };
  const clearTarget = () => { setTargetLabel(''); setProjectPath(''); setProjectFiles([]); setProjectTargetPath(''); setTargetName(''); setTree([]); setSelected(undefined); setContent(''); setDraft(''); setSavedContent(''); setSaveMessage(''); setLoadError(''); setLoadingTarget(false); };

  if (!targetName) return <main className="flex min-h-[calc(100vh-4rem)] items-center justify-center p-6">
    <input ref={inputRef} className="hidden" type="file" multiple webkitdirectory="" onChange={handleInput} />
    <section className="w-full max-w-md text-center"><div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl border border-violet-400/30 bg-violet-500/10"><FolderOpen className="h-8 w-8 text-violet-300" /></div><h1 className="text-2xl font-semibold text-white">Open a target project</h1><p className="mt-3 text-sm leading-6 text-slate-400">Choose a local folder to inspect its files and explore individual pages.</p>{loadError ? <p className="mt-4 text-sm text-red-300">{loadError}</p> : null}<button type="button" onClick={chooseTarget} className="mx-auto mt-7 inline-flex items-center gap-2 rounded-lg bg-violet-500 px-5 py-3 font-medium text-white transition hover:bg-violet-400 focus:outline-none focus:ring-2 focus:ring-violet-300 focus:ring-offset-2 focus:ring-offset-slate-950"><Plus className="h-5 w-5" />Add target</button></section>
  </main>;

  const preview = selected?.file && isPreviewable(selected.file);
  const isDirty = !preview && draft !== savedContent;
  return <main className="flex h-[calc(100vh-4rem)] min-h-[34rem] flex-col p-4 md:p-6">
    <input ref={inputRef} className="hidden" type="file" multiple webkitdirectory="" onChange={handleInput} />
    <header className="mb-4 flex shrink-0 items-center justify-between gap-4"><div className="min-w-0"><p className="text-xs font-medium uppercase tracking-[0.18em] text-violet-300">Target project</p><h1 className="truncate text-xl font-semibold text-white">{targetName}</h1></div><div className="flex shrink-0 gap-2"><button type="button" onClick={chooseTarget} className="inline-flex items-center gap-2 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white"><RefreshCw className="h-4 w-4" />Change target</button><button type="button" onClick={clearTarget} className="inline-flex items-center gap-2 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-red-400/60 hover:text-red-200"><X className="h-4 w-4" />Close</button></div></header>
  <div className="grid min-h-0 flex-1 grid-cols-1 overflow-hidden rounded-xl border border-slate-800 bg-slate-950 shadow-2xl md:grid-cols-[minmax(12rem,22%)_1fr]"><aside className="min-h-0 overflow-y-auto border-b border-slate-800 bg-slate-900/60 p-2 md:border-b-0 md:border-r" aria-label="Project files"><div className="flex items-center gap-2 px-2 py-2 text-xs font-medium uppercase tracking-wider text-slate-500"><Folder className="h-3.5 w-3.5" />Files</div>{loadingTarget ? <p className="px-2 py-3 text-sm text-slate-500">Loading project files…</p> : <FileTree nodes={tree} onSelect={selectFile} selectedPath={selected?.path} />}</aside><section className="min-h-0 overflow-hidden" aria-label="Page explorer">{selected ? <div className="flex h-full min-h-0 flex-col"><div className="flex shrink-0 items-center gap-2 border-b border-slate-800 bg-slate-900/40 px-4 py-3"><FileIcon className="h-4 w-4 text-violet-300" /><span className="truncate font-mono text-sm text-slate-200">{selected.path}</span><span className="ml-auto text-xs text-slate-500">{(selected.nativeSize ?? selected.file?.size ?? 0).toLocaleString()} bytes</span>{!preview ? <button type="button" onClick={saveFile} disabled={!isDirty || saving} className="ml-2 inline-flex items-center gap-1.5 rounded bg-violet-500 px-2.5 py-1.5 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-50"><Save className="h-3.5 w-3.5" />{saving ? 'Saving…' : selected.handle?.createWritable ? 'Save' : 'Download'}</button> : null}</div>{saveMessage ? <p className="shrink-0 border-b border-slate-800 px-4 py-2 text-xs text-emerald-300">{saveMessage}</p> : null}{loadingFile ? <div className="p-5 text-sm text-slate-400">Loading page…</div> : loadError ? <div className="p-5 text-sm text-red-300">{loadError}</div> : preview ? (selected.file?.type.startsWith('image/') ? <div className="flex h-full items-center justify-center overflow-auto p-6"><img src={content} alt={selected.name} className="max-h-full max-w-full object-contain" /></div> : <iframe title={selected.name} src={content} className="h-full w-full bg-white" />) : <textarea aria-label="Page editor" value={draft} onChange={(event) => { setDraft(event.target.value); setSaveMessage(''); }} spellCheck={false} className="h-full w-full resize-none bg-slate-950 p-5 font-mono text-sm leading-6 text-slate-200 outline-none focus:bg-slate-900/30" />}</div> : <div className="flex h-full flex-col items-center justify-center p-8 text-center"><FileCode2 className="h-8 w-8 text-slate-600" /><h2 className="mt-4 text-base font-medium text-slate-300">{loadingTarget ? 'Opening target project…' : 'Select a page to explore'}</h2><p className="mt-2 max-w-sm text-sm text-slate-500">{loadingTarget ? 'The explorers are ready while we scan the project files.' : 'Choose any file from the project explorer to edit it here.'}</p></div>}</section></div>
  </main>;
};
