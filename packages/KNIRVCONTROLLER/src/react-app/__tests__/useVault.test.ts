import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useVault } from '@/react-app/hooks/useVault';

// Mock the knirvwallet-module
const mockGenerateMnemonic = vi.fn(() => 'mock mnemonic twelve words seed phrase here');
const mockSerialize = vi.fn((password: string) => `serialized:${password}`);
const mockDeserialize = vi.fn(async (serialized: string, password: string) => {
  if (serialized === `serialized:${password}`) {
    return {
      accounts: [{ address: '0xabc123', getAddress: () => 'knirv1abc123' }],
      currentAccount: { address: '0xabc123', getAddress: () => 'knirv1abc123' },
      signTransaction: vi.fn(() => Promise.resolve({ hash: '0xtxhash' })),
    };
  }
  throw new Error('Invalid password');
});

vi.mock('knirvwallet-module/wallet', () => ({
  KnirvWallet: {
    generateMnemonic: mockGenerateMnemonic,
    createByMnemonic: vi.fn(async (mnemonic: string) => ({
      mnemonic,
      serialize: mockSerialize,
      accounts: [{ address: '0xabc123', getAddress: () => 'knirv1abc123' }],
      currentAccount: { address: '0xabc123', getAddress: () => 'knirv1abc123' },
      signTransaction: vi.fn(() => Promise.resolve({ hash: '0xtxhash' })),
    })),
    deserialize: mockDeserialize,
  },
}));

vi.mock('knirvwallet-module/wallet/account', () => ({
  Account: class MockAccount {
    address: string;
    constructor(address: string) {
      this.address = address;
    }
    getAddress() {
      return this.address;
    }
  },
}));

// Setup localStorage mock
beforeEach(() => {
  localStorage.clear();
});

describe('useVault', () => {
  it('initial vault status is "initializing"', () => {
    const { result } = renderHook(() => useVault());
    expect(result.current.status).toBe('initializing');
  });

  it('transitions to "no_vault" when no stored vault exists', () => {
    const { result } = renderHook(() => useVault());

    // Wait for the useEffect to run
    act(() => {
      // This triggers the useEffect check
    });

    expect(result.current.status).toBe('no_vault');
  });

  it('transitions to "locked" when stored vault exists', () => {
    localStorage.setItem('knirv_vault', 'some_serialized_vault');

    const { result } = renderHook(() => useVault());

    act(() => {
      // Trigger useEffect
    });

    expect(result.current.status).toBe('locked');
  });

  it('creates a vault and transitions to unlocked', async () => {
    const { result } = renderHook(() => useVault());

    act(() => {
      // Trigger initial useEffect
    });

    expect(result.current.status).toBe('no_vault');

    let mnemonic = '';
    await act(async () => {
      mnemonic = await result.current.createVault('test-password');
    });

    expect(mnemonic).toBeTruthy();
    expect(result.current.status).toBe('unlocked');
  });

  it('locks the vault and transitions to locked', async () => {
    const { result } = renderHook(() => useVault());

    // First create a vault
    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    // Now lock it
    act(() => {
      result.current.lockVault();
    });

    expect(result.current.status).toBe('locked');
  });

  it('unlocks a locked vault', async () => {
    localStorage.setItem('knirv_vault', 'serialized:test-password');

    const { result } = renderHook(() => useVault());

    act(() => {
      // Initial useEffect
    });

    // Should be locked since we set storage
    await act(async () => {
      // Wait for useEffect to fire
      await new Promise(r => setTimeout(r, 0));
    });
    expect(result.current.status).toBe('locked');

    // Unlock with correct password
    await act(async () => {
      await result.current.unlockVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');
  });

  it('derives a DVE wallet from unlocked vault', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    let dveWallet: any;
    await act(async () => {
      dveWallet = await result.current.deriveDVEWallet('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    expect(dveWallet).toBeDefined();
    expect(dveWallet.dveID).toBe('DVE-001');
    expect(dveWallet.dveName).toBe('DVE-Alpha');
    expect(dveWallet.teeType).toBe('sgx');
    expect(dveWallet.derivationIndex).toBe(0);
    expect(dveWallet.walletAddress).toContain('DVE-001');
    expect(result.current.dveWallets).toHaveLength(1);
  });

  it('fails to derive DVE wallet when vault is locked', async () => {
    const { result } = renderHook(() => useVault());

    // Set up locked state
    localStorage.setItem('knirv_vault', 'serialized:test-password');

    act(() => {
      // Let initial useEffect run
    });

    await expect(async () => {
      await act(async () => {
        await result.current.deriveDVEWallet('DVE-001', 'DVE-Alpha', 'sgx', 0);
      });
    }).rejects.toThrow('Vault is locked or not initialized');
  });

  it('clears the vault and transitions to no_vault', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    act(() => {
      result.current.clearVault();
    });

    expect(result.current.status).toBe('no_vault');
    expect(localStorage.getItem('knirv_vault')).toBeNull();
  });

  it('updates DVE wallet status', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    await act(async () => {
      await result.current.deriveDVEWallet('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    act(() => {
      result.current.updateDVEWalletStatus('DVE-001', { status: 'online', stakeAmount: 5000 });
    });

    const updatedWallet = result.current.dveWallets.find(w => w.dveID === 'DVE-001');
    expect(updatedWallet?.status).toBe('online');
    expect(updatedWallet?.stakeAmount).toBe(5000);
  });

  it('removes a DVE wallet', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    await act(async () => {
      await result.current.deriveDVEWallet('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    expect(result.current.dveWallets).toHaveLength(1);

    act(() => {
      result.current.removeDVEWallet('DVE-001');
    });

    expect(result.current.dveWallets).toHaveLength(0);
  });

  it('exposes currentAccount after unlock', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.currentAccount).toBeTruthy();
    expect(result.current.currentAccount?.getAddress('knirv')).toBe('knirv1abc123');
  });
});
