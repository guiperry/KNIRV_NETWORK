export interface MetricOptions {
  name: string;
  help: string;
  labelNames?: string[];
}

export interface CounterOptions extends MetricOptions {
  labelNames?: string[];
}

export interface GaugeOptions extends MetricOptions {
  labelNames?: string[];
}

export interface HistogramOptions extends MetricOptions {
  buckets?: number[];
  labelNames?: string[];
}

export interface SummaryOptions extends MetricOptions {
  percentiles?: number[];
  labelNames?: string[];
}

export interface MetricLabels {
  [key: string]: string | number;
}

export interface MetricValue {
  value: number;
  labels?: MetricLabels;
}

export interface HistogramMeasurement {
  labels?: MetricLabels;
  value: number;
}