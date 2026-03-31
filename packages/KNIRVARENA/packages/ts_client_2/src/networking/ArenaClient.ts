import { Socket, Channel } from "phoenix";

export interface BountyContext {
  error_id: string;
  failed_output: string;
  error_class: string;
  instruction: string;
  environment_metadata: Record<string, any>;
  nrn_fee: number;
  bounty_tier: 'low' | 'medium' | 'high';
}

export class ArenaClient {
  private socket: Socket;
  private lobbyChannel: Channel;
  private resolutionChannels: Map<string, Channel> = new Map();
  private userToken: string;

  constructor(userToken: string, endpoint: string = "/socket") {
    this.userToken = userToken;
    this.socket = new Socket(endpoint, { params: { token: userToken } });
    this.socket.connect();
    
    // Join the lobby to hear about new bounties
    this.lobbyChannel = this.socket.channel("arena:lobby", {});
    this.lobbyChannel.join()
      .receive("ok", () => console.log("Joined KNIRVARENA Lobby"))
      .receive("error", resp => console.error("Unable to join lobby", resp));
      
    // Listen for new bounties
    this.lobbyChannel.on("new_bounty", (payload: BountyContext) => {
      console.log("New Bounty Received:", payload.error_id);
      // Emit event for UI or other services
      this.onNewBounty(payload);
    });
  }

  private onNewBounty(_payload: BountyContext) {
    // Override this for custom handling
  }

  public joinResolution(errorId: string, userId: string, reputationScore: number): Channel {
    if (this.resolutionChannels.has(errorId)) {
      return this.resolutionChannels.get(errorId)!;
    }

    const channel = this.socket.channel(`arena:resolution:${errorId}`, {});
    channel.join()
      .receive("ok", () => {
        console.log(`Entered Error Node resolution loop: ${errorId}`);
        // Send join_session message as per implementation in Elixir
        channel.push("join_session", { user_id: userId, reputation_score: reputationScore });
      })
      .receive("error", resp => console.error("Access Denied", resp));

    this.resolutionChannels.set(errorId, channel);
    return channel;
  }

  public submitSolution(errorId: string, userId: string, trajectory: any): Promise<any> {
    const channel = this.resolutionChannels.get(errorId);
    if (!channel) return Promise.reject("Not joined in this resolution channel");

    return new Promise((resolve, reject) => {
      channel.push("submit_solution", { trajectory, user_id: userId })
        .receive("ok", (msg) => {
          console.log("Solution Validated!", msg);
          resolve(msg);
        })
        .receive("error", (reasons) => {
          console.error("Logic Failure", reasons);
          reject(reasons);
        });
    });
  }

  public leaveResolution(errorId: string) {
    const channel = this.resolutionChannels.get(errorId);
    if (channel) {
      channel.leave();
      this.resolutionChannels.delete(errorId);
    }
  }

  public disconnect() {
    this.socket.disconnect();
  }
}

// Export a singleton or factory function as needed
export const createArenaClient = (token: string) => new ArenaClient(token);
