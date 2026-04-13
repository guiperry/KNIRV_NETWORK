'use client';

import React, { useState } from 'react';
import { Network, Link2, RefreshCw, Search, Server, BookOpen, Bug, Shield, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useOntology, useOntologySearch, OntologyEntity, OntologyStats } from '@/hooks/use-ontology';

const entityTypeIcons: Record<string, React.ReactNode> = {
  dve_node: <Server className="w-4 h-4 text-blue-400" />,
  validation_task: <BookOpen className="w-4 h-4 text-green-400" />,
  validation_result: <CheckCircle className="w-4 h-4 text-green-400" />,
  adaptation_event: <RefreshCw className="w-4 h-4 text-purple-400" />,
  failure_pattern: <Bug className="w-4 h-4 text-red-400" />,
  guardrail_policy: <Shield className="w-4 h-4 text-cyan-400" />,
  policy_violation: <AlertTriangle className="w-4 h-4 text-red-400" />,
};

const entityTypeColors: Record<string, string> = {
  dve_node: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  validation_task: 'bg-green-500/20 text-green-400 border-green-500/30',
  validation_result: 'bg-green-500/20 text-green-400 border-green-500/30',
  adaptation_event: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  failure_pattern: 'bg-red-500/20 text-red-400 border-red-500/30',
  guardrail_policy: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
  policy_violation: 'bg-red-500/20 text-red-400 border-red-500/30',
};

interface KnowledgeGraphOverviewProps {
  className?: string;
}

export function KnowledgeGraphOverview({ className }: KnowledgeGraphOverviewProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const { stats, entities, relations, refetch } = useOntology();
  const { data: searchResults, isLoading: isSearching } = useOntologySearch(searchQuery);

  const displayEntities = searchQuery.length > 0 ? searchResults : entities.data;

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Network className="w-5 h-5 text-green-400" />
            <CardTitle>Knowledge Graph</CardTitle>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4 mr-1" />
            Refresh
          </Button>
        </div>
        <CardDescription>Ontology entity and relation overview</CardDescription>
      </CardHeader>
      <CardContent>
        {/* Stats Summary */}
        {stats.isLoading && (
          <div className="flex items-center justify-center py-4">
            <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {stats.data && (
          <div className="grid grid-cols-2 gap-3 mb-4">
            <div className="p-3 rounded-lg bg-card/50 border border-border text-center">
              <div className="text-2xl font-bold">{stats.data.entity_count}</div>
              <div className="text-xs text-muted-foreground">Entities</div>
            </div>
            <div className="p-3 rounded-lg bg-card/50 border border-border text-center">
              <div className="text-2xl font-bold">{stats.data.relation_count}</div>
              <div className="text-xs text-muted-foreground">Relations</div>
            </div>
          </div>
        )}

        {/* Entity Type Breakdown */}
        {stats.data?.entity_types && (
          <div className="mb-4">
            <h4 className="text-sm font-medium mb-2">Entity Types</h4>
            <div className="flex flex-wrap gap-2">
              {Object.entries(stats.data.entity_types).map(([type, count]) => (
                <Badge key={type} variant="outline" className={entityTypeColors[type] || 'bg-card'}>
                  {type.replace('_', ' ')}: {count}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {/* Search */}
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search entities..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        {/* Entity List */}
        {entities.isLoading && !searchQuery && (
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {displayEntities && displayEntities.length > 0 && (
          <ScrollArea className="h-[250px]">
            <div className="space-y-2">
              {displayEntities.slice(0, 20).map((entity) => (
                <div
                  key={entity.id}
                  className="p-3 rounded-lg bg-card/50 border border-border hover:bg-card transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <div className="mt-0.5">
                      {entityTypeIcons[entity.type] || <Network className="w-4 h-4 text-muted-foreground" />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium truncate">{entity.label}</span>
                        <Badge variant="secondary" className="text-xs">
                          {entity.type}
                        </Badge>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        ID: {entity.id}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}

        {displayEntities && displayEntities.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <Network className="w-10 h-10 text-muted-foreground mb-2" />
            <p>No entities found</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function CheckCircle(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg {...props} xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  );
}

export default KnowledgeGraphOverview;
