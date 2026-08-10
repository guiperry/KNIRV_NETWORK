'use client';

import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useKnirvbaseMetrics, type KnirvbaseMetric } from '@/hooks/use-knirvbase-metrics';
import { useKnirvchainMetrics, type KnirvchainMetric } from '@/hooks/use-knirvchain-metrics';
import { useKnirvbaseHealth } from '@/hooks/use-knirvbase-health';
import { useKnirvchainHealth } from '@/hooks/use-knirvchain-health';
import { Activity, Cpu, HardDrive, Network, Server, GitBranch, DollarSign } from 'lucide-react';
import { useKnirvgraphScalability, useKnirvgraphEmbeddings } from '@/hooks/use-knirvgraph-scalability';
import { useKNIRVOracleEconomics } from '@/hooks/use-knirvoracle-economics';

function MetricTile({ name, value, help }: { name: string; value: number; help: string }) {
  return (
    <Card className="aether-bevel-dark rounded-xl">
      <CardHeader className="pb-2">
        <CardTitle className="text-xs text-gray-400 font-mono truncate" title={name}>
          {name}
        </CardTitle>
        <CardDescription className="text-[10px] text-gray-600">
          {help}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="text-xl font-bold text-gray-200">
          {typeof value === 'number' ? value.toLocaleString() : value}
        </div>
      </CardContent>
    </Card>
  );
}

function HealthBadge({ status }: { status: string }) {
  const color = status === 'ok' || status === 'up' ? 'text-emerald-400' : 'text-red-400';
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${color} bg-gray-900/50 border border-gray-800`}>
      {status}
    </span>
  );
}

export function NetworkMonitorMetrics() {
  const knirvbase = useKnirvbaseMetrics();
  const knirvchain = useKnirvchainMetrics();
  const knirvgraph = useKnirvgraphScalability();
  const knirvgraphEmbeddings = useKnirvgraphEmbeddings();
  const knirvoracle = useKNIRVOracleEconomics();
  const knirvbaseHealth = useKnirvbaseHealth();
  const knirvchainHealth = useKnirvchainHealth();

  const isLoading = knirvbase.isLoading && knirvchain.isLoading && knirvgraph.isLoading;
  const error = knirvbase.error || knirvchain.error || knirvgraph.error;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-gray-500">
        <Activity className="w-5 h-5 mr-2 animate-spin" />
        Loading metrics...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-amber-300">
        Metrics endpoint not yet available. This will be populated when KNIRVBASE/KNIRVCHAIN/KNIRVGRAPH nodes expose /metrics.
      </div>
    );
  }

  const baseMetrics: KnirvbaseMetric[] = knirvbase.data?.metrics ?? [];
  const chainMetrics: KnirvchainMetric[] = knirvchain.data?.metrics ?? [];
  const graphMetrics = knirvgraph.data?.metrics ?? [];
  const oracleMetrics = knirvoracle.data?.metrics ?? {};

  if (baseMetrics.length === 0 && chainMetrics.length === 0 && graphMetrics.length === 0 && Object.keys(oracleMetrics).length === 0) {
    return (
      <div className="rounded-lg border border-gray-800 bg-gray-900/30 p-4 text-gray-500">
        No metrics collected yet. Connect a KNIRVBASE, KNIRVCHAIN, or KNIRVGRAPH node to start collecting.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-200 flex items-center gap-2">
              <Server className="w-4 h-4 text-cyan-400" />
              KNIRVBASE
            </CardTitle>
            <CardDescription className="text-xs text-gray-500">
              Node health and Prometheus metrics
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-400">Status:</span>
              {knirvbaseHealth.data ? (
                <HealthBadge status={knirvbaseHealth.data.health.status} />
              ) : (
                <span className="text-xs text-gray-600">checking...</span>
              )}
            </div>
            {knirvbaseHealth.data && (
              <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-gray-400">
                <div>Blocks: {knirvbaseHealth.data.health.blocksCommitted.toLocaleString()}</div>
                <div>Connections: {knirvbaseHealth.data.health.activeConnections}</div>
                <div>Cache hit ratio: {(knirvbaseHealth.data.health.cacheHitRatio * 100).toFixed(1)}%</div>
                <div>Error rate: {knirvbaseHealth.data.health.errorRate}</div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-200 flex items-center gap-2">
              <Network className="w-4 h-4 text-amber-400" />
              KNIRVCHAIN
            </CardTitle>
            <CardDescription className="text-xs text-gray-500">
              Chain metrics and event rates
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-400">Status:</span>
              {knirvchainHealth.data ? (
                <HealthBadge status={knirvchainHealth.data.health.status} />
              ) : (
                <span className="text-xs text-gray-600">checking...</span>
              )}
            </div>
            {knirvchainHealth.data && (
              <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-gray-400">
                <div>Last check: {new Date(knirvchainHealth.data.health.lastCheck).toLocaleTimeString()}</div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {baseMetrics.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-200 mb-3 flex items-center gap-2">
            <HardDrive className="w-4 h-4 text-cyan-400" />
            KNIRVBASE Metrics
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {baseMetrics.map((metric) => (
              <MetricTile key={metric.name} name={metric.name} value={metric.value} help={metric.help} />
            ))}
          </div>
        </div>
      )}

      {chainMetrics.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-200 mb-3 flex items-center gap-2">
            <Cpu className="w-4 h-4 text-amber-400" />
            KNIRVCHAIN Metrics
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {chainMetrics.map((metric) => (
              <MetricTile key={metric.name} name={metric.name} value={metric.value} help={metric.help} />
            ))}
          </div>
        </div>
      )}

      {graphMetrics.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-200 mb-3 flex items-center gap-2">
            <GitBranch className="w-4 h-4 text-purple-400" />
            KNIRVGRAPH Metrics
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {graphMetrics.map((metric) => (
              <MetricTile key={metric.name} name={metric.name} value={metric.value} help={metric.help} />
            ))}
          </div>
        </div>
      )}

      {Object.keys(oracleMetrics).length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-200 mb-3 flex items-center gap-2">
            <DollarSign className="w-4 h-4 text-green-400" />
            KNIRVORACLE Economics
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {Object.entries(oracleMetrics).map(([key, value]) => (
              <Card key={key} className="aether-bevel-dark rounded-xl">
                <CardHeader className="pb-2">
                  <CardTitle className="text-xs text-gray-400 font-mono truncate" title={key}>
                    {key}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-xl font-bold text-gray-200">
                    {typeof value === 'number' ? value.toLocaleString() : JSON.stringify(value)}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
