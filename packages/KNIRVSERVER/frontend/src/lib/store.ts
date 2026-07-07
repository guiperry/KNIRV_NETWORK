import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

interface DashboardState {
  activeTab: string;
  setActiveTab: (tab: string) => void;

  isSidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (isOpen: boolean) => void;

  // Selected entities
  selectedNodeId: string | null;
  setSelectedNodeId: (id: string | null) => void;

  selectedTaskId: string | null;
  setSelectedTaskId: (id: string | null) => void;

  // Persist active DVE node state across navigation
  activeNodeIds: { [key: string]: boolean };
  setActiveNodeId: (nodeId: string, isActive: boolean) => void;
  clearActiveNodeIds: () => void;

  // Modal states
  modals: {
    rentDve: boolean;
    sshTerminal: boolean;
    validationExplorer: boolean;
  };
  setModalOpen: (modal: keyof DashboardState['modals'], isOpen: boolean) => void;
}

export const useDashboardStore = create<DashboardState>()(
  devtools(
    persist(
      (set) => ({
        activeTab: 'fabric',
        setActiveTab: (tab) => set({ activeTab: tab }),

        isSidebarOpen: true,
        toggleSidebar: () => set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),
        setSidebarOpen: (isOpen) => set({ isSidebarOpen: isOpen }),

        selectedNodeId: null,
        setSelectedNodeId: (id) => set({ selectedNodeId: id }),

        selectedTaskId: null,
        setSelectedTaskId: (id) => set({ selectedTaskId: id }),

        activeNodeIds: {},
        setActiveNodeId: (nodeId, isActive) => set((state) => ({
          activeNodeIds: { ...state.activeNodeIds, [nodeId]: isActive }
        })),
        clearActiveNodeIds: () => set({ activeNodeIds: {} }),

        modals: {
          rentDve: false,
          sshTerminal: false,
          validationExplorer: false,
        },
        setModalOpen: (modal, isOpen) => set((state) => ({
          modals: { ...state.modals, [modal]: isOpen }
        })),
      }),
      {
        name: 'knirv-dashboard-storage',
        partialize: (state) => ({
          activeTab: state.activeTab,
          isSidebarOpen: state.isSidebarOpen,
          activeNodeIds: state.activeNodeIds,
        }),
      }
    )
  )
);
