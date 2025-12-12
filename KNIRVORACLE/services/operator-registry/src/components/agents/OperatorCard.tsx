// src/components/agents/OperatorCard.tsx
import Link from 'next/link';
import Image from 'next/image';
import type { Operator } from '@/lib/types';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import AgentCapabilityIcon from '@/components/icons/AgentCapabilityIcon';
import { ArrowRight, CalendarDays } from 'lucide-react';
import { ExternalLink } from 'lucide-react';

interface OperatorCardProps {
  operator: Operator;
}

const OperatorCard = ({ operator }: OperatorCardProps) => {
  const shortDescription = operator.description ?
    (operator.description.length > 100 ? `${operator.description.substring(0, 100)}...` : operator.description)
    : 'No description available.';

  return (
    <Card className="flex flex-col h-full hover:shadow-lg transition-shadow duration-300 ease-in-out">
      <CardHeader className="flex flex-row items-start gap-4 space-y-0 pb-3">
        {operator.avatarUrl && (
          <Image
            src={operator.avatarUrl}
            alt={`${operator.name} avatar`}
            width={60}
            height={60}
            className="rounded-lg border"
            data-ai-hint={operator.dataAiHint || "operator avatar"}
          />
        )}
        <div className="flex-1">
          <CardTitle className="text-xl mb-1">{operator.name}</CardTitle>
          <CardDescription className="flex items-center text-xs text-muted-foreground">
            <AgentCapabilityIcon capability={operator.capability} className="w-4 h-4 mr-1.5" />
            {operator.capability}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="flex-grow pb-3">
        <p className="text-sm text-muted-foreground mb-3">{shortDescription}</p>
        <div className="flex flex-wrap gap-1 mb-2">
          {operator.provider && <Badge variant="secondary">{operator.provider}</Badge>}
          {operator.version && <Badge variant="outline">v{operator.version}</Badge>}
        </div>
         {operator.registeredAt && (
            <p className="text-xs text-muted-foreground flex items-center mt-2">
              <CalendarDays className="w-3 h-3 mr-1.5" />
              Registered: {new Date(operator.registeredAt).toLocaleDateString()}
            </p>
          )}
      </CardContent>
      <CardFooter className="flex gap-2">
        <Button asChild variant="default" size="sm" className="flex-1">
          <Link href={`/operators/${operator.id}`}>
            View Details <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
        <Button asChild variant="outline" size="sm" className="w-12">
          <a href={`http://localhost:3003/status?highlight=${encodeURIComponent(operator.id)}`} target="_blank" rel="noreferrer" title="View Operator Status on Registry">
            <ExternalLink className="h-4 w-4" />
          </a>
        </Button>
      </CardFooter>
    </Card>
  );
};

export default OperatorCard;
