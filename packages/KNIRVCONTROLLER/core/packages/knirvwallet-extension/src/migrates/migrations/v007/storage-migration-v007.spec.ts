import { decryptAES, encryptAES } from '../../../common/utils/crypto';
import { StorageMigration007 } from './storage-migration-v007';

const makeMockStorageData = (serialized = 'serialized-wallet') => ({
  NETWORKS: [],
  CURRENT_CHAIN_ID: 'test3',
  CURRENT_NETWORK_ID: 'test3',
  SERIALIZED: serialized,
  ENCRYPTED_STORED_PASSWORD: '',
  CURRENT_ACCOUNT_ID: '',
  ACCOUNT_NAMES: {},
  ESTABLISH_SITES: {},
  ADDRESS_BOOK: '',
  ACCOUNT_TOKEN_METAINFOS: {},
  QUESTIONNAIRE_EXPIRED_DATE: null,
  WALLET_CREATION_GUIDE_CONFIRM_DATE: null,
  ADD_ACCOUNT_GUIDE_CONFIRM_DATE: null,
});

let mockStorageData = makeMockStorageData();

describe('serialized wallet migration V007', () => {
  beforeAll(async () => {
    const serialized = await encryptAES(JSON.stringify({ accounts: [], keyrings: [] }), '123');
    mockStorageData = makeMockStorageData(serialized);
  });

  it('version', () => {
    const migration = new StorageMigration007();
    expect(migration.version).toBe(7);
  });

  it('up success', async () => {
    const mockData = {
      version: 6,
      data: mockStorageData,
    };
    const migration = new StorageMigration007();
    const result = await migration.up(mockData);

    expect(result.data.ACCOUNT_GRC721_COLLECTIONS).toEqual({});
    expect(result.data.ACCOUNT_GRC721_PINNED_PACKAGES).toEqual({});
  });

  it('up password success', async () => {
    const mockData = {
      version: 1,
      data: mockStorageData,
    };
    const password = '123';
    const migration = new StorageMigration007();
    const result = await migration.up(mockData);

    expect(result.version).toBe(7);
    expect(result.data).not.toBeNull();
    expect(result.data.ACCOUNT_GRC721_COLLECTIONS).toEqual({});
    expect(result.data.ACCOUNT_GRC721_PINNED_PACKAGES).toEqual({});

    const serialized = result.data.SERIALIZED;
    const decrypted = await decryptAES(serialized, password);
    const wallet = JSON.parse(decrypted);

    expect(wallet.accounts).toHaveLength(0);
    expect(wallet.keyrings).toHaveLength(0);
  });

  it('up failed throw error', async () => {
    const mockData: any = {
      version: 1,
      data: { ...mockStorageData, SERIALIZED: null },
    };
    const migration = new StorageMigration007();

    await expect(migration.up(mockData)).rejects.toThrow(
      'Storage Data does not match version V006',
    );
  });
});
