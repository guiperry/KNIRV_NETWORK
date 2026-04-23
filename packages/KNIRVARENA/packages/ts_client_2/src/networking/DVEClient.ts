import { Socket, Channel } from "phoenix";

export interface ValidationRecord {
  id: string;
  dve_id: string;
  score: number;
  status: "passed" | "failed" | "pending";
  created_at: number;
  validator: string;
}

export interface ValidationRequest {
  dve_id: string;
  user_id: string;
  input: string;
  expected?: string;
}

export interface DVEClientConfig {
  endpoint?: string;
  userToken?: string;
}

export class DVEClient {
  private socket: Socket | null = null;
  private dveChannels: Map<string, Channel> = new Map();
  private userToken: string;

  constructor(userToken: string, endpoint: string = "/socket") {
    this.userToken = userToken;
    this.socket = new Socket(endpoint, { params: { token: userToken } });
  }

  public connect(): Promise<void> {
    if (!this.socket) {
      return Promise.reject("Socket not initialized");
    }
    
    return new Promise((resolve, reject) => {
      this.socket!.connect();
      this.socket!.onOpen(() => resolve());
      this.socket!.onError((err) => reject(err));
    });
  }

  public disconnect(): void {
    if (this.socket) {
      this.socket.disconnect();
      this.socket = null;
    }
    this.dveChannels.clear();
  }

  public connectToDVE(dveId: string): Channel {
    if (this.dveChannels.has(dveId)) {
      return this.dveChannels.get(dveId)!;
    }

    if (!this.socket) {
      throw new Error("Socket not connected. Call connect() first.");
    }

    const channel = this.socket.channel(`dve:${dveId}`, {});
    channel.join()
      .receive("ok", () => console.log(`Connected to DVE: ${dveId}`))
      .receive("error", (resp: unknown) => console.error(`Failed to connect to DVE: ${dveId}`, resp));

    this.dveChannels.set(dveId, channel);
    return channel;
  }

  public onValidationRecord(callback: (record: ValidationRecord) => void): void {
    this.dveChannels.forEach((channel) => {
      channel.on("validation_record", (payload: ValidationRecord) => {
        callback(payload);
      });
    });
  }

  public sendValidationRequest(req: ValidationRequest): Promise<ValidationRecord> {
    const channel = this.dveChannels.get(req.dve_id);
    if (!channel) {
      return Promise.reject(`Not connected to DVE: ${req.dve_id}. Call connectToDVE() first.`);
    }

    return new Promise((resolve, reject) => {
      channel.push("validation_request", req)
        .receive("ok", (record: ValidationRecord) => {
          resolve(record);
        })
        .receive("error", (reason: unknown) => {
          reject(reason);
        });
    });
  }

  public leaveDVE(dveId: string): void {
    const channel = this.dveChannels.get(dveId);
    if (channel) {
      channel.leave();
      this.dveChannels.delete(dveId);
    }
  }

  public getChannel(dveId: string): Channel | undefined {
    return this.dveChannels.get(dveId);
  }

  public isConnected(dveId: string): boolean {
    const channel = this.dveChannels.get(dveId);
    return channel !== undefined && channel.joinedOnce;
  }
}

export default DVEClient;