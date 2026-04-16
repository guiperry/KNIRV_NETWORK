"use client";

import { useEffect, useState, useCallback } from 'react';
import styles from './HudOverlay.module.css';

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
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [isMinimized, setIsMinimized] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'connecting' | 'error'>('connecting');
  const [lastError, setLastError] = useState<string | null>(null);

  const fetchMetrics = useCallback(async () => {
    try {
      console.log('Fetching metrics from /api/v1/system/info');
      const response = await fetch('/api/v1/system/info');
      console.log('Response status:', response.status);
      if (!response.ok) {
        setConnectionStatus('error');
        setLastError(`HTTP ${response.status}`);
        console.error('Response not ok:', response.status);
        return;
      }
      const data = await response.json();
      console.log('Received data:', data);
      setMetrics(data);
      setConnectionStatus('connected');
      setLastError(null);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      setConnectionStatus('error');
      setLastError(error instanceof Error ? error.message : 'Unknown error');
    }
  }, []);

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchMetrics, refreshInterval]);

  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) {
      return `${days}d ${hours}h ${minutes}m`;
    }
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  const formatMemory = (mb: number): string => {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)}GB`;
    }
    return `${mb}MB`;
  };

  if (isMinimized) {
    return (
      <div className={`${styles.minimized} ${className}`}>
        <button 
          className={styles.minimizedButton}
          onClick={() => setIsMinimized(false)} 
          title="Restore HUD"
          aria-label="Restore system monitor"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 1L3 6h3v4h4V6h3L8 1z"/>
          </svg>
        </button>
      </div>
    );
  }

  return (
    <div className={`${styles.hudContainer} ${className}`}>
      <div className={styles.hudHeader}>
        <span className={styles.title}>KNIRV System Monitor</span>
        <div className={styles.controls}>
          <span className={`${styles.status} ${styles[connectionStatus]}`}>
            {connectionStatus === 'connected' && '●'}
            {connectionStatus === 'connecting' && '◐'}
            {connectionStatus === 'error' && '○'}
          </span>
          <button 
            className={styles.button}
            onClick={() => setIsMinimized(true)} 
            title="Minimize"
            aria-label="Minimize system monitor"
          >
            <svg width="10" height="10" viewBox="0 0 10 10" fill="currentColor">
              <rect x="0" y="4" width="10" height="2"/>
            </svg>
          </button>
        </div>
      </div>

      <div className={styles.metrics}>
        {metrics ? (
          <>
            <div className={styles.metric}>
              <label className={styles.label}>CPU</label>
              <div className={styles.bar}>
                <div
                  className={styles.fill}
                  style={{ width: `${Math.min(metrics.cpu, 100)}%` }}
                />
              </div>
              <span className={styles.value}>{metrics.cpu.toFixed(1)}%</span>
            </div>

            <div className={styles.metric}>
              <label className={styles.label}>Memory</label>
              <div className={styles.bar}>
                <div
                  className={styles.fill}
                  style={{
                    width: `${Math.min(metrics.memory.percentage, 100)}%`,
                  }}
                />
              </div>
              <span className={styles.value}>
                {formatMemory(metrics.memory.used_mb)} / {formatMemory(metrics.memory.total_mb)}
              </span>
            </div>

            <div className={styles.info}>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Uptime</span>
                <span className={styles.infoValue}>{formatUptime(metrics.uptime_seconds)}</span>
              </div>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>System</span>
                <span className={styles.infoValue}>{metrics.os} {metrics.arch}</span>
              </div>
              <div className={styles.infoRow}>
                <span className={styles.infoLabel}>Host</span>
                <span className={styles.infoValue}>{metrics.hostname}</span>
              </div>
            </div>
          </>
        ) : (
          <div className={styles.loading}>
            <div className={styles.spinner} />
            <span>Loading metrics...</span>
            {lastError && <span className={styles.error}>{lastError}</span>}
          </div>
        )}
      </div>
    </div>
  );
}