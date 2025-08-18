
// src/app/agents/[id]/page.tsx
import { notFound } from 'next/navigation';
import { getAgentById } from '@/lib/agent-service'; // Updated import
import AgentProfileDetails from '@/components/agents/AgentProfileDetails';
import { generateAgentSummary } from '@/ai/flows/agent-summary'; // GenAI import
import type { AgentSummaryInput, AgentSummaryOutput } from '@/ai/flows/agent-summary';
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { BotMessageSquare } from 'lucide-react';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

interface AgentProfilePageProps {
  params: {
    id: string;
  };
}

export async function generateMetadata({ params }: AgentProfilePageProps) {
  const decodedId = decodeURIComponent(params.id);
  const agent = await getAgentById(decodedId);
  if (!agent) {
    return {
      title: 'Agent Not Found',
    };
  }
  return {
    title: `${agent.name} - Agent Profile | AgentVerse Registry`,
    description: agent.description || `Profile page for AI agent ${agent.name}.`,
  };
}

export default async function AgentProfilePage({ params }: AgentProfilePageProps) {
  const decodedId = decodeURIComponent(params.id);
  const agent = await getAgentById(decodedId);

  if (!agent) {
    notFound();
  }

  let aiSummary: string | null = null;
  try {
    const summaryInput: AgentSummaryInput = {
      agentName: agent.name,
      agentCapabilities: agent.capabilities?.join(', ') || agent.capability,
      agentDescription: agent.description,
    };
    const summaryOutput: AgentSummaryOutput = await generateAgentSummary(summaryInput);
    aiSummary = summaryOutput.summary;
  } catch (error) {
    console.error("Error generating AI summary:", error);
    // aiSummary will remain null, UI should handle this
  }
  
  return (
    <div className="space-y-8">
       <Button variant="outline" asChild className="mb-6">
        <Link href="/">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Discovery
        </Link>
      </Button>
      <AgentProfileDetails agent={agent} aiSummary={aiSummary} />
      {!aiSummary && (
        <Alert variant="default" className="mt-6">
          <BotMessageSquare className="h-4 w-4" />
          <AlertTitle>AI Summary Note</AlertTitle>
          <AlertDescription>
            The AI-generated summary for this agent could not be loaded at this time.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
