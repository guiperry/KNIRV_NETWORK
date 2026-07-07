// Global type declarations for test utilities

interface MockGraphData {
  nodes: Array<{ id: string; label: string; type: string }>;
  edges: Array<{ source: string; target: string; type: string }>;
}

interface MockBlockchainData {
  blocks: Array<{ id: string; hash: string; timestamp: number; transactions: number }>;
  transactions: Array<{ id: string; from: string; to: string; amount: number }>;
}

interface MockNRVData {
  vectors: Array<{ id: string; coordinates: number[]; confidence: number }>;
}

declare global {
  var testUtils: {
    createMockGraphData: () => MockGraphData;
    createMockBlockchainData: () => MockBlockchainData;
    createMockNRVData: () => MockNRVData;
    waitFor: (condition: () => boolean, timeout?: number) => Promise<void>;
    mockComponent: (name: string) => React.ComponentType<{ children?: React.ReactNode }>;
  };

  namespace NodeJS {
    interface Global {
      testUtils: {
        createMockGraphData: () => MockGraphData;
        createMockBlockchainData: () => MockBlockchainData;
        createMockNRVData: () => MockNRVData;
        waitFor: (condition: () => boolean, timeout?: number) => Promise<void>;
        mockComponent: (name: string) => React.ComponentType<{ children?: React.ReactNode }>;
      };
    }
  }
}

export {};
