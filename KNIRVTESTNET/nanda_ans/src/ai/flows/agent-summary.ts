'use server';

/**
 * @fileOverview Provides a Genkit flow for generating a concise AI summary of an agent's functionality.
 *
 * - generateAgentSummary - Generates a short summary of the agent's capabilities.
 * - AgentSummaryInput - The input type for the generateAgentSummary function.
 * - AgentSummaryOutput - The return type for the generateAgentSummary function.
 */

import {ai} from '@/ai/genkit';
import {z} from 'genkit';

const AgentSummaryInputSchema = z.object({
  agentName: z.string().describe('The name of the agent.'),
  agentCapabilities: z.string().describe('A description of the agent\u0027s capabilities.'),
  agentDescription: z.string().optional().describe('A detailed description of the agent.'),
});
export type AgentSummaryInput = z.infer<typeof AgentSummaryInputSchema>;

const AgentSummaryOutputSchema = z.object({
  summary: z.string().describe('A concise AI-generated summary of the agent\u0027s functionality.'),
});
export type AgentSummaryOutput = z.infer<typeof AgentSummaryOutputSchema>;

export async function generateAgentSummary(input: AgentSummaryInput): Promise<AgentSummaryOutput> {
  return agentSummaryFlow(input);
}

const agentSummaryPrompt = ai.definePrompt({
  name: 'agentSummaryPrompt',
  input: {schema: AgentSummaryInputSchema},
  output: {schema: AgentSummaryOutputSchema},
  prompt: `You are an AI agent specializing in summarizing the functionality of other AI agents.

  Given the following information about an agent, create a concise summary (1-2 sentences) that new users can use to quickly understand its purpose and capabilities:

  Agent Name: {{{agentName}}}
  Agent Capabilities: {{{agentCapabilities}}}
  {{#if agentDescription}}
  Agent Description: {{{agentDescription}}}
  {{/if}}

  Summary:
  `,
});

const agentSummaryFlow = ai.defineFlow(
  {
    name: 'agentSummaryFlow',
    inputSchema: AgentSummaryInputSchema,
    outputSchema: AgentSummaryOutputSchema,
  },
  async input => {
    const {output} = await agentSummaryPrompt(input);
    return output!;
  }
);
