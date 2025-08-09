import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import { AIAgent } from '@/types/game';
import { createAIAgent, moveAgentToTarget } from '@/lib/gameUtils';
import * as THREE from 'three';

interface AgentState {
  agents: AIAgent[];
  selectedAgentId: string | null;
  
  // Actions
  spawnAgent: (owner: string, position?: THREE.Vector3) => void;
  removeAgent: (agentId: string) => void;
  selectAgent: (agentId: string | null) => void;
  moveAgent: (agentId: string, targetPosition: THREE.Vector3) => void;
  updateAgentPosition: (agentId: string, position: THREE.Vector3) => void;
  setAgentTask: (agentId: string, task: string) => void;
  updateAgentEnergy: (agentId: string, energy: number) => void;
  updateAgentThoughts: (agentId: string, thoughts: string[]) => void;
  updateAllAgents: (deltaTime: number) => void;
}

export const useAgents = create<AgentState>()(
  subscribeWithSelector((set, get) => ({
    agents: [],
    selectedAgentId: null,

    spawnAgent: (owner, position) => {
      set((state) => {
        const agentId = `agent-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
        const newAgent = createAIAgent(agentId, owner);
        
        if (position) {
          newAgent.position = position.clone();
          newAgent.targetPosition = position.clone();
        }

        return {
          agents: [...state.agents, newAgent]
        };
      });
    },

    removeAgent: (agentId) => {
      set((state) => ({
        agents: state.agents.filter(a => a.id !== agentId),
        selectedAgentId: state.selectedAgentId === agentId ? null : state.selectedAgentId
      }));
    },

    selectAgent: (agentId) => {
      set({ selectedAgentId: agentId });
    },

    moveAgent: (agentId, targetPosition) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, targetPosition: targetPosition.clone() }
            : agent
        )
      }));
    },

    updateAgentPosition: (agentId, position) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, position: position.clone() }
            : agent
        )
      }));
    },

    setAgentTask: (agentId, task) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, currentTask: task }
            : agent
        )
      }));
    },

    updateAgentEnergy: (agentId, energy) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, energy: Math.max(0, Math.min(agent.maxEnergy, energy)) }
            : agent
        )
      }));
    },

    updateAgentThoughts: (agentId, thoughts) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, thoughtProcess: thoughts }
            : agent
        )
      }));
    },

    updateAllAgents: (deltaTime) => {
      set((state) => ({
        agents: state.agents.map(agent => {
          // Move agent towards target
          moveAgentToTarget(agent, deltaTime);
          
          // Update energy (slowly regenerate)
          const newEnergy = Math.min(agent.maxEnergy, agent.energy + deltaTime * 2);
          
          // Update thought process periodically
          const shouldUpdateThoughts = Math.random() < 0.01; // 1% chance per frame
          let newThoughts = agent.thoughtProcess;
          
          if (shouldUpdateThoughts && agent.currentTask) {
            const possibleThoughts = [
              `Analyzing ${agent.currentTask}...`,
              `Computing optimal solution for ${agent.currentTask}...`,
              `Applying ${agent.specialization} knowledge...`,
              `Progress: ${Math.floor(Math.random() * 100)}%`,
              'Consulting KNIRVGRAPH database...',
              'Optimizing approach...',
              'Validating solution...'
            ];
            
            newThoughts = [
              possibleThoughts[Math.floor(Math.random() * possibleThoughts.length)],
              ...agent.thoughtProcess.slice(0, 2)
            ];
          }
          
          return {
            ...agent,
            energy: newEnergy,
            thoughtProcess: newThoughts
          };
        })
      }));
    }
  }))
);
