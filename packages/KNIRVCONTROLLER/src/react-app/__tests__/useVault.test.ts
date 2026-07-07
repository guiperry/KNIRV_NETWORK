import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useVault } from '@/react-app/hooks/useVault';

const { mockGenerateMnemonic, mockSerialize, mockDeserialize } = vi.hoisted(() => {
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

  return {
    mockGenerateMnemonic,
    mockSerialize,
    mockDeserialize,
  };
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

beforeEach(() => {
  localStorage.clear();
});

async function flushMicrotasks() {
  await act(async () => {
    await new Promise(r => setTimeout(r, 0));
  });
}

describe('useVault', () => {
  it('initial vault status resolves to "no_vault" after async check', async () => {
    const { result } = renderHook(() => useVault());
    expect(result.current.status).toBe('initializing');
    await flushMicrotasks();
    expect(result.current.status).toBe('no_vault');
  });

  it('transitions to "no_vault" when no stored vault exists', async () => {
    const { result } = renderHook(() => useVault());
    await flushMicrotasks();
    expect(result.current.status).toBe('no_vault');
  });

  it('transitions to "locked" when stored vault exists', async () => {
    localStorage.setItem('knirv_vault', 'some_serialized_vault');

    const { result } = renderHook(() => useVault());
    await flushMicrotasks();

    expect(result.current.status).toBe('locked');
  });

  it('creates a vault and transitions to unlocked', async () => {
    const { result } = renderHook(() => useVault());

    await flushMicrotasks();
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

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    act(() => {
      result.current.lockVault();
    });

    expect(result.current.status).toBe('locked');
  });

  it('unlocks a locked vault', async () => {
    localStorage.setItem('knirv_vault', 'serialized:test-password');

    const { result } = renderHook(() => useVault());
    await flushMicrotasks();

    expect(result.current.status).toBe('locked');

    await act(async () => {
      await result.current.unlockVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');
  });

  it('derives a DVE vault from unlocked vault', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    let dveVault: any;
    await act(async () => {
      dveVault = await result.current.deriveDVEVault('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    expect(dveVault).toBeDefined();
    expect(dveVault.dveID).toBe('DVE-001');
    expect(dveVault.dveName).toBe('DVE-Alpha');
    expect(dveVault.teeType).toBe('sgx');
    expect(dveVault.derivationIndex).toBe(0);
    expect(dveVault.vaultAddress).toContain('DVE-001');
    expect(result.current.dveVaults).toHaveLength(1);
  });

  it('fails to derive DVE vault when vault is locked', async () => {
    const { result } = renderHook(() => useVault());

    localStorage.setItem('knirv_vault', 'serialized:test-password');
    await flushMicrotasks();

    await expect(async () => {
      await act(async () => {
        await result.current.deriveDVEVault('DVE-001', 'DVE-Alpha', 'sgx', 0);
      });
    }).rejects.toThrow('Vault is locked or not initialized');
  });

  it('clears the vault and transitions to no_vault', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    expect(result.current.status).toBe('unlocked');

    await act(async () => {
      await result.current.clearVault();
    });

    expect(result.current.status).toBe('no_vault');
    expect(localStorage.getItem('knirv_vault')).toBeNull();
  });

  it('updates DVE vault status', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    await act(async () => {
      await result.current.deriveDVEVault('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    act(() => {
      result.current.updateDVEVaultStatus('DVE-001', { status: 'online', stakeAmount: 5000 });
    });

    const updatedVault = result.current.dveVaults.find((w: any) => w.dveID === 'DVE-001');
    expect(updatedVault?.status).toBe('online');
    expect(updatedVault?.stakeAmount).toBe(5000);
  });

  it('removes a DVE vault', async () => {
    const { result } = renderHook(() => useVault());

    await act(async () => {
      await result.current.createVault('test-password');
    });

    await act(async () => {
      await result.current.deriveDVEVault('DVE-001', 'DVE-Alpha', 'sgx', 0);
    });

    expect(result.current.dveVaults).toHaveLength(1);

    act(() => {
      result.current.removeDVEVault('DVE-001');
    });

    expect(result.current.dveVaults).toHaveLength(0);
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
