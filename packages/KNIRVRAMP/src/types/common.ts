export interface ProcessingStep {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  message?: string;
  duration?: number;
}

export interface ProcessingResult {
  success: boolean;
  steps: ProcessingStep[];
  compiledAgent?: {
    id: string;
    name: string;
    version: string;
    size: number;
    checksum: string;
  };
  errors?: string[];
  warnings?: string[];
}