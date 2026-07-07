import { decryptAES, encryptAES } from '../../../common/utils/crypto';
import { StorageMigration002 } from './storage-migration-v002';

const makeMockStorageData = (serialized = 'serialized-wallet') => ({
  NETWORKS: [],
  CURRENT_CHAIN_ID: '',
  CURRENT_NETWORK_ID: '',
  SERIALIZED: serialized,
  ENCRYPTED_STORED_PASSWORD: '',
  CURRENT_ACCOUNT_ID: '',
  ACCOUNT_NAMES: {},
  ESTABLISH_SITES: {},
  ADDRESS_BOOK: {},
  ACCOUNT_TOKEN_METAINFOS: {},
});

describe('serialized wallet migration V002', () => {
  it('version', () => {
    const migration = new StorageMigration002();
    expect(migration.version).toBe(2);
  });

  it('up success', async () => {
    const mockData = {
      version: 1,
      data: makeMockStorageData(),
    };
    const migration = new StorageMigration002();
    const result = await migration.up(mockData);

    expect(result.version).toBe(2);
    expect(result.data).not.toBeNull();
    expect(result.data.NETWORKS).toEqual([]);
    expect(result.data.CURRENT_CHAIN_ID).toBe('');
    expect(result.data.CURRENT_NETWORK_ID).toBe('');
    expect(result.data.SERIALIZED).toBe(
      'serialized-wallet',
    );
    expect(result.data.ENCRYPTED_STORED_PASSWORD).toBe('');
    expect(result.data.CURRENT_ACCOUNT_ID).toBe('');
    expect(result.data.ACCOUNT_NAMES).toEqual({});
    expect(result.data.ESTABLISH_SITES).toEqual({});
    expect(result.data.ADDRESS_BOOK).toEqual([]);
  });

  it('up password success', async () => {
    const password = '123';
    const walletFixture = {
      accounts: [],
      keyrings: [],
    };
    const serialized = await encryptAES(JSON.stringify(walletFixture), password);
    const mockData = {
      version: 1,
      data: makeMockStorageData(serialized),
    };
    const migration = new StorageMigration002();
    const result = await migration.up(mockData, password);

    expect(result.version).toBe(2);
    expect(result.data).not.toBeNull();
    expect(result.data.NETWORKS).toEqual([]);
    expect(result.data.CURRENT_CHAIN_ID).toBe('');
    expect(result.data.CURRENT_NETWORK_ID).toBe('');
    expect(result.data.SERIALIZED).not.toBe('');
    expect(result.data.ENCRYPTED_STORED_PASSWORD).toBe('');
    expect(result.data.CURRENT_ACCOUNT_ID).toBe('');
    expect(result.data.ACCOUNT_NAMES).toEqual({});
    expect(result.data.ESTABLISH_SITES).toEqual({});
    expect(result.data.ADDRESS_BOOK).toEqual([]);

    const resultSerialized = result.data.SERIALIZED;
    const decrypted = await decryptAES(resultSerialized, password);
    const wallet = JSON.parse(decrypted);

    expect(wallet.accounts).toHaveLength(0);
    expect(wallet.keyrings).toHaveLength(0);
  });

  it('up failed throw error', async () => {
    const mockData: any = {
      version: 1,
      data: { ...makeMockStorageData(), SERIALIZED: null },
    };
    const migration = new StorageMigration002();

    await expect(migration.up(mockData)).rejects.toThrow(
      'Storage Data does not match version V001',
    );
  });
});
