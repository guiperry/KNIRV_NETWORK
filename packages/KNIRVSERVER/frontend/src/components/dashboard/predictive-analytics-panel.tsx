'use client';

import React from 'react';
import { TrendingUp, TrendingDown, Minus, ArrowUp, ArrowDown, RefreshCw, Brain, BarChart3, Clock, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { useAnalytics, LoadPrediction, RecommendedAction, TrendDirection, CapacityForecast } from '@/hooks/use-analytics';

const actionColors: Record<string, string> = {
  scale_up: 'bg-green-500/20 text-green-400 border-green-500/30',
  scale_down: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  maintain: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
};

const trendIcons: Record<string, React.ReactNode> = {
  increasing: <TrendingUp className="w-4 h-4 text-green-400" />,
  decreasing: <TrendingDown className="w-4 h-4 text-red-400" />,
  stable: <Minus className="w-4 h-4 text-yellow-400" />,
};

interface PredictiveAnalyticsPanelProps {
  className?: string;
}

export function PredictiveAnalyticsPanel({ className }: PredictiveAnalyticsPanelProps) {
  const { predictLoad, recommendations, trend, forecast, refetchAll } = useAnalytics();

  const handleRefresh = () => {
    refetchAll();
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Brain className="w-5 h-5 text-purple-400" />
            <CardTitle>Predictive Analytics</CardTitle>
          </div>
          <Button variant="outline" size="sm" onClick={handleRefresh}>
            <RefreshCw className="w-4 h-4 mr-1" />
            Refresh
          </Button>
        </div>
        <CardDescription>AI-powered load predictions and scaling recommendations</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-2">
          {/* Load Prediction */}
          <div className="p-4 rounded-lg bg-card border border-border">
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-medium flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-purple-400" />
                Load Prediction
              </h4>
              {predictLoad.data?.is_anomalous && (
                <Badge variant="outline" className="bg-red-500/20 text-red-400 border-red-500/30">
                  <AlertTriangle className="w-3 h-3 mr-1" />
                  Anomaly
                </Badge>
              )}
            </div>
            
            {predictLoad.isLoading && (
              <div className="flex items-center justify-center py-4">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {predictLoad.data && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-2xl font-bold">
                    {Math.round(predictLoad.data.predicted_load * 100)}%
                  </span>
                  <span className="text-sm text-muted-foreground">
                    Confidence: {Math.round(predictLoad.data.confidence * 100)}%
                  </span>
                </div>
                
                {predictLoad.data.linear_regression && (
                  <div className="text-xs text-muted-foreground space-y-1">
                    <div>R²: {predictLoad.data.linear_regression.r2.toFixed(2)}</div>
                    <div>Trend: {predictLoad.data.linear_regression.slope > 0 ? '+' : ''}
                        {predictLoad.data.linear_regression.slope.toFixed(3)}/min</div>
                  </div>
                )}

                {predictLoad.data.is_anomalous && (
                  <div className="text-xs text-red-400">
                    Anomaly Score: {(predictLoad.data.anomaly_score * 100).toFixed(1)}%
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Scaling Recommendation */}
          <div className="p-4 rounded-lg bg-card border border-border">
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-medium flex items-center gap-2">
                <ArrowUp className="w-4 h-4 text-blue-400" />
                Recommendation
              </h4>
              {recommendations.data?.should_trigger_proactive && (
                <Badge className="bg-purple-500/20 text-purple-400 border-purple-500/30">
                  Proactive
                </Badge>
              )}
            </div>
            
            {recommendations.isLoading && (
              <div className="flex items-center justify-center py-4">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {recommendations.data && (
              <div className="space-y-3">
                <Badge className={actionColors[recommendations.data.recommended_action]}>
                  {recommendations.data.recommended_action === 'scale_up' && <ArrowUp className="w-3 h-3 mr-1" />}
                  {recommendations.data.recommended_action === 'scale_down' && <ArrowDown className="w-3 h-3 mr-1" />}
                  {recommendations.data.recommended_action === 'maintain' && <Minus className="w-3 h-3 mr-1" />}
                  {recommendations.data.recommended_action.replace('_', ' ')}
                </Badge>
                
                {recommendations.data.should_trigger_proactive && (
                  <p className="text-xs text-purple-400">
                    Proactive scaling recommended based on trend analysis
                  </p>
                )}
              </div>
            )}
          </div>

          {/* Trend Direction */}
          <div className="p-4 rounded-lg bg-card border border-border">
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-medium flex items-center gap-2">
                <TrendingUp className="w-4 h-4 text-green-400" />
                Trend
              </h4>
            </div>
            
            {trend.isLoading && (
              <div className="flex items-center justify-center py-4">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {trend.data && (
              <div className="flex items-center gap-3">
                {trendIcons[trend.data.direction]}
                <span className="text-lg font-medium capitalize">{trend.data.direction}</span>
              </div>
            )}
          </div>

          {/* Capacity Forecast */}
          <div className="p-4 rounded-lg bg-card border border-border">
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-medium flex items-center gap-2">
                <Clock className="w-4 h-4 text-cyan-400" />
                Capacity Forecast
              </h4>
              <span className="text-xs text-muted-foreground">
                {forecast.data?.horizon}
              </span>
            </div>
            
            {forecast.isLoading && (
              <div className="flex items-center justify-center py-4">
                <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
              </div>
            )}

            {forecast.data && (
              <div className="space-y-3">
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span>Current</span>
                    <span>{Math.round(forecast.data.current_utilization * 100)}%</span>
                  </div>
                  <Progress 
                    value={forecast.data.current_utilization * 100} 
                    className="h-2"
                  />
                </div>
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span>Projected</span>
                    <span>{Math.round(forecast.data.projected_utilization * 100)}%</span>
                  </div>
                  <Progress 
                    value={forecast.data.projected_utilization * 100} 
                    className={`h-2 ${
                      forecast.data.projected_utilization > 0.8 
                        ? 'bg-red-500/20' 
                        : forecast.data.projected_utilization > 0.6 
                          ? 'bg-yellow-500/20' 
                          : ''
                    }`}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default PredictiveAnalyticsPanel;
