"use client";

import React from 'react';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useDemoMode } from '@/contexts/demo-mode-context';
import { AlertTriangle, Database, TestTube } from 'lucide-react';

export const DemoModeToggle: React.FC = () => {
  const { isDemoMode, toggleDemoMode } = useDemoMode();

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {isDemoMode ? (
            <TestTube className="h-5 w-5 text-blue-500" />
          ) : (
            <Database className="h-5 w-5 text-green-500" />
          )}
          Demo Mode Configuration
        </CardTitle>
        <CardDescription>
          Control whether the application shows demo data or connects to real backend services
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label htmlFor="demo-mode" className="text-base font-medium">
              Demo Mode
            </Label>
            <div className="text-sm text-muted-foreground">
              {isDemoMode 
                ? "Showing mock data and simulated responses" 
                : "Connected to live backend services and database"
              }
            </div>
          </div>
          <Switch
            id="demo-mode"
            checked={isDemoMode}
            onCheckedChange={toggleDemoMode}
          />
        </div>

        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm font-medium">
            <AlertTriangle className="h-4 w-4 text-amber-500" />
            Current Mode: {isDemoMode ? "Demo" : "Production"}
          </div>
          
          {isDemoMode ? (
            <div className="text-sm text-muted-foreground space-y-2">
              <p><strong>Demo Mode Active:</strong></p>
              <ul className="list-disc list-inside space-y-1 ml-4">
                <li>Mock data for DVE nodes, validation tasks, and models</li>
                <li>Simulated real-time updates and metrics</li>
                <li>No actual backend connections required</li>
                <li>Safe for testing and demonstrations</li>
              </ul>
            </div>
          ) : (
            <div className="text-sm text-muted-foreground space-y-2">
              <p><strong>Production Mode Active:</strong></p>
              <ul className="list-disc list-inside space-y-1 ml-4">
                <li>Live data from backend database</li>
                <li>Real WebSocket connections for updates</li>
                <li>Actual DVE node management and validation</li>
                <li>Empty states shown when no data available</li>
              </ul>
            </div>
          )}
        </div>

        <div className="text-xs text-muted-foreground">
          <p>
            <strong>Note:</strong> This setting is saved locally and will persist between sessions.
            Changing this setting will refresh the page to apply the new mode.
          </p>
        </div>
      </CardContent>
    </Card>
  );
};
