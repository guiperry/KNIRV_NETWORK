
// src/app/page.tsx
"use client"; // For client-side search filtering

import { useState, useEffect, useMemo } from 'react';
import type { Agent } from '@/lib/types';
import { getAllAgents } from '@/lib/agent-service'; // Updated import
import AgentCard from '@/components/agents/AgentCard';
import AgentSearchForm from '@/components/agents/AgentSearchForm';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { BotMessageSquare, ShieldCheck } from 'lucide-react';

export default function DiscoverAgentsPage() {
  const [allAgents, setAllAgents] = useState<Agent[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchAgents = async () => {
      try {
        setIsLoading(true);
        const agents = await getAllAgents();
        setAllAgents(agents);
        setError(null);
      } catch (err) {
        console.error("Failed to fetch agents:", err);
        setError("Could not load agents. Please try again later.");
      } finally {
        setIsLoading(false);
      }
    };
    fetchAgents();
  }, []);

  const filteredAgents = useMemo(() => {
    if (!searchTerm) return allAgents;
    return allAgents.filter(agent =>
      agent.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      agent.capability.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (agent.provider && agent.provider.toLowerCase().includes(searchTerm.toLowerCase())) ||
      (agent.description && agent.description.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [searchTerm, allAgents]);

  const handleSearchSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    // Search is already live filtering, so submit doesn't need to do much more
    // but could be used for more complex server-side search in future
  };

  return (
    <div className="space-y-8">
      <section className="text-center py-8 bg-gradient-to-r from-primary/10 via-background to-accent/10 rounded-xl shadow-sm">
        <div className="inline-flex items-center justify-center mb-4">
         <ShieldCheck className="h-16 w-16 text-primary" />
        </div>
        <h1 className="text-4xl font-extrabold tracking-tight text-primary lg:text-5xl mb-4">
          Discover Verifiable AI Agents
        </h1>
        <p className="max-w-3xl mx-auto text-lg text-muted-foreground">
          Explore a tamper-evident, dual-trust registry for AI agents. Our NANDA+ANS architecture uses a decoupled, two-hop lookup model: the core NANDA Registry (Anchor Tier, now leveraging Firestore) provides lightweight, signed pointers (AgentAddr) to detailed, cryptographically verifiable AgentFacts (Metadata Distribution Tier, structured by ANS principles). Discover agents with assured capabilities through privacy-preserving techniques, building a resilient and trustworthy Internet of AI Agents.
        </p>
      </section>

      <AgentSearchForm
        searchTerm={searchTerm}
        onSearchTermChange={setSearchTerm}
        onSearchSubmit={handleSearchSubmit}
      />

      {error && (
         <Alert variant="destructive">
          <BotMessageSquare className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {[...Array(6)].map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : filteredAgents.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {filteredAgents.map(agent => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      ) : (
        <Alert className="mt-8">
           <BotMessageSquare className="h-4 w-4" />
           <AlertTitle>No Agents Found</AlertTitle>
           <AlertDescription>
             Your search for "{searchTerm}" did not match any agents. Try a different term or explore all agents. If this is your first time, try registering an agent!
           </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

const CardSkeleton = () => (
  <div className="flex flex-col space-y-3 p-4 border rounded-lg shadow">
    <div className="flex items-center space-x-4">
      <Skeleton className="h-12 w-12 rounded-lg" />
      <div className="space-y-2 flex-1">
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-3 w-1/2" />
      </div>
    </div>
    <Skeleton className="h-4 w-full" />
    <Skeleton className="h-4 w-5/6" />
    <div className="flex justify-between items-center pt-2">
      <Skeleton className="h-8 w-1/3" />
      <Skeleton className="h-8 w-1/4" />
    </div>
    <Skeleton className="h-10 w-full mt-2" />
  </div>
);
