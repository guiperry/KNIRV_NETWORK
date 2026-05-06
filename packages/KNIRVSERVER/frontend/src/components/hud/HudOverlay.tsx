"use client";

import { useEffect, useState, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import styles from './HudOverlay.module.css';
import { useTelemetry } from '@/hooks/use-telemetry';
import { useOntology } from '@/hooks/use-ontology';

export interface SystemMetrics {
  cpu: number;
  memory: {
    total_mb: number;
    used_mb: number;
    available_mb: number;
    percentage: number;
  };
  uptime_seconds: number;
  os: string;
  arch: string;
  hostname: string;
}

interface HudOverlayProps {
  className?: string;
  refreshInterval?: number;
}

export function HudOverlay({ className = '', refreshInterval = 2000 }: HudOverlayProps) {
  const router = useRouter();
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'ONLINE' | 'CONNECTING' | 'ERROR'>('CONNECTING');
  const [currentTime, setCurrentTime] = useState('');
  const [isMinimized, setIsMinimized] = useState(false);
  const [cpuHistory, setCpuHistory] = useState<number[]>(new Array(30).fill(0));
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const { telemetry } = useTelemetry();
  const { stats: ontologyStats } = useOntology();

  const fetchMetrics = useCallback(async () => {
    try {
      const response = await fetch('/api/v1/system/info');
      if (!response.ok) { setConnectionStatus('ERROR'); return; }
      const data: SystemMetrics = await response.json();
      setMetrics(data);
      setConnectionStatus('ONLINE');
      setCpuHistory(prev => [...prev.slice(1), data.cpu]);
    } catch {
      setConnectionStatus('ERROR');
    }
  }, []);

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchMetrics, refreshInterval]);

  useEffect(() => {
    const tick = () => setCurrentTime(new Date().toTimeString().slice(0, 8));
    tick();
    const interval = setInterval(tick, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = 'rgba(10, 30, 100, 0.3)';
    ctx.fillRect(0, 0, w, h);
    ctx.strokeStyle = 'rgba(72, 136, 255, 0.1)';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
      ctx.beginPath(); ctx.moveTo(0, (h / 4) * i); ctx.lineTo(w, (h / 4) * i); ctx.stroke();
    }
    if (cpuHistory.some(v => v > 0)) {
      ctx.strokeStyle = '#4888ff'; ctx.lineWidth = 1.5;
      ctx.shadowColor = 'rgba(72, 136, 255, 0.6)'; ctx.shadowBlur = 4;
      ctx.beginPath();
      cpuHistory.forEach((val, i) => {
        const x = (i / (cpuHistory.length - 1)) * w;
        const y = h - (val / 100) * h;
        if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
      });
      ctx.stroke();
    }
  }, [cpuHistory]);

  const formatUptime = (s: number) => {
    const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60);
    return `${h}h ${m}m`;
  };
  const formatMemory = (mb: number) =>
    mb >= 1024 ? `${(mb / 1024).toFixed(1)}GB` : `${mb}MB`;

  const cpuPct = metrics?.cpu ?? 0;
  const memPct = metrics?.memory?.percentage ?? 0;
  const statusClass =
    connectionStatus === 'ONLINE' ? styles.active :
    connectionStatus === 'ERROR'  ? styles.errorVal : styles.connecting;

  if (isMinimized) {
    return (
      <button className={`${styles.minimizedBtn} ${className}`} onClick={() => setIsMinimized(false)} title="Restore HUD">
        ▲ HUD
      </button>
    );
  }

  return (
    <>
      {/* Corner decorations */}
      <div className={`${styles.corner} ${styles.cornerTL}`} />
      <div className={`${styles.corner} ${styles.cornerTR}`} />
      <div className={`${styles.corner} ${styles.cornerBL}`} />
      <div className={`${styles.corner} ${styles.cornerBR}`} />

      {/* ── Top Panel ─────────────────────────────────────────────────── */}
      <div className={`${styles.panel} ${styles.topPanel} ${className}`}>
        {/* brand — hidden on very small screens */}
        <div className={`${styles.panelSection} ${styles.brandSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>KNIRV NEXUS</span>
          <span className={`${styles.value} ${styles.active}`}>DASHBOARD HUD</span>
        </div>
        {/* mobile brand — always visible */}
        <div className={`${styles.panelSection} ${styles.mobileBrand}`}>
          <span className={`${styles.value} ${styles.active}`}>KNIRV</span>
        </div>

        <div className={styles.panelSection}>
          <span className={styles.label}>STATUS</span>
          <span className={`${styles.value} ${statusClass}`}>{connectionStatus}</span>
        </div>
        <div className={`${styles.panelSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>CPU</span>
          <span className={styles.value}>{cpuPct.toFixed(1)}%</span>
        </div>
        <div className={`${styles.panelSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>MEMORY</span>
          <span className={styles.value}>{memPct.toFixed(1)}%</span>
        </div>
        <div className={styles.panelSection}>
          <span className={styles.label}>TIME</span>
          <span className={styles.value}>{currentTime || '—'}</span>
        </div>
        <div className={`${styles.panelSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>UPTIME</span>
          <span className={styles.value}>{metrics ? formatUptime(metrics.uptime_seconds) : '—'}</span>
        </div>
        <div className={`${styles.panelSection} ${styles.windowControls}`}>
          <button className={`${styles.controlBtn} ${styles.menuBackBtn} ${styles.desktopOnly}`} onClick={() => router.push('/menu')}>
            &#9673; MENU
          </button>
          <button className={styles.controlBtn} onClick={() => setIsMinimized(true)} title="Minimize">
            &#8722;
          </button>
        </div>
      </div>

      {/* ── Left Panel (desktop only) ──────────────────────────────────── */}
      <div className={`${styles.panel} ${styles.leftPanel}`}>
        <div className={styles.panelHeader}>SYSTEM METRICS</div>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>CPU</span>
          <div className={styles.metricBar}><div className={styles.metricFill} style={{ width: `${Math.min(cpuPct, 100)}%` }} /></div>
          <span className={styles.metricValue}>{cpuPct.toFixed(1)}%</span>
        </div>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>MEMORY</span>
          <div className={styles.metricBar}><div className={styles.metricFill} style={{ width: `${Math.min(memPct, 100)}%` }} /></div>
          <span className={styles.metricValue}>{memPct.toFixed(1)}%</span>
        </div>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>PROCESSES</span>
          <div className={styles.metricBar}><div className={styles.metricFill} style={{ width: '30%' }} /></div>
          <span className={styles.metricValue}>—</span>
        </div>
        <div className={styles.panelHeader} style={{ marginTop: '20px' }}>QUICK STATS</div>
        <div className={styles.statItem}><span>MEM USED: {metrics ? formatMemory(metrics.memory.used_mb) : '—'}</span></div>
        <div className={styles.statItem}><span>MEM TOTAL: {metrics ? formatMemory(metrics.memory.total_mb) : '—'}</span></div>
        <div className={styles.statItem}><span>HOST: {metrics?.hostname ?? '—'}</span></div>
        {telemetry.data && (
          <>
            <div className={styles.panelHeader} style={{ marginTop: '16px' }}>TELEMETRY</div>
            <div className={styles.statItem}><span>CPU PRESSURE: {telemetry.data.cpu_pressure.toFixed(1)}%</span></div>
            <div className={styles.statItem}><span>MEM PRESSURE: {telemetry.data.memory_pressure.toFixed(1)}%</span></div>
            <div className={styles.statItem}><span>CONNECTIONS: {telemetry.data.active_connections}</span></div>
            <div className={styles.statItem}><span>GOROUTINES: {telemetry.data.goroutines}</span></div>
          </>
        )}
      </div>

      {/* ── Right Panel (desktop only) ─────────────────────────────────── */}
      <div className={`${styles.panel} ${styles.rightPanel}`}>
        <div className={styles.panelHeader}>FRONTEND STATUS</div>
        <div className={styles.infoItem}>
          <span className={styles.infoLabel}>STATUS:</span>
          <span className={`${styles.value} ${statusClass}`}>{connectionStatus}</span>
        </div>
        <div className={styles.panelHeader} style={{ marginTop: '15px' }}>PERFORMANCE</div>
        <canvas ref={canvasRef} className={styles.chart} width={200} height={80} />
        <div className={styles.panelHeader} style={{ marginTop: '20px' }}>NOTIFICATIONS</div>
        <div className={styles.notificationItem}>
          <div className={`${styles.notifDot} ${connectionStatus === 'ONLINE' ? styles.notifActive : ''}`} />
          <span>{connectionStatus === 'ONLINE' ? 'System operational' : 'Connecting...'}</span>
        </div>
        <div className={styles.notificationItem}>
          <div className={styles.notifDot} /><span>No errors detected</span>
        </div>
        <div className={styles.notificationItem}>
          <div className={styles.notifDot} /><span>All services running</span>
        </div>
        <div className={styles.panelHeader} style={{ marginTop: '20px' }}>SYSTEM INFO</div>
        <div className={styles.infoItem}>
          <span className={styles.infoLabel}>OS:</span>
          <span className={styles.infoValue}>{metrics?.os ?? '—'}</span>
        </div>
        <div className={styles.infoItem}>
          <span className={styles.infoLabel}>ARCH:</span>
          <span className={styles.infoValue}>{metrics?.arch ?? '—'}</span>
        </div>
        {ontologyStats.data && (
          <>
            <div className={styles.panelHeader} style={{ marginTop: '16px' }}>KNOWLEDGE GRAPH</div>
            <div className={styles.statItem}><span>ENTITIES: {ontologyStats.data.entity_count}</span></div>
            <div className={styles.statItem}><span>RELATIONS: {ontologyStats.data.relation_count}</span></div>
          </>
        )}
      </div>

      {/* ── Mobile collapsed metrics strip (hidden on desktop) ────────── */}
      <div className={`${styles.panel} ${styles.mobileMetrics}`}>
        <div className={styles.mobileMetricItem}>
          <span className={styles.label}>CPU</span>
          <div className={styles.metricBar}><div className={styles.metricFill} style={{ width: `${Math.min(cpuPct, 100)}%` }} /></div>
          <span className={styles.mobileMetricVal}>{cpuPct.toFixed(0)}%</span>
        </div>
        <div className={styles.mobileMetricItem}>
          <span className={styles.label}>MEM</span>
          <div className={styles.metricBar}><div className={styles.metricFill} style={{ width: `${Math.min(memPct, 100)}%` }} /></div>
          <span className={styles.mobileMetricVal}>{memPct.toFixed(0)}%</span>
        </div>
        <div className={styles.mobileStat}>
          <span className={styles.label}>UPTIME</span>
          <span className={styles.mobileMetricVal}>{metrics ? formatUptime(metrics.uptime_seconds) : '—'}</span>
        </div>
        <div className={styles.mobileStat}>
          <span className={styles.label}>OS</span>
          <span className={styles.mobileMetricVal}>{metrics?.os ?? '—'} {metrics?.arch ?? ''}</span>
        </div>
        <div className={styles.mobileStat}>
          <span className={styles.label}>HOST</span>
          <span className={styles.mobileMetricVal}>{metrics?.hostname ?? '—'}</span>
        </div>
      </div>

      {/* ── Bottom Panel ──────────────────────────────────────────────── */}
      <div className={`${styles.panel} ${styles.bottomPanel}`}>
        <div className={styles.panelSection}>
          <span className={styles.label}>NET RX</span>
          <span className={styles.value}>—</span>
        </div>
        <div className={styles.panelSection}>
          <span className={styles.label}>NET TX</span>
          <span className={styles.value}>—</span>
        </div>
        <div className={`${styles.panelSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>DISK READ</span>
          <span className={styles.value}>—</span>
        </div>
        <div className={`${styles.panelSection} ${styles.desktopOnly}`}>
          <span className={styles.label}>DISK WRITE</span>
          <span className={styles.value}>—</span>
        </div>
        <div className={styles.panelSection}>
          <span className={styles.label}>HOST</span>
          <span className={styles.value}>{metrics?.hostname ?? '—'}</span>
        </div>
      </div>
    </>
  );
}
