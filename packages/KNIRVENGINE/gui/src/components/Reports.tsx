import { useMemo } from 'react';
import { Clipboard, Download, FileText, Trash2 } from 'lucide-react';
import { clearToolReports, createCombinedToolReport, useToolReports } from '../services/toolReports';

const downloadReport = (contents: string) => {
  const blob = new Blob([contents], { type: 'text/markdown;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `knirvengine-tool-reports-${new Date().toISOString().replace(/[:.]/g, '-')}.md`;
  anchor.click();
  URL.revokeObjectURL(url);
};

export const Reports = () => {
  const reports = useToolReports();
  const fullReport = useMemo(() => createCombinedToolReport(reports), [reports]);
  const completed = reports.filter((report) => report.status === 'completed').length;

  const copyReport = async () => {
    try {
      await navigator.clipboard.writeText(fullReport);
    } catch {
      // Clipboard access is optional in restricted desktop/browser contexts.
    }
  };

  return <div className="min-h-full bg-slate-900 p-6 space-y-6">
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-white"><FileText className="h-6 w-6 text-cyan-300" />Tool Reports</h1>
        <p className="mt-1 text-sm text-slate-400">A persistent record of every completed or failed tool run on this device.</p>
      </div>
      <div className="flex flex-wrap gap-2">
        <button onClick={copyReport} disabled={!reports.length} className="inline-flex items-center gap-2 rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 disabled:opacity-50"><Clipboard className="h-4 w-4" />Copy full report</button>
        <button onClick={() => downloadReport(fullReport)} disabled={!reports.length} className="inline-flex items-center gap-2 rounded-lg border border-cyan-500/40 bg-cyan-500/20 px-3 py-2 text-sm text-cyan-100 disabled:opacity-50"><Download className="h-4 w-4" />Download Markdown</button>
        <button onClick={clearToolReports} disabled={!reports.length} className="inline-flex items-center gap-2 rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 text-sm text-red-200 disabled:opacity-50"><Trash2 className="h-4 w-4" />Clear</button>
      </div>
    </header>

    <div className="grid gap-3 sm:grid-cols-3">
      <div className="rounded-lg border border-slate-700 bg-slate-800/60 p-4"><p className="text-xs uppercase tracking-wide text-slate-400">Total runs</p><p className="mt-1 text-2xl font-semibold text-white">{reports.length}</p></div>
      <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4"><p className="text-xs uppercase tracking-wide text-emerald-200">Completed</p><p className="mt-1 text-2xl font-semibold text-white">{completed}</p></div>
      <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4"><p className="text-xs uppercase tracking-wide text-red-200">Failed</p><p className="mt-1 text-2xl font-semibold text-white">{reports.length - completed}</p></div>
    </div>

    {reports.length ? <div className="space-y-3">
      {reports.map((report) => <article key={report.id} className="rounded-lg border border-slate-700 bg-slate-800/60 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2"><div><h2 className="font-semibold text-white">{report.tool}</h2><p className="text-xs text-slate-400">{report.execution} · {new Date(report.completedAt).toLocaleString()}</p></div><span className={`rounded px-2 py-1 text-xs font-medium ${report.status === 'completed' ? 'bg-emerald-500/20 text-emerald-200' : 'bg-red-500/20 text-red-200'}`}>{report.status}</span></div>
        {report.error && <p className="mt-3 text-sm text-red-200">{report.error}</p>}
        <pre className="mt-3 max-h-52 overflow-auto whitespace-pre-wrap rounded bg-slate-950/70 p-3 font-mono text-xs leading-relaxed text-slate-300">{report.output || 'No output returned.'}</pre>
      </article>)}
    </div> : <div className="rounded-lg border border-dashed border-slate-600 p-10 text-center text-slate-400">Run a tool to begin building the full report.</div>}
  </div>;
};

export default Reports;
