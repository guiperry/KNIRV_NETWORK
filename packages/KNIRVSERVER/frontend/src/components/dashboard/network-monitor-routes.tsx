'use client';

import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { useGatewayRoutes, type GatewayRoute } from '@/hooks/use-gateway-routes';
import { Activity } from 'lucide-react';

export function NetworkMonitorRoutes() {
  const { data, isLoading, error } = useGatewayRoutes();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-gray-500">
        <Activity className="w-5 h-5 mr-2 animate-spin" />
        Loading routes...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-amber-300">
        Gateway route table endpoint not yet available. This will be populated when KNIRVGATEWAY exposes /api/network-monitor/routes.
      </div>
    );
  }

  const routes: GatewayRoute[] = data?.routes ?? [];

  if (routes.length === 0) {
    return (
      <div className="rounded-lg border border-gray-800 bg-gray-900/30 p-4 text-gray-500">
        No proxy routes registered yet.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold text-gray-200">Gateway Proxy Routes</h3>
        <p className="text-sm text-gray-500">
          Registered mux routes with live status and latency
        </p>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 text-left text-gray-500">
              <th className="pb-2 font-medium">Name</th>
              <th className="pb-2 font-medium">Path Prefix</th>
              <th className="pb-2 font-medium">Target</th>
              <th className="pb-2 font-medium">Protocol</th>
              <th className="pb-2 font-medium">Status</th>
              <th className="pb-2 font-medium text-right">Latency</th>
            </tr>
          </thead>
          <tbody>
            {routes.map((route) => (
              <tr key={route.name + route.pathPrefix} className="border-b border-gray-800/50">
                <td className="py-2 font-mono text-gray-300">{route.name}</td>
                <td className="py-2 font-mono text-cyan-400">{route.pathPrefix}</td>
                <td className="py-2 font-mono text-gray-400">{route.target}</td>
                <td className="py-2 text-gray-400">{route.protocol}</td>
                <td className="py-2">
                  <Badge
                    variant="outline"
                    className={
                      route.status === 'up'
                        ? 'border-green-500/30 text-green-300'
                        : route.status === 'down'
                          ? 'border-red-500/30 text-red-300'
                          : 'border-gray-500/30 text-gray-300'
                    }
                  >
                    {route.status}
                  </Badge>
                </td>
                <td className="py-2 text-right font-mono text-gray-400">
                  {route.latencyMs}ms
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
