import {
  KNIRVSERVERClient,
  type KNIRVSERVERConfig,
  type DVERequest,
  type DVEResult,
  type DVETask,
  type DVENode,
  type DVEValidationResponse,
} from "../services/KNIRVSERVERClient";

export type {
  KNIRVSERVERConfig as DVEClientConfig,
  DVERequest,
  DVEResult,
  DVETask,
  DVENode,
  DVEValidationResponse,
} from "../services/KNIRVSERVERClient";

export class DVEClient {
  private readonly client: KNIRVSERVERClient;

  constructor(config: Partial<KNIRVSERVERConfig> = {}) {
    this.client = new KNIRVSERVERClient(config);
  }

  healthCheck(): Promise<boolean> {
    return this.client.healthCheck();
  }

  isConnected(): boolean {
    return this.client.isServerConnected();
  }

  updateConfig(config: Partial<KNIRVSERVERConfig>): void {
    this.client.updateConfig(config);
  }

  getConfig(): KNIRVSERVERConfig {
    return this.client.getConfig();
  }

  validateWithDVE(request: DVERequest): Promise<DVEResult> {
    return this.client.validateWithDVE(request);
  }

  submitValidationResult(taskId: string, result: DVEResult): Promise<boolean> {
    return this.client.submitValidationResult(taskId, result);
  }

  getDVETasks(status?: string): Promise<DVETask[]> {
    return this.client.getDVETasks(status);
  }

  allocateDVETask(request: DVERequest): Promise<DVETask | null> {
    return this.client.allocateDVETask(request);
  }

  getDVENodes(): Promise<DVENode[]> {
    return this.client.getDVENodes();
  }

  getDVENodeMetrics(nodeId: string): Promise<Record<string, unknown> | null> {
    return this.client.getDVENodeMetrics(nodeId);
  }
}

export const createDVEClient = (config: Partial<KNIRVSERVERConfig> = {}): DVEClient =>
  new DVEClient(config);

