import { Socket, Channel } from "phoenix";

export interface DVEClientConfig {
  baseUrl: string;
  apiKey?: string;
  timeout: number;
  retryAttempts: number;
}

export type { DVERequest, DVEResult, DVETask, DVENode, DVEValidationResponse } from "../services/KNIRVSERVERClient";

export class DVEClient {
  private socket: Socket;
  private dveChannels: Map<string, Channel> = new Map();
  private userToken: string;
  private config: DVEClientConfig;

  constructor(config: Partial<DVEClientConfig> = {}) {
    this.config = {
      baseUrl: config.baseUrl || "http://localhost:8082",
      apiKey: config.apiKey,
      timeout: config.timeout || 30000,
      retryAttempts: config.retryAttempts || 3,
    };
    this.userToken = this.config.apiKey || "";

    const endpoint = `${this.config.baseUrl}/socket`;
    this.socket = new Socket(endpoint, { params: { token: this.userToken } });
    this.socket.connect();
  }

  healthCheck(): Promise<boolean> {
    return this.socket.isConnected()
      ? Promise.resolve(true)
      : new Promise((resolve) => {
          const timeout = setTimeout(() => resolve(false), this.config.timeout);
          this.socket.onOpen(() => {
            clearTimeout(timeout);
            resolve(true);
          });
          this.socket.onError(() => {
            clearTimeout(timeout);
            resolve(false);
          });
        });
  }

  isConnected(): boolean {
    return this.socket.isConnected();
  }

  updateConfig(config: Partial<DVEClientConfig>): void {
    this.config = { ...this.config, ...config };
    if (config.apiKey !== undefined) {
      this.userToken = config.apiKey;
    }
  }

  getConfig(): DVEClientConfig {
    return { ...this.config };
  }

  validateWithDVE(request: DVERequest): Promise<DVEResult> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel("dve:tasks");
      channel.push("validate", request)
        .receive("ok", (msg: DVEResult) => resolve(msg))
        .receive("error", (err: any) => reject(err));
    });
  }

  submitValidationResult(taskId: string, result: DVEResult): Promise<boolean> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel("dve:tasks");
      channel.push("submit_result", { taskId, result })
        .receive("ok", () => resolve(true))
        .receive("error", () => reject(new Error("Failed to submit validation result")));
    });
  }

  getDVETasks(status?: string): Promise<DVETask[]> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel("dve:tasks");
      channel.push("list_tasks", { status })
        .receive("ok", (msg: { tasks: DVETask[] }) => resolve(msg.tasks))
        .receive("error", (err: any) => reject(err));
    });
  }

  allocateDVETask(request: DVERequest): Promise<DVETask | null> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel("dve:tasks");
      channel.push("allocate", request)
        .receive("ok", (msg: DVETask) => resolve(msg))
        .receive("error", () => resolve(null));
    });
  }

  getDVENodes(): Promise<DVENode[]> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel("dve:lobby");
      channel.push("list_nodes")
        .receive("ok", (msg: { nodes: DVENode[] }) => resolve(msg.nodes))
        .receive("error", (err: any) => reject(err));
    });
  }

  getDVENodeMetrics(nodeId: string): Promise<Record<string, unknown> | null> {
    return new Promise((resolve, reject) => {
      const channel = this.getOrCreateChannel(`dve:${nodeId}`);
      channel.push("get_metrics")
        .receive("ok", (msg: { metrics: Record<string, unknown> }) => resolve(msg.metrics))
        .receive("error", () => resolve(null));
    });
  }

  onDVENodeDiscovered(callback: (node: DVENode) => void): void {
    const channel = this.getOrCreateChannel("dve:lobby");
    channel.on("dve-node-discovered", callback);
  }

  onTaskUpdated(callback: (task: DVETask) => void): void {
    const channel = this.getOrCreateChannel("dve:tasks");
    channel.on("task_updated", callback);
  }

  onValidationResult(callback: (response: DVEValidationResponse) => void): void {
    const channel = this.getOrCreateChannel("dve:tasks");
    channel.on("validation_result", callback);
  }

  private getOrCreateChannel(topic: string): Channel {
    if (this.dveChannels.has(topic)) {
      return this.dveChannels.get(topic)!;
    }

    const channel = this.socket.channel(topic, {});
    channel.join()
      .receive("ok", () => console.log(`Joined DVE channel: ${topic}`))
      .receive("error", (resp: any) => console.error(`Unable to join DVE channel ${topic}`, resp));

    this.dveChannels.set(topic, channel);
    return channel;
  }

  disconnect(): void {
    this.dveChannels.forEach((channel) => channel.leave());
    this.dveChannels.clear();
    this.socket.disconnect();
  }
}

export const createDVEClient = (config: Partial<DVEClientConfig> = {}): DVEClient =>
  new DVEClient(config);

