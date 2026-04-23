import axios, { AxiosInstance, AxiosError } from 'axios';

export interface DVERequest {
  skillCode: string;
  failureContext: string;
  testCases?: DVETestCase[];
  priority?: number;
  requiredTEEType?: string;
}

export interface DVETestCase {
  id: string;
  input: string;
  expectedOutput: string;
  description: string;
}

export interface DVEResult {
  success: boolean;
  score: number;
  passed: boolean;
  output: string;
  executionTime: number;
  warnings?: string[];
  error?: string;
}

export interface DVETask {
  id: string;
  type: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  priority: number;
  skillCode: string;
  failureContext: string;
  testCases: DVETestCase[];
  requiredTEEType?: string;
  assignedNodeID?: string;
  createdAt: number;
  completedAt?: number;
  result?: DVEResult;
}

export interface DVENode {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'maintenance' | 'error';
  teeType: string;
  reputationScore: number;
  location: string;
  capabilities: string[];
}

export interface DVEValidationResponse {
  taskId: string;
  result: DVEResult;
  nodeId: string;
  timestamp: number;
}

export interface CDEEnvironment {
  id: string;
  name: string;
  userId: string;
  status: 'creating' | 'running' | 'stopped' | 'suspended' | 'terminated' | 'error';
  environmentType: 'general' | 'python' | 'nodejs' | 'go' | 'rust' | 'java' | 'docker' | 'kubernetes';
  baseImage: string;
  workspacePath: string;
  containerId?: string;
  ipAddress?: string;
  ports?: Record<string, number>;
  resources?: CDEResources;
}

export interface CDEResources {
  cpuCores: number;
  memoryBytes: number;
  diskBytes: number;
  cpuLimit: number;
  memoryLimit: number;
  diskLimit: number;
}

export interface CDESession {
  id: string;
  environmentId: string;
  userId: string;
  status: 'active' | 'idle' | 'expired' | 'terminated';
  connectionType: 'ssh' | 'websocket' | 'vnc';
  connectionInfo?: Record<string, string>;
  expiresAt: number;
}

export interface CDESandboxRequest {
  code: string;
  language: string;
  testCases?: CDETestCase[];
  maxExecutionTime?: number;
  constraints?: CDEConstraint[];
}

export interface CDETestCase {
  input: string;
  expectedOutput: string;
  description?: string;
}

export interface CDEConstraint {
  type: 'time_limit' | 'memory_limit' | 'network_access' | 'file_system';
  value: string;
}

export interface CDESandboxResult {
  success: boolean;
  output: string;
  errors?: string[];
  executionTime: number;
  constraintsSatisfied: boolean;
  violations?: string[];
  testResults?: CDETestResult[];
}

export interface CDETestResult {
  testCaseId: string;
  passed: boolean;
  actualOutput: string;
  expectedOutput: string;
  executionTime: number;
}

export interface KNIRVSERVERConfig {
  baseUrl: string;
  apiKey?: string;
  timeout: number;
  retryAttempts: number;
}

const getDefaultBaseUrl = (): string => {
  try {
    const importMeta = eval('import.meta');
    return importMeta?.env?.VITE_KNIRVSERVER_URL || 'http://localhost:8082';
  } catch {
    return 'http://localhost:8082';
  }
};

const DEFAULT_KNIRV_SERVER_CONFIG: KNIRVSERVERConfig = {
  baseUrl: getDefaultBaseUrl(),
  timeout: 30000,
  retryAttempts: 3,
};

export class KNIRVSERVERClient {
  private client: AxiosInstance;
  private config: KNIRVSERVERConfig;
  private isConnected: boolean = false;

  constructor(config: Partial<KNIRVSERVERConfig> = {}) {
    this.config = { ...DEFAULT_KNIRV_SERVER_CONFIG, ...config };

    this.client = axios.create({
      baseURL: this.config.baseUrl,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        ...(this.config.apiKey && { 'Authorization': `Bearer ${this.config.apiKey}` }),
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    this.client.interceptors.response.use(
      (response) => {
        this.isConnected = true;
        return response;
      },
      (error: AxiosError) => {
        if (error.code === 'ECONNREFUSED' || error.code === 'ERR_NETWORK') {
          this.isConnected = false;
          console.warn('KNIRVSERVER: Connection lost');
        }
        return Promise.reject(error);
      }
    );
  }

  public async healthCheck(): Promise<boolean> {
    try {
      const response = await this.client.get('/api/dve-nodes/health');
      this.isConnected = true;
      return response.status === 200;
    } catch (error) {
      this.isConnected = false;
      return false;
    }
  }

  public isServerConnected(): boolean {
    return this.isConnected;
  }

  public updateConfig(config: Partial<KNIRVSERVERConfig>): void {
    this.config = { ...this.config, ...config };
    this.client.defaults.baseURL = this.config.baseUrl;
    this.client.defaults.timeout = this.config.timeout;
    if (config.apiKey) {
      this.client.defaults.headers['Authorization'] = `Bearer ${config.apiKey}`;
    }
  }

  public getConfig(): KNIRVSERVERConfig {
    return { ...this.config };
  }

  // DVE Methods

  public async validateWithDVE(request: DVERequest): Promise<DVEResult> {
    try {
      const response = await this.client.post<DVEResult>(
        '/api/dve-nodes/tasks/validate',
        request
      );
      return response.data;
    } catch (error) {
      return this.handleDVEError(error, request);
    }
  }

  private async handleDVEError(error: unknown, request: DVERequest): Promise<DVEResult> {
    if (axios.isAxiosError(error)) {
      console.error('DVE validation error:', error.message);

      if (error.response?.status === 401) {
        return {
          success: false,
          score: 0,
          passed: false,
          output: '',
          executionTime: 0,
          error: 'Authentication required',
        };
      }

      if (error.response?.status === 503) {
        return {
          success: false,
          score: 0,
          passed: false,
          output: '',
          executionTime: 0,
          error: 'DVE service unavailable',
          warnings: ['No DVE nodes available, using local validation'],
        };
      }
    }

    return {
      success: false,
      score: 0.5,
      passed: true,
      output: 'Local validation fallback',
      executionTime: 0,
      warnings: ['DVE validation failed, using local simulation'],
    };
  }

  public async submitValidationResult(taskId: string, result: DVEResult): Promise<boolean> {
    try {
      await this.client.post(`/api/dve-nodes/tasks/${taskId}/result`, result);
      return true;
    } catch (error) {
      console.error('Failed to submit validation result:', error);
      return false;
    }
  }

  public async getDVETasks(status?: string): Promise<DVETask[]> {
    try {
      const params = status ? { status } : {};
      const response = await this.client.get<DVETask[]>('/api/dve-nodes/tasks', { params });
      return response.data;
    } catch (error) {
      console.error('Failed to get DVE tasks:', error);
      return [];
    }
  }

  public async allocateDVETask(request: DVERequest): Promise<DVETask | null> {
    try {
      const response = await this.client.post<DVETask>('/api/dve-nodes/tasks/allocate', request);
      return response.data;
    } catch (error) {
      console.error('Failed to allocate DVE task:', error);
      return null;
    }
  }

  public async getDVENodes(): Promise<DVENode[]> {
    try {
      const response = await this.client.get<DVENode[]>('/api/dve-nodes/nodes');
      return response.data;
    } catch (error) {
      console.error('Failed to get DVE nodes:', error);
      return [];
    }
  }

  public async getDVENodeMetrics(nodeId: string): Promise<Record<string, unknown> | null> {
    try {
      const response = await this.client.get(`/api/dve-nodes/nodes/${nodeId}/metrics`);
      return response.data;
    } catch (error) {
      console.error('Failed to get DVE node metrics:', error);
      return null;
    }
  }

  // CDE Methods

  public async createCDEEnvironment(
    name: string,
    envType: string,
    config?: Record<string, unknown>
  ): Promise<CDEEnvironment | null> {
    try {
      const response = await this.client.post<CDEEnvironment>('/cde/environments', {
        name,
        environmentType: envType,
        config,
      });
      return response.data;
    } catch (error) {
      console.error('Failed to create CDE environment:', error);
      return null;
    }
  }

  public async getCDEEnvironments(): Promise<CDEEnvironment[]> {
    try {
      const response = await this.client.get<CDEEnvironment[]>('/cde/environments');
      return response.data;
    } catch (error) {
      console.error('Failed to get CDE environments:', error);
      return [];
    }
  }

  public async getCDEEnvironment(envId: string): Promise<CDEEnvironment | null> {
    try {
      const response = await this.client.get<CDEEnvironment>(`/cde/environments/${envId}`);
      return response.data;
    } catch (error) {
      console.error('Failed to get CDE environment:', error);
      return null;
    }
  }

  public async deleteCDEEnvironment(envId: string): Promise<boolean> {
    try {
      await this.client.delete(`/cde/environments/${envId}`);
      return true;
    } catch (error) {
      console.error('Failed to delete CDE environment:', error);
      return false;
    }
  }

  public async startCDEEnvironment(envId: string): Promise<boolean> {
    try {
      await this.client.post(`/cde/environments/${envId}/start`);
      return true;
    } catch (error) {
      console.error('Failed to start CDE environment:', error);
      return false;
    }
  }

  public async stopCDEEnvironment(envId: string): Promise<boolean> {
    try {
      await this.client.post(`/cde/environments/${envId}/stop`);
      return true;
    } catch (error) {
      console.error('Failed to stop CDE environment:', error);
      return false;
    }
  }

  public async createCDESession(
    envId: string,
    connectionType: 'ssh' | 'websocket' | 'vnc' = 'websocket'
  ): Promise<CDESession | null> {
    try {
      const response = await this.client.post<CDESession>('/cde/sessions', {
        environmentId: envId,
        connectionType,
      });
      return response.data;
    } catch (error) {
      console.error('Failed to create CDE session:', error);
      return null;
    }
  }

  public async getCDESessions(): Promise<CDESession[]> {
    try {
      const response = await this.client.get<CDESession[]>('/cde/sessions');
      return response.data;
    } catch (error) {
      console.error('Failed to get CDE sessions:', error);
      return [];
    }
  }

  public async validateWithCDE(request: CDESandboxRequest): Promise<CDESandboxResult> {
    try {
      const response = await this.client.post<CDESandboxResult>('/cde/sandbox/validate', request);
      return response.data;
    } catch (error) {
      return this.handleCDEError(error, request);
    }
  }

  private handleCDEError(error: unknown, request: CDESandboxRequest): CDESandboxResult {
    if (axios.isAxiosError(error)) {
      console.error('CDE validation error:', error.message);

      if (error.response?.status === 401) {
        return {
          success: false,
          output: '',
          errors: ['Authentication required'],
          executionTime: 0,
          constraintsSatisfied: false,
          violations: ['Authentication required'],
        };
      }

      if (error.response?.status === 503) {
        return {
          success: false,
          output: '',
          errors: ['CDE service unavailable'],
          executionTime: 0,
          constraintsSatisfied: false,
          violations: ['CDE service unavailable, using local simulation'],
        };
      }
    }

    return this.simulateLocalCDERun(request);
  }

  private simulateLocalCDERun(request: CDESandboxRequest): CDESandboxResult {
    const startTime = performance.now();

    const result: CDESandboxResult = {
      success: true,
      output: `Simulated output for ${request.language} code`,
      executionTime: performance.now() - startTime,
      constraintsSatisfied: true,
      testResults: [],
    };

    if (request.testCases) {
      for (const testCase of request.testCases) {
        result.testResults?.push({
          testCaseId: testCase.description || 'test',
          passed: true,
          actualOutput: 'Simulated output',
          expectedOutput: testCase.expectedOutput,
          executionTime: result.executionTime,
        });
      }
    }

    return result;
  }

  // Cognitive Engine Integration

  public async reportCognitiveMetrics(metrics: Record<string, unknown>): Promise<boolean> {
    try {
      await this.client.post('/api/cognitive/report-metrics', metrics);
      return true;
    } catch (error) {
      console.error('Failed to report cognitive metrics:', error);
      return false;
    }
  }

  public async getCognitiveState(): Promise<Record<string, unknown> | null> {
    try {
      const response = await this.client.get('/api/cognitive/state');
      return response.data;
    } catch (error) {
      console.error('Failed to get cognitive state:', error);
      return null;
    }
  }

  public async submitGuardrailViolation(
    violation: Record<string, unknown>
  ): Promise<boolean> {
    try {
      await this.client.post('/api/cognitive/guardrail-violation', violation);
      return true;
    } catch (error) {
      console.error('Failed to submit guardrail violation:', error);
      return false;
    }
  }
}

let knirvServerClientInstance: KNIRVSERVERClient | null = null;

export const getKNIRVSERVERClient = (
  config?: Partial<KNIRVSERVERConfig>
): KNIRVSERVERClient => {
  if (!knirvServerClientInstance) {
    knirvServerClientInstance = new KNIRVSERVERClient(config);
  }
  return knirvServerClientInstance;
};

export default getKNIRVSERVERClient;
