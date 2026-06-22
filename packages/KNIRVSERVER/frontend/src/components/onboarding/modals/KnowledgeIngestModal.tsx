'use client';

import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Database, Loader2 } from "lucide-react";

interface KnowledgeIngestModalProps {
  isOpen: boolean;
  onClose: () => void;
  ingestUrl: string;
  setIngestUrl: (value: string) => void;
  ingestLog: string[];
  isIngesting: boolean;
  onIngest: () => void;
  ingestedRepos: string[];
}

export function KnowledgeIngestModal({
  isOpen,
  onClose,
  ingestUrl,
  setIngestUrl,
  ingestLog,
  isIngesting,
  onIngest,
  ingestedRepos,
}: KnowledgeIngestModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-[96vw] max-w-5xl max-h-[92vh] overflow-hidden bg-[#0a0a0c] border-white/10 text-slate-200 flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Database className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">Knowledge Repository Ingestion</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Import repositories or documents into the graph from this modal instead of inline on the setup page.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 pr-1 custom-scrollbar space-y-4">
          <p className="text-xs text-slate-400">
            Enter a repository URL and queue it for ingestion into the knowledge graph.
          </p>
          <div className="flex gap-2">
            <Input
              type="text"
              value={ingestUrl}
              onChange={(e) => setIngestUrl(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  onIngest();
                }
              }}
              placeholder="https://github.com/org/repo"
              className="flex-1 bg-black/40 border-white/10 text-white text-sm font-mono"
            />
            <Button
              onClick={onIngest}
              disabled={isIngesting || !ingestUrl.trim()}
              className="bg-blue-600 hover:bg-blue-500"
            >
              {isIngesting ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Ingest'}
            </Button>
          </div>

          {ingestLog.length > 0 && (
            <div className="bg-black/40 rounded p-3 font-mono text-[10px] text-green-400 max-h-32 overflow-y-auto">
              {ingestLog.map((line, i) => <div key={i}>{line}</div>)}
            </div>
          )}

          {ingestedRepos.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {ingestedRepos.map((repo, i) => (
                <Badge key={i} variant="outline" className="border-blue-500/30 text-blue-400 text-xs">
                  {repo.split('/').slice(-2).join('/')}
                </Badge>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default KnowledgeIngestModal;
