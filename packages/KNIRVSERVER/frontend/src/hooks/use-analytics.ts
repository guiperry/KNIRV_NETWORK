'use client';

import { useQuery } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export interface LoadPrediction {
  metric: string;
  predicted_load: number;
  confidence: number;
  predicted_at: string;
  prediction_window: string;
  is_anomalous: boolean;
  anomaly_score: number;
  linear_regression: {
    slope: number;
    intercept: number;
    r2: number;
  };
}

export interface RecommendedAction {
  recommended_action: 'scale_up' | 'scale_down' | 'maintain';
  should_trigger_proactive: boolean;
  timestamp: string;
}

export interface TrendDirection {
  metric: string;
  direction: 'increasing' | 'decreasing' | 'stable';
  timestamp: string;
}

export interface CapacityForecast {
  current_utilization: number;
  projected_utilization: number;
  horizon: string;
  timestamp: string;
}

export interface MetricData {
  values: number[];
  count: number;
  latest: number;
  min: number;
  max: number;
  avg: number;
}

export interface AnalyticsMetrics {
  [metric: string]: MetricData;
}

export interface AnalyticsResponse {
  success: boolean;
  data?: unknown;
  message?: string;
  error?: string;
  timestamp: string;
}

export const useAnalytics = () => {
  const queryKey = ['analytics'];

  const predictLoad = useQuery<LoadPrediction>({
    queryKey: [...queryKey, 'predict'],
    queryFn: async () => {
      const response = await apiRequest<LoadPrediction>(`${API_BASE_URL}/api/analytics/predict`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch load prediction');
    },
    staleTime: 30000,
  });

  const recommendations = useQuery<RecommendedAction>({
    queryKey: [...queryKey, 'recommendations'],
    queryFn: async () => {
      const response = await apiRequest<RecommendedAction>(`${API_BASE_URL}/api/analytics/recommendations`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch recommendations');
    },
    staleTime: 30000,
  });

  const trend = useQuery<TrendDirection>({
    queryKey: [...queryKey, 'trend'],
    queryFn: async () => {
      const response = await apiRequest<TrendDirection>(`${API_BASE_URL}/api/analytics/trend`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch trend direction');
    },
    staleTime: 30000,
  });

  const forecast = useQuery<CapacityForecast>({
    queryKey: [...queryKey, 'forecast'],
    queryFn: async () => {
      const response = await apiRequest<CapacityForecast>(`${API_BASE_URL}/api/analytics/forecast`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch capacity forecast');
    },
    staleTime: 30000,
  });

  const metrics = useQuery<AnalyticsMetrics>({
    queryKey: [...queryKey, 'metrics'],
    queryFn: async () => {
      const response = await apiRequest<AnalyticsMetrics>(`${API_BASE_URL}/api/analytics/metrics`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch metrics');
    },
    staleTime: 30000,
  });

  return {
    predictLoad,
    recommendations,
    trend,
    forecast,
    metrics,
    refetchAll: async () => {
      await Promise.all([
        predictLoad.refetch(),
        recommendations.refetch(),
        trend.refetch(),
        forecast.refetch(),
        metrics.refetch(),
      ]);
    },
  };
};

export default useAnalytics;