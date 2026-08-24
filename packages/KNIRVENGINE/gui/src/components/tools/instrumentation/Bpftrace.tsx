import React, { useState } from 'react';
import { Activity, Play, Square } from 'lucide-react';

const presets = [
  { label: 'syscall counts by process', script: 'tracepoint:raw_syscalls:sys_enter\n{ @[comm] = count(); }' },
  { label: 'trace target execve', script: 'tracepoint:syscalls:sys_enter_execve\n{ printf("%s -> %s\\n", comm, str(args->filename)); }' },
  { label: 'uprobe on libssl SSL_write', script: 'uprobe:/usr/lib/x86_64-linux-gnu/libssl.so.3:SSL_write\n{ printf("pid %d wrote %d bytes\\n", pid, arg2); }' },
  { label: 'TCP connect latency', script: 'kprobe:tcp_v4_connect\n{ @start[tid] = nsecs; }\nkretprobe:tcp_v4_connect\n/@start[tid]/\n{ @us = hist((nsecs - @start[tid]) / 1000); delete(@start[tid]); }' },
];

const eventRows = [
  { time: '00:00:01.203', comm: 'target_binary', pid: 4821, probe: 'sys_enter_connect', detail: 'fd=7 dst=10.0.0.4:443' },
  { time: '00:00:01.207', comm: 'target_binary', pid: 4821, probe: 'sys_enter_write', detail: 'fd=7 len=512' },
  { time: '00:00:02.891', comm: 'agentd', pid: 1187, probe: 'sys_enter_execve', detail: '/usr/bin/curl -s https://api.targetapp.local' },
  { time: '00:00:03.014', comm: 'target_binary', pid: 4821, probe: 'sys_enter_openat', detail: '/proc/self/maps' },
];

const Bpftrace: React.FC = () => {
  const [script, setScript] = useState(presets[3].script);
  const [running, setRunning] = useState(true);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-purple-500/20 rounded-lg">
          <Activity className="w-6 h-6 text-purple-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">bpftrace</h1>
          <p className="text-slate-400 text-sm font-mono">eBPF · kernel 6.5 · BTF: enabled</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <h3 className="text-xs uppercase text-slate-500 mb-3">Presets</h3>
          <div className="space-y-1">
            {presets.map(p => (
              <button
                key={p.label}
                onClick={() => setScript(p.script)}
                className="w-full text-left px-2 py-2 rounded text-xs font-mono text-slate-400 hover:bg-slate-700/30 hover:text-white transition-colors"
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>

        <div className="xl:col-span-2 space-y-4">
          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 border-b border-slate-700/50">
              <span className="text-xs text-slate-500 font-mono">trace.bt</span>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setRunning(true)}
                  className="flex items-center space-x-1 text-xs px-2 py-1 rounded bg-purple-500/20 text-purple-300 hover:bg-purple-500/30"
                >
                  <Play className="w-3.5 h-3.5" />
                  <span>bpftrace trace.bt</span>
                </button>
                <button
                  onClick={() => setRunning(false)}
                  className="flex items-center space-x-1 text-xs px-2 py-1 rounded bg-slate-700/50 text-slate-300 hover:bg-slate-700"
                >
                  <Square className="w-3.5 h-3.5" />
                  <span>^C</span>
                </button>
              </div>
            </div>
            <textarea
              value={script}
              onChange={e => setScript(e.target.value)}
              spellCheck={false}
              className="w-full h-28 bg-slate-900/60 text-slate-200 font-mono text-xs p-3 focus:outline-none resize-none"
            />
          </div>

          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-slate-500">@us[tcp_v4_connect latency, us]</span>
              <span className={`text-xs flex items-center space-x-1 ${running ? 'text-green-400' : 'text-slate-600'}`}>
                <span className="w-1.5 h-1.5 rounded-full bg-current"></span>
                <span>{running ? 'attached' : 'stopped'}</span>
              </span>
            </div>
            <pre className="text-xs font-mono text-purple-300 leading-tight">
{`[0, 2)        1 |                                        |
[2, 4)        3 |@                                       |
[4, 8)       42 |@@@@@@@@@@@@@@@@@@@@@@                  |
[8, 16)      78 |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@|
[16, 32)     29 |@@@@@@@@@@@@@@                          |
[32, 64)      6 |@@@                                     |`}
            </pre>
          </div>

          <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
            <table className="w-full text-xs font-mono">
              <thead>
                <tr className="border-b border-slate-700/50 text-slate-500 uppercase">
                  <th className="text-left px-3 py-2">Time</th>
                  <th className="text-left px-3 py-2">Comm</th>
                  <th className="text-right px-3 py-2">Pid</th>
                  <th className="text-left px-3 py-2">Probe</th>
                  <th className="text-left px-3 py-2">Detail</th>
                </tr>
              </thead>
              <tbody>
                {eventRows.map((r, i) => (
                  <tr key={i} className="border-b border-slate-800">
                    <td className="px-3 py-1.5 text-slate-600">{r.time}</td>
                    <td className="px-3 py-1.5 text-slate-300">{r.comm}</td>
                    <td className="px-3 py-1.5 text-right text-slate-500">{r.pid}</td>
                    <td className="px-3 py-1.5 text-purple-300">{r.probe}</td>
                    <td className="px-3 py-1.5 text-slate-400">{r.detail}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Bpftrace;
