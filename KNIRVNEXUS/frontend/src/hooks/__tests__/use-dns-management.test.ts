import { renderHook, act, waitFor } from '@testing-library/react';
import { useDNSManagement } from '../use-dns-management';
import type { DNSRecord, DNSZone, DNSStatus, CreateDNSRecordRequest, UpdateDNSRecordRequest, APIResponse } from '@/types/api';
import { apiRequest } from '@/lib/api';

// Mock the API module
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8082',
}));

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

// Helper function to create mock API responses with timestamp
const createMockResponse = <T>(data: T, success: boolean = true, error?: string): APIResponse<T> => ({
  success,
  data,
  timestamp: new Date().toISOString(),
  ...(error && { error })
});

const mockDNSRecords: DNSRecord[] = [
  {
    id: 'record-1',
    name: 'test.example.com',
    type: 'A',
    value: '192.168.1.1',
    ttl: 300,
    zone: 'example.com',
    proxied: false,
    priority: 0,
    comment: 'Test A record',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  },
  {
    id: 'record-2',
    name: 'mail.example.com',
    type: 'MX',
    value: 'mail.example.com',
    ttl: 3600,
    zone: 'example.com',
    proxied: false,
    priority: 10,
    comment: 'Mail server record',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z'
  },
  {
    id: 'record-3',
    name: 'api.test.com',
    type: 'CNAME',
    value: 'api-server.test.com',
    ttl: 1800,
    zone: 'test.com',
    proxied: true,
    priority: 0,
    comment: 'API endpoint',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z'
  }
];

const mockDNSZones: DNSZone[] = [
  {
    id: 'zone-1',
    name: 'example.com',
    type: 'primary',
    status: 'active',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  },
  {
    id: 'zone-2',
    name: 'test.com',
    type: 'primary',
    status: 'active',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z'
  },
  {
    id: 'zone-3',
    name: 'inactive.com',
    type: 'primary',
    status: 'inactive',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z'
  }
];

const mockDNSStatus: DNSStatus = {
  service: 'dns',
  status: 'running',
  zones: 5,
  records: 25,
  timestamp: '2024-01-01T00:00:00Z'
};

describe('useDNSManagement Hook', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockApiRequest.mockClear();
    // Set up default mock responses to prevent errors during hook initialization
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/status')) {
        return Promise.resolve(createMockResponse({ total_records: 0, active_zones: 0, health_status: 'healthy', last_updated: new Date().toISOString() }));
      }
      return Promise.resolve(createMockResponse([]));
    });
  });

  it('initializes with default state', async () => {
    const { result } = renderHook(() => useDNSManagement());

    expect(result.current.records).toEqual([]);
    expect(result.current.zones).toEqual([]);
    expect(result.current.status).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.isLoading).toBe(false);
  });

  it('fetches DNS records successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockDNSRecords));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchRecords();
    });

    expect(result.current.records).toEqual(mockDNSRecords);
    expect(result.current.error).toBeNull();
  });

  it('fetches DNS records with zone filter', async () => {
    const filteredRecords = mockDNSRecords.filter(r => r.zone === 'example.com');

    const { result } = renderHook(() => useDNSManagement());

    mockApiRequest.mockResolvedValueOnce(createMockResponse(filteredRecords));

    await act(async () => {
      await result.current.fetchRecords('example.com');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records?zone=example.com', {
      method: 'GET'
    });
    expect(result.current.records).toEqual(filteredRecords);
  });

  it('fetches DNS records with type filter', async () => {
    const filteredRecords = mockDNSRecords.filter(r => r.type === 'A');

    const { result } = renderHook(() => useDNSManagement());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    mockApiRequest.mockResolvedValueOnce(createMockResponse(filteredRecords));

    await act(async () => {
      await result.current.fetchRecords(undefined, 'A');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records?type=A', {
      method: 'GET'
    });
    expect(result.current.records).toEqual(filteredRecords);
  });

  it('fetches DNS records with both zone and type filters', async () => {
    const filteredRecords = mockDNSRecords.filter(r => r.zone === 'example.com' && r.type === 'A');
    mockApiRequest.mockResolvedValueOnce(createMockResponse(filteredRecords));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchRecords('example.com', 'A');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records?zone=example.com&type=A', {
      method: 'GET'
    });
    expect(result.current.records).toEqual(filteredRecords);
  });

  it('handles fetch DNS records error', async () => {
    const errorMessage = 'Failed to fetch DNS records';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchRecords();
    });

    expect(result.current.records).toEqual([]);
    expect(result.current.error).toBe(errorMessage);
  });

  it('fetches DNS zones successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockDNSZones));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchZones();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/zones', {
      method: 'GET'
    });
    expect(result.current.zones).toEqual(mockDNSZones);
  });

  it('fetches DNS status successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockDNSStatus));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchStatus();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/status', {
      method: 'GET'
    });
    expect(result.current.status).toEqual(mockDNSStatus);
  });

  it('creates DNS record successfully', async () => {
    const newRecord = { ...mockDNSRecords[0], id: 'record-new', name: 'new.example.com' };
    mockApiRequest.mockResolvedValueOnce(createMockResponse(newRecord));

    const { result } = renderHook(() => useDNSManagement());

    const createRequest: CreateDNSRecordRequest = {
      name: 'new.example.com',
      type: 'A',
      value: '192.168.1.2',
      ttl: 300,
      zone: 'example.com',
      proxied: false,
      priority: 0,
      comment: 'New test record'
    };

    let createdRecord: DNSRecord | null = null;
    await act(async () => {
      createdRecord = await result.current.createRecord(createRequest);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records', {
      method: 'POST',
      body: JSON.stringify(createRequest)
    });
    expect(createdRecord).toEqual(newRecord);
  });

  it('updates DNS record successfully', async () => {
    const updatedRecord = { ...mockDNSRecords[0], value: '192.168.1.3' };
    mockApiRequest.mockResolvedValueOnce(createMockResponse(updatedRecord));

    const { result } = renderHook(() => useDNSManagement());

    const updateRequest: UpdateDNSRecordRequest = {
      value: '192.168.1.3',
      ttl: 600
    };

    let updated: DNSRecord | null = null;
    await act(async () => {
      updated = await result.current.updateRecord('record-1', updateRequest);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records/record-1', {
      method: 'PUT',
      body: JSON.stringify(updateRequest)
    });
    expect(updated).toEqual(updatedRecord);
  });

  it('deletes DNS record successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useDNSManagement());

    let deleted: boolean = false;
    await act(async () => {
      deleted = await result.current.deleteRecord('record-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records/record-1', {
      method: 'DELETE'
    });
    expect(deleted).toBe(true);
  });

  it('refreshes all data successfully', async () => {
    mockApiRequest
      .mockResolvedValueOnce(createMockResponse(mockDNSRecords))
      .mockResolvedValueOnce(createMockResponse(mockDNSZones))
      .mockResolvedValueOnce(createMockResponse(mockDNSStatus));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.refreshAll();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/records', { method: 'GET' });
    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/zones', { method: 'GET' });
    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/dns/status', { method: 'GET' });
    
    expect(result.current.records).toEqual(mockDNSRecords);
    expect(result.current.zones).toEqual(mockDNSZones);
    expect(result.current.status).toEqual(mockDNSStatus);
  });

  it('gets records by zone', () => {
    const { result } = renderHook(() => useDNSManagement());

    // Set initial data
    act(() => {
      result.current.records = mockDNSRecords;
    });

    const exampleRecords = result.current.getRecordsByZone('example.com');
    const testRecords = result.current.getRecordsByZone('test.com');

    expect(exampleRecords).toHaveLength(2);
    expect(exampleRecords.every(r => r.zone === 'example.com')).toBe(true);
    expect(testRecords).toHaveLength(1);
    expect(testRecords[0].zone).toBe('test.com');
  });

  it('gets records by type', () => {
    const { result } = renderHook(() => useDNSManagement());

    // Set initial data
    act(() => {
      result.current.records = mockDNSRecords;
    });

    const aRecords = result.current.getRecordsByType('A');
    const mxRecords = result.current.getRecordsByType('MX');
    const cnameRecords = result.current.getRecordsByType('CNAME');

    expect(aRecords).toHaveLength(1);
    expect(aRecords[0].type).toBe('A');
    expect(mxRecords).toHaveLength(1);
    expect(mxRecords[0].type).toBe('MX');
    expect(cnameRecords).toHaveLength(1);
    expect(cnameRecords[0].type).toBe('CNAME');
  });

  it('gets record types summary', () => {
    const { result } = renderHook(() => useDNSManagement());

    // Set initial data
    act(() => {
      result.current.records = mockDNSRecords;
    });

    const summary = result.current.getRecordTypesSummary();

    expect(summary).toEqual({
      A: 1,
      MX: 1,
      CNAME: 1
    });
  });

  it('handles loading states correctly', async () => {
    let resolvePromise: (value: any) => void;
    const promise = new Promise((resolve) => {
      resolvePromise = resolve;
    });
    mockApiRequest.mockReturnValueOnce(promise as Promise<APIResponse<unknown>>);

    const { result } = renderHook(() => useDNSManagement());

    // Start async operation
    act(() => {
      result.current.fetchRecords();
    });

    // Should be loading
    expect(result.current.isLoading).toBe(true);

    // Resolve the promise
    await act(async () => {
      resolvePromise!(createMockResponse(mockDNSRecords));
      await promise;
    });

    // Should no longer be loading
    expect(result.current.isLoading).toBe(false);
  });

  it('handles API response errors', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null, false, 'DNS service unavailable'));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchRecords();
    });

    expect(result.current.error).toBe('DNS service unavailable');
    expect(result.current.records).toEqual([]);
  });

  it('handles non-array response data', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useDNSManagement());

    await act(async () => {
      await result.current.fetchRecords();
    });

    expect(result.current.error).toBe('Failed to fetch DNS records');
    expect(result.current.records).toEqual([]);
  });

  it('handles create record failure', async () => {
    mockApiRequest.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useDNSManagement());

    const createRequest: CreateDNSRecordRequest = {
      name: 'fail.example.com',
      type: 'A',
      value: '192.168.1.2',
      ttl: 300,
      zone: 'example.com',
      proxied: false,
      priority: 0,
      comment: 'This will fail'
    };

    let createdRecord: DNSRecord | null = null;
    await act(async () => {
      createdRecord = await result.current.createRecord(createRequest);
    });

    expect(createdRecord).toBeNull();
    expect(result.current.error).toBe('Network error');
  });

  it('handles update record failure', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null, false, 'Record not found'));

    const { result } = renderHook(() => useDNSManagement());

    const updateRequest: UpdateDNSRecordRequest = {
      value: '192.168.1.3'
    };

    let updated: DNSRecord | null = null;
    await act(async () => {
      updated = await result.current.updateRecord('nonexistent', updateRequest);
    });

    expect(updated).toBeNull();
    expect(result.current.error).toBe('Record not found');
  });

  it('handles delete record failure', async () => {
    mockApiRequest.mockRejectedValueOnce(new Error('Permission denied'));

    const { result } = renderHook(() => useDNSManagement());

    let deleted: boolean = false;
    await act(async () => {
      deleted = await result.current.deleteRecord('record-1');
    });

    expect(deleted).toBe(false);
    expect(result.current.error).toBe('Permission denied');
  });
});
