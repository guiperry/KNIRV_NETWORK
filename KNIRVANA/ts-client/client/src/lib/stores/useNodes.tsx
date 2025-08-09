import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import { ErrorNode, SkillNode } from '@/types/game';

interface NodeState {
  selectedNodeId: string | null;
  hoveredNodeId: string | null;
  
  // Actions
  selectNode: (nodeId: string | null) => void;
  setHoveredNode: (nodeId: string | null) => void;
  updateErrorNodeProgress: (nodeId: string, progress: number) => void;
  setNodeBeingResolved: (nodeId: string, isBeingResolved: boolean, resolvedBy?: string) => void;
}

export const useNodes = create<NodeState>()(
  subscribeWithSelector((set) => ({
    selectedNodeId: null,
    hoveredNodeId: null,

    selectNode: (nodeId) => {
      set({ selectedNodeId: nodeId });
    },

    setHoveredNode: (nodeId) => {
      set({ hoveredNodeId: nodeId });
    },

    updateErrorNodeProgress: (nodeId, progress) => {
      // This would typically update the progress in the main game state
      // For now, we'll handle this through the main Knirvana store
    },

    setNodeBeingResolved: (nodeId, isBeingResolved, resolvedBy) => {
      // This would typically update the node resolution state
      // For now, we'll handle this through the main Knirvana store
    }
  }))
);
