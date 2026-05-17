'use client';

import { useState, useCallback } from 'react';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

export interface BadgeTemplate {
  id: string;
  name: string;
  badge_type: string;
  description: string;
  value_signals: string[];
  ontology_signals: string[];
  ai_text_tag: string;
  primary_color: string;
  secondary_color: string;
  background_color: string;
  alignment_threshold: number;
  auth_credentials: {
    api_key_provider?: string;
    api_key_scope?: string;
    jwt_claim?: string;
    jwt_value?: string;
    selected_role?: string;
  };
  svg_design: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
}

export interface MintFromTemplateResult {
  template_id: string;
  template_name: string;
  dve_id: string;
  badge_params: Record<string, unknown>;
  status: string;
}

interface UseBadgeTemplatesReturn {
  templates: BadgeTemplate[];
  isLoading: boolean;
  error: string | null;
  createTemplate: (tmpl: Partial<BadgeTemplate>) => Promise<BadgeTemplate | null>;
  listTemplates: (activeOnly?: boolean) => Promise<BadgeTemplate[]>;
  getTemplate: (id: string) => Promise<BadgeTemplate | null>;
  updateTemplate: (id: string, tmpl: Partial<BadgeTemplate>) => Promise<BadgeTemplate | null>;
  deleteTemplate: (id: string) => Promise<boolean>;
  mintFromTemplate: (templateId: string, dveId: string) => Promise<MintFromTemplateResult | null>;
}

export function useBadgeTemplates(): UseBadgeTemplatesReturn {
  const [templates, setTemplates] = useState<BadgeTemplate[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleResponse = async <T,>(response: Response): Promise<T | null> => {
    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(err.error || `HTTP ${response.status}`);
    }
    return response.json();
  };

  const createTemplate = useCallback(async (tmpl: Partial<BadgeTemplate>): Promise<BadgeTemplate | null> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(tmpl),
      });
      const data = await handleResponse<BadgeTemplate>(response);
      if (data) {
        setTemplates(prev => [data, ...prev]);
      }
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create template';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const listTemplates = useCallback(async (activeOnly = true): Promise<BadgeTemplate[]> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates?active=${activeOnly}`, {
        headers: getAuthHeaders(),
      });
      const data = await handleResponse<{ templates: BadgeTemplate[] }>(response);
      if (data) {
        setTemplates(data.templates);
        return data.templates;
      }
      return [];
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to list templates';
      setError(msg);
      return [];
    } finally {
      setIsLoading(false);
    }
  }, []);

  const getTemplate = useCallback(async (id: string): Promise<BadgeTemplate | null> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates/${id}`, {
        headers: getAuthHeaders(),
      });
      return await handleResponse<BadgeTemplate>(response);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to get template';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const updateTemplate = useCallback(async (id: string, tmpl: Partial<BadgeTemplate>): Promise<BadgeTemplate | null> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates/${id}`, {
        method: 'PUT',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(tmpl),
      });
      const data = await handleResponse<BadgeTemplate>(response);
      if (data) {
        setTemplates(prev => prev.map(t => t.id === id ? data : t));
      }
      return data;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to update template';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const deleteTemplate = useCallback(async (id: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates/${id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        setTemplates(prev => prev.filter(t => t.id !== id));
        return true;
      }
      throw new Error('Delete failed');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to delete template';
      setError(msg);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const mintFromTemplate = useCallback(async (templateId: string, dveId: string): Promise<MintFromTemplateResult | null> => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/api/badge/templates/${templateId}/mint`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ dve_id: dveId }),
      });
      return await handleResponse<MintFromTemplateResult>(response);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to mint from template';
      setError(msg);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    templates,
    isLoading,
    error,
    createTemplate,
    listTemplates,
    getTemplate,
    updateTemplate,
    deleteTemplate,
    mintFromTemplate,
  };
}
