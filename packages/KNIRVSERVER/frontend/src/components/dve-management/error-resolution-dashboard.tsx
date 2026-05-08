'use client';

import React from 'react';
import { AlertTriangle, TerminalSquare, Zap } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import WebTerminal from './web-terminal';

interface ErrorResolutionDashboardProps {
  sessionId: string;
  supportedTypes: string[];
  port?: number;
  className?: string;
}

export const ErrorResolutionDashboard: React.FC<ErrorResolutionDashboardProps> = ({
  sessionId,
  supportedTypes,
  port,
  className = '',
}) => {
  return (
    <div className={`space-y-6 ${className}`}>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TerminalSquare className="w-5 h-5" />
            <span>Error Resolution Terminal</span>
          </CardTitle>
          <CardDescription>
            Live xterm session proxied through the backend for the DVE error-resolution service.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <WebTerminal
            sessionId={sessionId}
            endpoint={`127.0.0.1:${port ?? 0}`}
            wsPath={`/ws/error-resolution/${sessionId}`}
            connectedMessage={`Connected to error-resolution service${port ? ` on port ${port}` : ''}`}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-orange-500" />
            <span>Supported Modes</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap gap-2">
            {supportedTypes.map((type) => (
              <Badge key={type} variant="outline">
                {type}
              </Badge>
            ))}
          </div>
          <div className="flex items-start gap-2 text-sm text-muted-foreground">
            <AlertTriangle className="w-4 h-4 mt-0.5 text-yellow-500" />
            <span>
              This terminal is restricted to the backend-managed error-resolution session range `24000-24999`.
            </span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default ErrorResolutionDashboard;
