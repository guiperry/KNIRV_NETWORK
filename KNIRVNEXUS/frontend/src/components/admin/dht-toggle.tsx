"use client";

import React, { useState } from 'react';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useDHT } from '@/contexts/dht-context';
import { AlertTriangle, Network, WifiOff, Loader2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

export const DHTToggle: React.FC = () => {
  const { isDHTEnabled, toggleDHT, setDHTEnabled } = useDHT();
  const { toast } = useToast();
  const [isUpdating, setIsUpdating] = useState(false);

  const handleToggle = async (checked: boolean) => {
    setIsUpdating(true);

    try {
      // Call backend API to update DHT settings
      const response = await fetch('/api/system/dht', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ enabled: checked }),
      });

      if (!response.ok) {
        throw new Error('Failed to update DHT settings');
      }

      const data = await response.json();

      // Update local state
      setDHTEnabled(checked);

      toast({
        title: "DHT Settings Updated",
        description: `DHT has been ${checked ? 'enabled' : 'disabled'} successfully.`,
      });
    } catch (error) {
      console.error('Failed to update DHT settings:', error);
      toast({
        title: "Update Failed",
        description: "Failed to update DHT settings. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsUpdating(false);
    }
  };

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
          <div className="flex items-center gap-2">
            <div className={`relative ${!isDHTEnabled ? "ring-2 ring-red-500 rounded-full" : ""}`}>
              <Switch
                id="dht-toggle"
                checked={isDHTEnabled}
                onCheckedChange={handleToggle}
                disabled={isUpdating}
              />
              {!isDHTEnabled && (
                <div className="absolute -top-1 -right-1 w-3 h-3 bg-red-500 rounded-full animate-pulse"></div>
              )}
            </div>
            {isUpdating && (
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            )}
          </div>
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
