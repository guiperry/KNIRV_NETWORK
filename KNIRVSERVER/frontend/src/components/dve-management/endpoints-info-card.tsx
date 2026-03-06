"use client";

import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Terminal, CheckCircle, Zap, ExternalLink, Clock } from 'lucide-react';
import type { TEEEndpoint } from '@/types/api';

interface EndpointsInfoCardProps {
  endpoints: TEEEndpoint[];
  onEndpointClick?: (endpoint: TEEEndpoint) => void;
  className?: string;
}

export const EndpointsInfoCard: React.FC<EndpointsInfoCardProps> = ({
  endpoints,
  onEndpointClick,
  className,
}) => {
  const getEndpointIcon = (endpointType: string) => {
    switch (endpointType) {
      case 'ssh':
        return <Terminal className="w-4 h-4 text-green-500" />;
      case 'validation':
        return <CheckCircle className="w-4 h-4 text-blue-500" />;
      case 'error-resolution':
        return <Zap className="w-4 h-4 text-orange-500" />;
      default:
        return <ExternalLink className="w-4 h-4 text-gray-500" />;
    }
  };

  const getEndpointColor = (endpointType: string) => {
    switch (endpointType) {
      case 'ssh':
        return 'border-green-500/30 bg-green-500/5';
      case 'validation':
        return 'border-blue-500/30 bg-blue-500/5';
      case 'error-resolution':
        return 'border-orange-500/30 bg-orange-500/5';
      default:
        return 'border-gray-500/30 bg-gray-500/5';
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge className="bg-green-500 text-white text-xs">Active</Badge>;
      case 'inactive':
        return <Badge className="bg-yellow-500 text-white text-xs">Inactive</Badge>;
      case 'terminated':
        return <Badge className="bg-red-500 text-white text-xs">Terminated</Badge>;
      default:
        return <Badge className="bg-gray-500 text-white text-xs">{status}</Badge>;
    }
  };

  const formatEndpointType = (type: string) => {
    return type.split('-').map(word =>
      word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
  };

  if (endpoints.length === 0) {
    return (
      <Card className={`knirv-card-gradient ${className}`}>
        <CardContent className="p-6 text-center">
          <ExternalLink className="w-12 h-12 text-gray-500 mx-auto mb-4" />
          <p className="text-slate-400">No endpoints available</p>
          <p className="text-sm text-slate-500 mt-1">Endpoints will be available after container provisioning</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={`knirv-card-gradient ${className}`}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ExternalLink className="w-5 h-5" />
          Available Endpoints
        </CardTitle>
        <CardDescription>
          Access points for SSH, validation, and error resolution services
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {endpoints.map((endpoint, index) => (
            <div
              key={endpoint.id || index}
              className={`p-4 rounded-lg border ${getEndpointColor(endpoint.endpoint_type)} cursor-pointer hover:opacity-80 transition-opacity`}
              onClick={() => onEndpointClick?.(endpoint)}
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center space-x-3">
                  {getEndpointIcon(endpoint.endpoint_type)}
                  <div>
                    <h4 className="font-medium text-slate-200">
                      {formatEndpointType(endpoint.endpoint_type)}
                    </h4>
                    <p className="text-sm text-slate-400">
                      {endpoint.host}:{endpoint.port} ({endpoint.protocol.toUpperCase()})
                    </p>
                  </div>
                </div>
                {getStatusBadge(endpoint.status)}
              </div>

              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center space-x-2 text-slate-400">
                  <Clock className="w-3 h-3" />
                  <span>Expires: {new Date(endpoint.expires_at).toLocaleString()}</span>
                </div>
                <div className="text-xs text-slate-500">
                  ID: {endpoint.id.slice(-8)}
                </div>
              </div>

              {endpoint.credentials && (
                <div className="mt-3 p-2 bg-slate-700/50 rounded text-xs">
                  <div className="text-slate-400 mb-1">Credentials available</div>
                  <div className="font-mono text-slate-300">
                    {endpoint.credentials.username && `User: ${endpoint.credentials.username}`}
                    {endpoint.credentials.key_fingerprint && ` | Key: ${endpoint.credentials.key_fingerprint.slice(-16)}...`}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="mt-6 p-3 bg-blue-500/10 border border-blue-500/30 rounded-lg">
          <p className="text-sm text-blue-400 font-medium mb-1">How to Access</p>
          <p className="text-xs text-slate-300">
            Click on any endpoint above to open the corresponding access interface.
            SSH endpoints provide terminal access, validation endpoints offer reasoning tools,
            and error resolution endpoints provide debugging utilities.
          </p>
        </div>
      </CardContent>
    </Card>
  );
};

export default EndpointsInfoCard;
