import React, { useState, useEffect } from 'react'

interface DVEWorkspaceConfig {
  enable_overlayfs: boolean
  busybox_source: 'embedded' | 'package' | 'download'
  busybox_version: string
  fuse_overlayfs_bin: string
  enable_network_isolation: boolean
  skill_exec_timeout: string
  skill_max_memory_mb: number
  max_concurrent_wasm: number
  workspace_retention_hours: number
  max_environments: number
  max_cpu_per_env: number
  max_memory_per_env: number
}

export function DVEWorkspaceSettings() {
  const [cfg, setCfg] = useState<DVEWorkspaceConfig | null>(null)
  const [saved, setSaved] = useState(false)
  const [rootfsStatus, setRootfsStatus] = useState<'ok' | 'missing' | 'checking'>('checking')

  useEffect(() => {
    fetch('/api/dve-workspace/config')
      .then(r => r.json()).then(d => setCfg(d.config))
    fetch('/api/dve-workspace/rootfs-status')
      .then(r => r.json()).then(d => setRootfsStatus(d.ready ? 'ok' : 'missing'))
  }, [])

  const update = (key: string, value: any) => {
    if (!cfg) return
    setCfg({ ...cfg, [key]: value })
  }

  const save = async () => {
    if (!cfg) return
    await fetch('/api/dve-workspace/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(cfg)
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const bootstrapRootfs = async () => {
    await fetch('/api/dve-workspace/rootfs-bootstrap', { method: 'POST' })
    setRootfsStatus('ok')
  }

  if (!cfg) return <div className="p-4 text-gray-400">Loading DVE Workspace configuration...</div>

  return (
    <div className="bg-gray-900 border border-gray-700 rounded-lg p-6 space-y-6">
      <div className="flex items-center gap-3 border-b border-gray-700 pb-4">
        <h2 className="text-lg font-semibold text-white">DVE Workspace Configuration</h2>
        <span className="text-xs text-gray-500 ml-auto">Changes apply to new workspaces only</span>
      </div>

      {/* BusyBox Rootfs Status */}
      <div className="bg-gray-800 rounded p-3 flex items-center justify-between">
        <div>
          <span className="text-sm font-medium text-gray-300">BusyBox Rootfs</span>
          <span className="ml-2 px-2 py-0.5 text-xs rounded bg-gray-700 text-gray-400">Q1</span>
        </div>
        <div className="flex items-center gap-2">
          <span className={`text-sm ${rootfsStatus === 'ok' ? 'text-green-400' : rootfsStatus === 'missing' ? 'text-yellow-400' : 'text-gray-400'}`}>
            {rootfsStatus === 'ok' ? 'Ready' : rootfsStatus === 'missing' ? 'Not Bootstrapped' : 'Checking...'}
          </span>
          <button onClick={bootstrapRootfs} className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700">
            Bootstrap
          </button>
        </div>
      </div>

      {/* Isolation Settings */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Isolation</h3>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">enable_overlayfs</label>
            <select value={String(cfg.enable_overlayfs)} onChange={e => update('enable_overlayfs', e.target.value === 'true')}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value="true">true — kernel OverlayFS</option>
              <option value="false">false — host dir only</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">enable_network_isolation</label>
            <select value={String(cfg.enable_network_isolation)} onChange={e => update('enable_network_isolation', e.target.value === 'true')}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value="false">false — shared host network</option>
              <option value="true">true — isolated (requires slirp4netns)</option>
            </select>
          </div>
        </div>
      </div>

      {/* BusyBox Source */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">BusyBox Source</h3>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">busybox_source</label>
            <select value={cfg.busybox_source} onChange={e => update('busybox_source', e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value="embedded">embedded — bundled in binary</option>
              <option value="package">package — apt-get on host</option>
              <option value="download">download — from busybox.net</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">busybox_version</label>
            <input type="text" value={cfg.busybox_version} onChange={e => update('busybox_version', e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white" />
          </div>
        </div>
      </div>

      {/* Wazero Skill Executor */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Wazero Skill Executor</h3>
        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">skill_exec_timeout</label>
            <select value={cfg.skill_exec_timeout} onChange={e => update('skill_exec_timeout', e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value="30s">30s</option>
              <option value="60s">60s</option>
              <option value="120s">120s</option>
              <option value="300s">300s</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">skill_max_memory_mb</label>
            <select value={cfg.skill_max_memory_mb} onChange={e => update('skill_max_memory_mb', Number(e.target.value))}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value="128">128MB</option>
              <option value="256">256MB</option>
              <option value="512">512MB</option>
              <option value="1024">1024MB</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">max_concurrent_wasm</label>
            <input type="number" value={cfg.max_concurrent_wasm} onChange={e => update('max_concurrent_wasm', Number(e.target.value))}
              min={1} max={100}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white" />
          </div>
        </div>
      </div>

      {/* Workspace Lifecycle */}
      <div>
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-3">Workspace Lifecycle</h3>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">workspace_retention_hours</label>
            <select value={cfg.workspace_retention_hours} onChange={e => update('workspace_retention_hours', Number(e.target.value))}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white">
              <option value={0}>0 — immediate cleanup</option>
              <option value={6}>6 hours</option>
              <option value={24}>24 hours</option>
              <option value={48}>48 hours</option>
              <option value={168}>1 week</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">max_environments</label>
            <input type="number" value={cfg.max_environments} onChange={e => update('max_environments', Number(e.target.value))}
              min={1} max={10000}
              className="w-full bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm text-white" />
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-3 pt-4 border-t border-gray-700">
        <button onClick={save} className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 text-sm font-medium">
          {saved ? 'Saved!' : 'Save Configuration'}
        </button>
        <button onClick={async () => {
          const res = await fetch('/api/dve-workspace/stats')
          const data = await res.json()
          alert(JSON.stringify(data, null, 2))
        }} className="px-4 py-2 bg-gray-700 text-gray-300 rounded hover:bg-gray-600 text-sm">
          View Workspace Stats
        </button>
      </div>
    </div>
  )
}
