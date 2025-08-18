
// src/app/register/page.tsx
"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle }  from "@/components/ui/card";
import { toast } from "@/hooks/use-toast";
import { BotMessageSquare, ShieldPlus, Loader2 } from "lucide-react";
import { registerNewAgent } from '@/lib/agent-service';
import type { Agent, AgentSignature } from '@/lib/types';
import { useState } from 'react';

const generateMockSignature = (agentId: string): AgentSignature => {
  const creationDate = new Date().toISOString();
  return {
    type: "RsaSignature2018", // PKI-relevant signature type
    created: creationDate,
    verificationMethod: `${agentId}#key-pki-1`, // Mock key identifier linked to the agent's DID
    proofPurpose: "assertionMethod", // Purpose of the signature (e.g., agent asserts its identity/facts)
    proofValue: Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''), // 64-byte hex string
    simulatedIssuer: "NANDA+ANS Global CA G2", // Mock Certificate Authority
    simulatedPublicKey: `04:${Array(64).fill(0).map(() => Math.floor(Math.random() * 256).toString(16).padStart(2, '0')).join(':').toUpperCase()}`, // Mock public key in hex format
  };
};

const realisticDefaults = {
  agentName: "My New AI Agent",
  agentDID: `did:nanda:agent-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`, // More unique default DID
  capability: "Generic AI Task",
  description: "This is a newly registered AI agent. Its purpose is to perform general tasks and integrate within the NANDA+ANS ecosystem. Specific capabilities and details will be updated in its AgentFacts. This registration includes a simulated digital signature to demonstrate PKI concepts.",
  factsUrl: "https://example.com/.well-known/agent-facts-default.jsonld",
  providerName: "Independent Developer",
  version: "0.1.0-alpha",
  ansName: "", // Optional, can be derived or set later
  capabilities: ["Generic AI Task"], // Default capability array
  attestations: [],
  avatarUrl: 'https://placehold.co/100x100.png',
  dataAiHint: 'agent avatar',
  addr_ttl: 3600,
  signature: undefined as AgentSignature | undefined, // Will be populated in onSubmit or if defaults are used
};
// Initialize signature with a default mock signature based on the default DID
realisticDefaults.signature = generateMockSignature(realisticDefaults.agentDID);


const agentRegistrationSchema = z.object({
  agentName: z.string().min(3, { message: "Agent name must be at least 3 characters if provided." }).optional().or(z.literal("")),
  agentDID: z.string().startsWith("did:", { message: "Must be a valid DID (e.g., did:nanda:xyz) if provided." }).optional().or(z.literal("")),
  capability: z.string().min(3, { message: "Primary capability must be at least 3 characters if provided." }).optional().or(z.literal("")),
  description: z.string()
    .min(10, { message: "Description must be at least 10 characters if provided." })
    .max(500, { message: "Description must not exceed 500 characters if provided." })
    .optional().or(z.literal("")),
  factsUrl: z.string().url({ message: "Please enter a valid URL for AgentFacts if provided." }).optional().or(z.literal("")),
  providerName: z.string().optional().or(z.literal("")),
  version: z.string().optional().or(z.literal("")),
  ansName: z.string().optional().or(z.literal("")), // Added ansName
});

type AgentRegistrationFormValues = z.infer<typeof agentRegistrationSchema>;

export default function RegisterAgentPage() {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<AgentRegistrationFormValues>({
    resolver: zodResolver(agentRegistrationSchema),
    defaultValues: { 
      agentName: realisticDefaults.agentName,
      agentDID: realisticDefaults.agentDID,
      capability: realisticDefaults.capability,
      description: realisticDefaults.description,
      factsUrl: realisticDefaults.factsUrl,
      providerName: realisticDefaults.providerName,
      version: realisticDefaults.version,
      ansName: realisticDefaults.ansName,
    },
    mode: "onChange",
  });

  async function onSubmit(data: AgentRegistrationFormValues) {
    setIsSubmitting(true);

    try {
      const agentId = data.agentDID?.trim() || realisticDefaults.agentDID;

      const agentData: Partial<Agent> = {
        id: agentId,
        name: data.agentName?.trim() || realisticDefaults.agentName,
        ansName: data.ansName?.trim() || `${data.providerName?.trim() || realisticDefaults.providerName}.${data.capability?.trim() || realisticDefaults.capability}.${agentId.split(':').pop()}`.toLowerCase().replace(/\s+/g, ''),
        capability: data.capability?.trim() || realisticDefaults.capability,
        capabilities: data.capability?.trim() ? [data.capability.trim()] : realisticDefaults.capabilities,
        provider: data.providerName?.trim() || realisticDefaults.providerName,
        version: data.version?.trim() || realisticDefaults.version,
        description: data.description?.trim() || realisticDefaults.description,
        addr_facts_url: data.factsUrl?.trim() || realisticDefaults.factsUrl,
        attestations: realisticDefaults.attestations,
        avatarUrl: realisticDefaults.avatarUrl,
        dataAiHint: data.agentName?.trim() ? data.agentName.trim().toLowerCase().split(' ').slice(0,2).join(' ') : realisticDefaults.dataAiHint,
        addr_ttl: realisticDefaults.addr_ttl,
      };

      console.log("Registering agent with KNIRV-ORACLE integration...");

      // Register agent with KNIRV-ORACLE integration
      const result = await registerNewAgent(agentData);

      console.log("Agent registered successfully:", result);

      toast({
        title: "Agent Registered Successfully! 🎉",
        description: (
          <div className="mt-2 w-[340px] rounded-md bg-green-950 p-4">
            <p className="text-green-100 font-medium">Your agent has been registered with KNIRV-ORACLE!</p>
            <div className="mt-2 space-y-1 text-sm text-green-200">
              <p><strong>Agent ID:</strong> {result.agent.id}</p>
              <p><strong>Transaction Hash:</strong> {result.transactionHash}</p>
              <p><strong>Capability ID:</strong> {result.capabilityId}</p>
              <p><strong>Status:</strong> {result.agent.oracleStatus}</p>
            </div>
            <p className="mt-2 text-xs text-green-300">
              The agent is now registered in the NANDA ANS registry and recorded on KNIRV-ORACLE.
            </p>
          </div>
        ),
        variant: "default",
        duration: 15000,
      });
    } catch (error) {
      console.error("Error registering agent:", error);

      toast({
        title: "Registration Failed",
        description: (
          <div className="mt-2 w-[340px] rounded-md bg-red-950 p-4">
            <p className="text-red-100">Failed to register agent with KNIRV-ORACLE.</p>
            <p className="mt-1 text-sm text-red-200">
              {error instanceof Error ? error.message : 'Unknown error occurred'}
            </p>
            <p className="mt-2 text-xs text-red-300">
              Please check that KNIRV-ORACLE is running and try again.
            </p>
          </div>
        ),
        variant: "destructive",
        duration: 10000,
      });
    } finally {
      setIsSubmitting(false);
    }

    const newDefaultDID = `did:nanda:agent-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`;
    form.reset({
      ...realisticDefaults,
      agentDID: newDefaultDID,
      agentName: realisticDefaults.agentName,
      capability: realisticDefaults.capability,
      description: realisticDefaults.description,
      factsUrl: realisticDefaults.factsUrl,
      providerName: realisticDefaults.providerName,
      version: realisticDefaults.version,
      ansName: "",
      // The signature in realisticDefaults itself doesn't need to be updated here,
      // as a new one is generated on each submission.
    });
  }

  return (
    <div className="max-w-2xl mx-auto">
      <Card className="shadow-lg">
        <CardHeader className="text-center">
          <div className="inline-flex items-center justify-center mb-4">
            <ShieldPlus className="h-12 w-12 text-primary" />
          </div>
          <CardTitle className="text-3xl font-bold">Register New Agent</CardTitle>
          <CardDescription className="text-muted-foreground">
            Add your agent to the NANDA+ANS ecosystem. Registration involves creating a NANDA `AgentAddr` (a lightweight, signed pointer) that directs to your detailed `AgentFacts` (verifiable metadata). All fields are optional; if left blank, sensible defaults will be used. This establishes your agent's identity and demonstrates cryptographically assured capabilities through simulated digital signatures. (Currently, this form only demonstrates data preparation and does not save to a database).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="agentName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Agent Name</FormLabel>
                    <FormControl>
                      <Input placeholder={realisticDefaults.agentName} {...field} />
                    </FormControl>
                    <FormDescription>A user-friendly name for your agent. (Optional, defaults will be used if blank)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="agentDID"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Agent DID (Decentralized Identifier)</FormLabel>
                    <FormControl>
                      <Input placeholder={realisticDefaults.agentDID} {...field} />
                    </FormControl>
                    <FormDescription>The agent's unique DID (NANDA root identity). This will be used as the database ID. (Optional, a unique default will be generated)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="capability"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Primary Capability</FormLabel>
                    <FormControl>
                      <Input placeholder={realisticDefaults.capability} {...field} />
                    </FormControl>
                    <FormDescription>The main function your agent provides. (Optional, defaults will be used if blank)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
               <FormField
                control={form.control}
                name="ansName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>ANS Name (Optional)</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., provider.capability.agentid.v1" {...field} />
                    </FormControl>
                    <FormDescription>Agent Name Service (ANS) string for capability-based addressing. (Optional, a default will be constructed if blank)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Description</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder={realisticDefaults.description}
                        className="resize-y min-h-[100px]"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>Detailed description for AgentFacts. (Optional, defaults will be used if blank)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
               <FormField
                control={form.control}
                name="factsUrl"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>AgentFacts URL (NANDA Pointer Target)</FormLabel>
                    <FormControl>
                      <Input type="url" placeholder={realisticDefaults.factsUrl} {...field} />
                    </FormControl>
                    <FormDescription>Publicly accessible URL to your AgentFacts (JSON-LD). (Optional, defaults will be used if blank)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <FormField
                  control={form.control}
                  name="providerName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Provider Name</FormLabel>
                      <FormControl>
                        <Input placeholder={realisticDefaults.providerName} {...field} />
                      </FormControl>
                      <FormDescription>(Optional)</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="version"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Version</FormLabel>
                      <FormControl>
                        <Input placeholder={realisticDefaults.version} {...field} />
                      </FormControl>
                      <FormDescription>(Optional)</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <Button type="submit" className="w-full" size="lg" disabled={isSubmitting}>
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Registering with KNIRV-ORACLE...
                  </>
                ) : (
                  <>
                    <ShieldPlus className="mr-2 h-4 w-4" />
                    Register Agent with KNIRV-ORACLE
                  </>
                )}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}
