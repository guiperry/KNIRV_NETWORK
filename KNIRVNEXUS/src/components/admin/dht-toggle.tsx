"use client";

import React from 'react';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useDHT } from '@/contexts/dht-context';
import { AlertTriangle, Network, WifiOff } from 'lucide-react';

export const DHTToggle: React.FC = () => {
  const { isDHTEnabled, toggleDHT } = useDHT();

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {isDHTEnabled ? (
            <Network className="h-5 w-5 text-green-500" />
          ) : (
            <WifiOff className="h-5 w-5 text-red-500" />
          )}
          DHT Configuration
        </CardTitle>
        <CardDescription>
          Control whether the Distributed Hash Table (DHT) is enabled for peer discovery and resource sharing
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label htmlFor="dht-toggle" className="text-base font-medium">
              Enable DHT
            </Label>
            <div className="text-sm text-muted-foreground">
              {isDHTEnabled
                ? "DHT is active for peer discovery and network coordination"
                : "DHT is disabled to reduce network noise and resource usage"
              }
            </div>
          </div>
          <Switch
            id="dht-toggle"
            checked={isDHTEnabled}
            onCheckedChange={toggleDHT}
          />
        </div>

        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm font-medium">
            <AlertTriangle className="h-4 w-4 text-amber-500" />
            Current Status: {isDHTEnabled ? "Enabled" : "Disabled"}
          </div>

          {isDHTEnabled ? (
            <div className="text-sm text-muted-foreground space-y-2">
              <p><strong>DHT Active:</strong></p>
              <ul className="list-disc list-inside space-y-1 ml-4">
                <li>Peer discovery and network coordination enabled</li>
                <li>Resource announcements and lookups active</li>
                <li>Increased network traffic and resource usage</li>
                <li>Full P2P functionality available</li>
              </ul>
            </div>
          ) : (
            <div className="text-sm text-muted-foreground space-y-2">
              <p><strong>DHT Disabled:</strong></p>
              <ul className="list-disc list-inside space-y-1 ml-4">
                <li>Reduced network noise and background activity</li>
                <li>Limited peer discovery capabilities</li>
                <li>Lower resource consumption</li>
                <li>Local operations only (no network-wide coordination)</li>
              </ul>
            </div>
          )}
        </div>

        <div className="text-xs text-muted-foreground">
          <p>
            <strong>Note:</strong> This setting controls DHT participation for peer discovery and resource sharing.
            When disabled, the system operates in local-only mode with reduced network activity.
            Changes take effect immediately and are saved locally.
          </p>
        </div>
      </CardContent>
    </Card>
  );
};
