'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Activity, Network, Route, BarChart3, Users, AlertTriangle } from 'lucide-react';
import { NetworkMonitorHealth } from './network-monitor-health';
import { NetworkMonitorMetrics } from './network-monitor-metrics';
import { NetworkMonitorRoutes } from './network-monitor-routes';
import { NetworkMonitorGrafana } from './network-monitor-grafana';
import { NetworkMonitorPrometheus } from './network-monitor-prometheus';
import { NetworkMonitorOnboarding } from './network-monitor-onboarding';
import { NetworkMonitorAlerts } from './network-monitor-alerts';

type MonitorView = 'health' | 'metrics' | 'routes' | 'grafana' | 'prometheus' | 'onboarding' | 'alerts';

const subTabs: { value: MonitorView; label: string; icon: React.ReactNode }[] = [
  { value: 'health', label: 'Health', icon: <Activity className="w-4 h-4" /> },
  { value: 'metrics', label: 'Metrics', icon: <BarChart3 className="w-4 h-4" /> },
  { value: 'routes', label: 'Routes', icon: <Route className="w-4 h-4" /> },
  { value: 'grafana', label: 'Grafana', icon: <Activity className="w-4 h-4" /> },
  { value: 'prometheus', label: 'Prometheus', icon: <Network className="w-4 h-4" /> },
  { value: 'onboarding', label: 'Onboarding', icon: <Users className="w-4 h-4" /> },
  { value: 'alerts', label: 'Alerts', icon: <AlertTriangle className="w-4 h-4" /> },
];

export function NetworkMonitorPanel() {
  const [monitorView, setMonitorView] = useState<MonitorView>('health');

  return (
    <Card className="aether-bevel-dark rounded-2xl">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-gray-200">
              <Network className="w-5 h-5 text-cyan-400" />
              Network Monitor
            </CardTitle>
          </div>
          <Badge variant="outline" className="text-cyan-400 border-cyan-500/30">
            Admin Only
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <Tabs value={monitorView} onValueChange={(v) => setMonitorView(v as MonitorView)} className="space-y-4">
          <TabsList className="grid w-full grid-cols-7 bg-gray-900/50 border border-gray-800">
            {subTabs.map((tab) => (
              <TabsTrigger
                key={tab.value}
                value={tab.value}
                className="text-gray-400 data-[state=active]:text-cyan-400 data-[state=active]:bg-cyan-500/10"
              >
                <span className="hidden lg:inline-flex items-center gap-1">
                  {tab.icon}
                  {tab.label}
                </span>
                <span className="lg:hidden">{tab.label}</span>
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="health" className="space-y-4">
            <NetworkMonitorHealth />
          </TabsContent>

          <TabsContent value="metrics" className="space-y-4">
            <NetworkMonitorMetrics />
          </TabsContent>

          <TabsContent value="routes" className="space-y-4">
            <NetworkMonitorRoutes />
          </TabsContent>

          <TabsContent value="grafana" className="space-y-4">
            <NetworkMonitorGrafana />
          </TabsContent>

          <TabsContent value="prometheus" className="space-y-4">
            <NetworkMonitorPrometheus />
          </TabsContent>

          <TabsContent value="onboarding" className="space-y-4">
            <NetworkMonitorOnboarding />
          </TabsContent>

          <TabsContent value="alerts" className="space-y-4">
            <NetworkMonitorAlerts />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
