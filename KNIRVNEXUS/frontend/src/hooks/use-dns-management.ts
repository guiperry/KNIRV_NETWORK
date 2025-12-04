"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  DNSRecord,
  DNSZone,
  DNSStatus,
  CreateDNSRecordRequest,
  UpdateDNSRecordRequest
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export const useDNSManagement = () => {
  const [records, setRecords] = useState<DNSRecord[]>([]);
  const [zones, setZones] = useState<DNSZone[]>([]);
  const [status, setStatus] = useState<DNSStatus | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch DNS records
  const fetchRecords = useCallback(async (zone?: string, type?: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const queryParams = new URLSearchParams();
      if (zone) queryParams.append('zone', zone);
      if (type) queryParams.append('type', type);

      const url = `${API_BASE_URL}/api/dns/records${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
      const response: APIResponse<DNSRecord[]> = await apiRequest(url, { method: 'GET' });

      if (response.success && Array.isArray(response.data)) {
        setRecords(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch DNS records');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DNS records:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch DNS zones
  const fetchZones = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/zones`;
      const response: APIResponse<DNSZone[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setZones(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch DNS zones');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DNS zones:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch DNS service status
  const fetchStatus = useCallback(async () => {
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/status`;
      const response: APIResponse<DNSStatus> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        setStatus(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch DNS status');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DNS status:', err);
    }
  }, []);

  // Create DNS record
  const createRecord = useCallback(async (recordData: CreateDNSRecordRequest): Promise<DNSRecord | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/records`;
      const response: APIResponse<DNSRecord> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(recordData),
      });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        const newRecord = response.data;
        setRecords(prevRecords => [...prevRecords, newRecord]);
        return newRecord;
      } else {
        throw new Error(response.error || 'Failed to create DNS record');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create DNS record:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Update DNS record
  const updateRecord = useCallback(async (recordId: string, updateData: UpdateDNSRecordRequest): Promise<DNSRecord | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/records/${recordId}`;
      const response: APIResponse<DNSRecord> = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updateData),
      });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        const updatedRecord = response.data;
        setRecords(prevRecords =>
          prevRecords.map(record =>
            record.id === recordId ? updatedRecord : record
          )
        );
        return updatedRecord;
      } else {
        throw new Error(response.error || 'Failed to update DNS record');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update DNS record:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Delete DNS record
  const deleteRecord = useCallback(async (recordId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/records/${recordId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        setRecords(prevRecords => prevRecords.filter(record => record.id !== recordId));
        return true;
      } else {
        throw new Error(response.error || 'Failed to delete DNS record');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to delete DNS record:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Get DNS record by ID
  const getRecord = useCallback(async (recordId: string): Promise<DNSRecord | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dns/records/${recordId}`;
      const response: APIResponse<DNSRecord> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch DNS record');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DNS record:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchRecords(),
      fetchZones(),
      fetchStatus(),
    ]);
  }, [fetchRecords, fetchZones, fetchStatus]);

  // Filter records by zone
  const getRecordsByZone = useCallback((zoneName: string) => 
    records.filter(record => record.zone === zoneName), [records]);

  // Filter records by type
  const getRecordsByType = useCallback((recordType: string) => 
    records.filter(record => record.type === recordType), [records]);

  // Get record types summary
  const getRecordTypesSummary = useCallback(() => {
    const summary: Record<string, number> = {};
    records.forEach(record => {
      summary[record.type] = (summary[record.type] || 0) + 1;
    });
    return summary;
  }, [records]);

  // Initial fetch on mount - disabled for testing
  // useEffect(() => {
  //   refreshAll();
  // }, [refreshAll]);

  return {
    records,
    zones,
    status,
    isLoading,
    error,
    fetchRecords,
    fetchZones,
    fetchStatus,
    createRecord,
    updateRecord,
    deleteRecord,
    getRecord,
    refreshAll,
    getRecordsByZone,
    getRecordsByType,
    getRecordTypesSummary,
  };
};

export default useDNSManagement;
