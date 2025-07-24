// Blockchain API Types
export interface Block {
  header: {
    height: number;
    previous_hash: string;
    timestamp: string;
    merkle_root: string;
    validator_set: string[];
    proposer: string;
    state_root: string;
  };
  transactions: Transaction[];
  hash: string;
}

export interface Transaction {
  id: string;
  from: string;
  to: string;
  amount: number;
  fee: number;
  data: string;
  timestamp: string;
  signature: string;
  nonce: number;
}

export interface Account {
  address: string;
  balance: number;
  nonce: number;
}

export interface BlockchainStats {
  height: number;
  totalTransactions: number;
  totalAccounts: number;
  avgBlockTime: number;
}

// API Configuration
const API_BASE_URL = 'http://localhost:8080';

class BlockchainAPI {
  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
        ...options,
      });

      if (!response.ok) {
        throw new Error(`API Error: ${response.status} ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Network error');
    }
  }

  async getCurrentHeight(): Promise<number> {
    const response = await this.request<{ height: number }>('/height');
    return response.height;
  }

  async getBlock(height: number): Promise<Block> {
    return await this.request<Block>(`/block/${height}`);
  }

  async getAccount(address: string): Promise<Account> {
    return await this.request<Account>(`/account/${address}`);
  }

  async submitTransaction(transaction: Transaction): Promise<{ status: string; tx_id: string }> {
    return await this.request<{ status: string; tx_id: string }>('/transaction', {
      method: 'POST',
      body: JSON.stringify(transaction),
    });
  }

  async getRecentBlocks(count: number = 10): Promise<Block[]> {
    const height = await this.getCurrentHeight();
    const promises = [];
    
    for (let i = Math.max(0, height - count + 1); i <= height; i++) {
      promises.push(this.getBlock(i));
    }
    
    const blocks = await Promise.allSettled(promises);
    return blocks
      .filter((result): result is PromiseFulfilledResult<Block> => result.status === 'fulfilled')
      .map(result => result.value)
      .reverse(); // Most recent first
  }

  async getBlockchainStats(): Promise<BlockchainStats> {
    try {
      const height = await this.getCurrentHeight();
      const recentBlocks = await this.getRecentBlocks(10);
      
      let totalTransactions = 0;
      let totalTime = 0;
      
      recentBlocks.forEach((block, index) => {
        totalTransactions += block.transactions.length;
        if (index > 0) {
          const currentTime = new Date(block.header.timestamp).getTime();
          const prevTime = new Date(recentBlocks[index - 1].header.timestamp).getTime();
          totalTime += currentTime - prevTime;
        }
      });
      
      const avgBlockTime = recentBlocks.length > 1 ? totalTime / (recentBlocks.length - 1) / 1000 : 0;
      
      return {
        height,
        totalTransactions,
        totalAccounts: 0, // This would require additional API endpoint
        avgBlockTime,
      };
    } catch (error) {
      throw new Error('Failed to fetch blockchain stats');
    }
  }
}

export const blockchainApi = new BlockchainAPI();