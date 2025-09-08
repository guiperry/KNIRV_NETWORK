// src/components/agents/AgentProfileDetails.tsx
import type { Agent } from '@/lib/types';
import Image from 'next/image';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import AgentCapabilityIcon from '@/components/icons/AgentCapabilityIcon';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { AlertCircle, CheckCircle2, ShieldQuestion, Globe, GitBranch, Tag, Info, Layers3, PlugZap, CalendarDays, Clock, Fingerprint, Link2, KeyRound, Building } from 'lucide-react';

interface AgentProfileDetailsProps {
  agent: Agent;
  aiSummary: string | null; // Pass AI summary as prop
}

const AgentProfileDetails = ({ agent, aiSummary }: AgentProfileDetailsProps) => {
  return (
    <div className="space-y-6">
      <Card className="overflow-hidden shadow-lg">
        <CardHeader className="bg-gradient-to-br from-primary/20 via-background to-accent/10 p-6">
          <div className="flex flex-col md:flex-row items-start gap-6">
            {agent.avatarUrl && (
              <Image
                src={agent.avatarUrl}
                alt={`${agent.name} avatar`}
                width={128}
                height={128}
                className="rounded-xl border-4 border-background shadow-md"
                data-ai-hint={agent.dataAiHint || "agent avatar"}
              />
            )}
            <div className="flex-1">
              <CardTitle className="text-3xl font-bold text-primary mb-1">{agent.name}</CardTitle>
              <CardDescription className="text-lg text-muted-foreground flex items-center">
                <AgentCapabilityIcon capability={agent.capability} className="w-5 h-5 mr-2 text-accent" />
                {agent.capability}
              </CardDescription>
              {agent.provider && (
                <p className="text-sm text-muted-foreground mt-1">
                  Provider: <span className="font-semibold text-foreground">{agent.provider}</span>
                </p>
              )}
              {agent.version && (
                 <div className="text-sm text-muted-foreground mt-1">
                  Version: <Badge variant="secondary">{agent.version}</Badge>
                  {agent.extension && <Badge variant="outline" className="ml-2">{agent.extension}</Badge>}
                </div>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-6 space-y-6">
          {aiSummary && (
            <Card>
              <CardHeader>
                <CardTitle className="text-xl flex items-center"><Info className="mr-2 h-5 w-5 text-primary"/>AI Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-base text-foreground">{aiSummary}</p>
              </CardContent>
            </Card>
          )}

          {agent.description && (
            <Card>
              <CardHeader>
                <CardTitle className="text-xl">Description</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-foreground whitespace-pre-wrap">{agent.description}</p>
              </CardContent>
            </Card>
          )}
          
          <Accordion type="single" collapsible className="w-full" defaultValue="item-ans-nanda">
            <AccordionItem value="item-ans-nanda">
                <AccordionTrigger className="text-lg font-semibold"><ShieldQuestion className="mr-2 h-5 w-5 text-primary"/>NANDA+ANS Registry Details</AccordionTrigger>
                <AccordionContent className="pt-2 space-y-1">
                    <p className="text-sm text-foreground"><Globe className="inline h-4 w-4 mr-1.5 text-muted-foreground"/><strong>Agent DID (NANDA Identifier):</strong> {agent.id}</p>
                    {agent.ansName && <p className="text-sm text-foreground"><Tag className="inline h-4 w-4 mr-1.5 text-muted-foreground"/><strong>ANS Name (Capability Address):</strong> {agent.ansName}</p>}
                    {agent.addr_facts_url && <p className="text-sm text-foreground break-all"><Link2 className="inline h-4 w-4 mr-1.5 text-muted-foreground"/><strong>NANDA Facts URL Pointer:</strong> {agent.addr_facts_url}</p>}
                    {agent.addr_ttl && <p className="text-sm text-foreground"><Clock className="inline h-4 w-4 mr-1.5 text-muted-foreground"/><strong>NANDA Pointer TTL:</strong> {agent.addr_ttl} seconds</p>}
                    {agent.registeredAt && (
                        <p className="text-sm text-foreground"><CalendarDays className="inline h-4 w-4 mr-1.5 text-muted-foreground"/>
                        <strong>Registered:</strong> {new Date(agent.registeredAt).toLocaleString()}
                        </p>
                    )}
                    <p className="text-xs text-muted-foreground pt-2">
                      NANDA provides a lightweight, signed pointer (AgentAddr) including the DID, Facts URL, and TTL. This points to the detailed, cryptographically verifiable AgentFacts hosted on the Metadata Distribution Tier, structured according to ANS principles. Agent identity and capability attestations are secured via PKI mechanisms, simulated here for demonstration.
                    </p>
                </AccordionContent>
            </AccordionItem>

            <AccordionItem value="item-capabilities">
              <AccordionTrigger className="text-lg font-semibold"><Layers3 className="mr-2 h-5 w-5 text-primary"/>Capabilities</AccordionTrigger>
              <AccordionContent className="pt-2">
                {agent.capabilities && agent.capabilities.length > 0 ? (
                  <ul className="list-disc list-inside space-y-1 pl-2">
                    {agent.capabilities.map((cap, index) => (
                      <li key={index} className="text-foreground flex items-center">
                        <AgentCapabilityIcon capability={cap} className="w-4 h-4 mr-2 text-muted-foreground" /> {cap}
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-muted-foreground">No detailed capabilities listed.</p>
                )}
              </AccordionContent>
            </AccordionItem>

            {agent.endpoints && (
            <AccordionItem value="item-endpoints">
              <AccordionTrigger className="text-lg font-semibold"><PlugZap className="mr-2 h-5 w-5 text-primary"/>Endpoints</AccordionTrigger>
              <AccordionContent className="pt-2 space-y-2">
                {agent.endpoints.static_endpoint?.map((ep, i) => (
                  <p key={`static-${i}`} className="text-sm text-foreground break-all">
                    <strong className="text-muted-foreground">Static:</strong> {ep}
                  </p>
                ))}
                {agent.endpoints.adaptive_router_url && (
                  <p className="text-sm text-foreground break-all">
                    <strong className="text-muted-foreground">Adaptive Router:</strong> {agent.endpoints.adaptive_router_url}
                  </p>
                )}
              </AccordionContent>
            </AccordionItem>
            )}
            
            {agent.attestations && agent.attestations.length > 0 && (
              <AccordionItem value="item-attestations">
                <AccordionTrigger className="text-lg font-semibold"><CheckCircle2 className="mr-2 h-5 w-5 text-green-600"/>Attestations</AccordionTrigger>
                <AccordionContent className="pt-2">
                  <ul className="list-disc list-inside space-y-1 pl-2">
                    {agent.attestations.map((att, index) => (
                      <li key={index} className="text-foreground text-sm break-all">{att}</li>
                    ))}
                  </ul>
                </AccordionContent>
              </AccordionItem>
            )}

            {agent.protocolExtensions && Object.keys(agent.protocolExtensions).filter(k => k !== "@type").length > 0 && (
              <AccordionItem value="item-protocol-extensions">
                <AccordionTrigger className="text-lg font-semibold"><GitBranch className="mr-2 h-5 w-5 text-primary"/>Protocol Extensions</AccordionTrigger>
                <AccordionContent className="pt-2 space-y-3">
                  {Object.entries(agent.protocolExtensions)
                    .filter(([key]) => key !== "@type")
                    .map(([protocol, ext]) => (
                    <div key={protocol} className="p-3 border rounded-md bg-background/50">
                      <h4 className="font-semibold text-foreground uppercase">{protocol} ({ext.name || 'Details'})</h4>
                      {ext.details && <pre className="mt-1 text-xs bg-muted/50 p-2 rounded-md overflow-x-auto">{JSON.stringify(ext.details, null, 2)}</pre>}
                    </div>
                  ))}
                </AccordionContent>
              </AccordionItem>
            )}
            
            {agent.signature && (
              <AccordionItem value="item-signature">
                <AccordionTrigger className="text-lg font-semibold"><Fingerprint className="mr-2 h-5 w-5 text-primary"/>Simulated Signature & PKI Details</AccordionTrigger>
                <AccordionContent className="pt-2 space-y-2">
                  <p className="text-xs text-muted-foreground pb-2">
                    This section demonstrates how an agent's information could be cryptographically signed within a PKI framework. In a real system, this signature would be verifiable against the agent's public key, which itself could be part of a certificate chain leading to a trusted NANDA+ANS CA.
                  </p>
                  <p className="text-sm text-foreground"><strong className="text-muted-foreground"><AlertCircle className="inline h-4 w-4 mr-1.5"/>Signature Type:</strong> {agent.signature.type}</p>
                  <p className="text-sm text-foreground"><strong className="text-muted-foreground"><CalendarDays className="inline h-4 w-4 mr-1.5"/>Signed On:</strong> {new Date(agent.signature.created).toLocaleString()}</p>
                  {agent.signature.simulatedIssuer && (
                    <p className="text-sm text-foreground"><strong className="text-muted-foreground"><Building className="inline h-4 w-4 mr-1.5"/>Simulated Issuer (CA):</strong> {agent.signature.simulatedIssuer}</p>
                  )}
                  <p className="text-sm text-foreground"><strong className="text-muted-foreground"><ShieldQuestion className="inline h-4 w-4 mr-1.5"/>Verification Method (Key ID):</strong> {agent.signature.verificationMethod}</p>
                   {agent.signature.simulatedPublicKey && (
                    <p className="text-sm text-foreground break-all"><strong className="text-muted-foreground"><KeyRound className="inline h-4 w-4 mr-1.5"/>Simulated Public Key:</strong> {agent.signature.simulatedPublicKey}</p>
                  )}
                  <p className="text-sm text-foreground"><strong className="text-muted-foreground"><CheckCircle2 className="inline h-4 w-4 mr-1.5"/>Proof Purpose:</strong> {agent.signature.proofPurpose}</p>
                  <p className="text-sm text-foreground break-all"><strong className="text-muted-foreground"><Fingerprint className="inline h-4 w-4 mr-1.5"/>Simulated Signature Value:</strong> {agent.signature.proofValue}</p>
                </AccordionContent>
              </AccordionItem>
            )}

          </Accordion>
        </CardContent>
      </Card>
    </div>
  );
};

export default AgentProfileDetails;
