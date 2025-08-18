// src/components/agents/AgentSearchForm.tsx
"use client";

import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Search, X } from 'lucide-react';
import type React from 'react';

interface AgentSearchFormProps {
  searchTerm: string;
  onSearchTermChange: (term: string) => void;
  onSearchSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
}

const AgentSearchForm = ({ searchTerm, onSearchTermChange, onSearchSubmit }: AgentSearchFormProps) => {
  
  const handleClearSearch = () => {
    onSearchTermChange('');
  };

  return (
    <form onSubmit={onSearchSubmit} className="mb-8">
      <div className="flex w-full max-w-2xl mx-auto items-center space-x-2 bg-card p-2 rounded-lg shadow-sm border">
        <div className="relative flex-grow">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search agents by name, capability, or provider..."
            value={searchTerm}
            onChange={(e) => onSearchTermChange(e.target.value)}
            className="pl-10 pr-10 h-12 text-base border-0 focus-visible:ring-0 focus-visible:ring-offset-0 shadow-none"
            aria-label="Search agents"
          />
          {searchTerm && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8 rounded-full"
              onClick={handleClearSearch}
              aria-label="Clear search"
            >
              <X className="h-5 w-5 text-muted-foreground" />
            </Button>
          )}
        </div>
        {/* <Button type="submit" size="lg" className="h-12">
          <Search className="mr-2 h-4 w-4 md:hidden" />
          <span className="hidden md:inline">Search</span>
        </Button> */}
      </div>
    </form>
  );
};

export default AgentSearchForm;
