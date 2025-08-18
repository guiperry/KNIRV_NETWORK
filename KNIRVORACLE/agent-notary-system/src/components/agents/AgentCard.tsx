// src/components/agents/AgentCard.tsx
import Link from 'next/link';
import Image from 'next/image';
import type { Agent } from '@/lib/types';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import AgentCapabilityIcon from '@/components/icons/AgentCapabilityIcon';
import { ArrowRight, CalendarDays } from 'lucide-react';

interface AgentCardProps {
  agent: Agent;
}

const AgentCard = ({ agent }: AgentCardProps) => {
  const shortDescription = agent.description ? 
    (agent.description.length > 100 ? `${agent.description.substring(0, 100)}...` : agent.description)
    : 'No description available.';

  return (
    <Card className="flex flex-col h-full hover:shadow-lg transition-shadow duration-300 ease-in-out">
      <CardHeader className="flex flex-row items-start gap-4 space-y-0 pb-3">
        {agent.avatarUrl && (
          <Image
            src={agent.avatarUrl}
            alt={`${agent.name} avatar`}
            width={60}
            height={60}
            className="rounded-lg border"
            data-ai-hint={agent.dataAiHint || "agent avatar"}
          />
        )}
        <div className="flex-1">
          <CardTitle className="text-xl mb-1">{agent.name}</CardTitle>
          <CardDescription className="flex items-center text-xs text-muted-foreground">
            <AgentCapabilityIcon capability={agent.capability} className="w-4 h-4 mr-1.5" />
            {agent.capability}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="flex-grow pb-3">
        <p className="text-sm text-muted-foreground mb-3">{shortDescription}</p>
        <div className="flex flex-wrap gap-1 mb-2">
          {agent.provider && <Badge variant="secondary">{agent.provider}</Badge>}
          {agent.version && <Badge variant="outline">v{agent.version}</Badge>}
        </div>
         {agent.registeredAt && (
            <p className="text-xs text-muted-foreground flex items-center mt-2">
              <CalendarDays className="w-3 h-3 mr-1.5" />
              Registered: {new Date(agent.registeredAt).toLocaleDateString()}
            </p>
          )}
      </CardContent>
      <CardFooter>
        <Button asChild variant="default" size="sm" className="w-full">
          <Link href={`/agents/${agent.id}`}>
            View Details <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </CardFooter>
    </Card>
  );
};

export default AgentCard;
