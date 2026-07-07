interface QRData {
  version: string;
  type: string;
  session_id: string;
  desktop_id: string;
  target_id?: string;
  expires_at: number;
  endpoint: string;
  public_key: string;
  capabilities?: string[];
  encrypted_payload?: string;
  signature: string;
}

interface ConnectionStatus {
  connected: boolean;
  desktop_id?: string;
  session_id?: string;
  secure_endpoint?: string;
  last_heartbeat?: number;
  error?: string;
}

interface MobileLinkageData {
  device_id: string;
  wallet_address: string;
  public_key: string;
  capabilities: string[];
  signature: string;
}

interface HRMProcessingRequest {
  sensory_data: number[];
  context: string;
  task_type: string;
}

interface HRMProcessingResponse {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations: number[];
  h_module_activations: number[];
}

interface WebSocketMessage {
  type: 'hrm_response' | 'heartbeat' | 'error' | 'status' | string;
  data?: HRMProcessingResponse | Record<string, unknown>;
  timestamp?: number;
  session_id?: string;
}

export class DesktopConnectionService {
  private connectionStatus: ConnectionStatus = { connected: false };
  private websocket: WebSocket | null = null;
  private heartbeatInterval: number | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  // Event handlers
  private onConnectionChange: ((status: ConnectionStatus) => void) | null = null;
  private onHRMResponse: ((response: HRMProcessingResponse) => void) | null = null;
  private onMessage: ((message: unknown) => void) | null = null;

  constructor() {
    this.generateDeviceId();
  }

  // Generate unique device ID for this mobile device
  private generateDeviceId(): string {
    let deviceId = localStorage.getItem('mobile_device_id');
    if (!deviceId) {
      deviceId = 'mobile_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
      localStorage.setItem('mobile_device_id', deviceId);
    }
    return deviceId;
  }

  // Set event handlers
  setConnectionChangeHandler(handler: (status: ConnectionStatus) => void) {
    this.onConnectionChange = handler;
  }

  setHRMResponseHandler(handler: (response: HRMProcessingResponse) => void) {
    this.onHRMResponse = handler;
  }

  setMessageHandler(handler: (message: unknown) => void) {
    this.onMessage = handler;
  }

  // Connect to desktop host using QR code data
  async connectToDesktop(qrData: QRData): Promise<boolean> {
    try {
      console.log('Connecting to desktop:', qrData);

      // Validate QR code
      if (!this.validateQRCode(qrData)) {
        throw new Error('Invalid QR code');
      }

      // Prepare mobile linkage data
      const linkageData: MobileLinkageData = {
        device_id: this.generateDeviceId(),
        wallet_address: this.getWalletAddress(),
        public_key: this.getPublicKey(),
        capabilities: this.getDeviceCapabilities(),
        signature: this.signLinkageData(qrData.session_id)
      };

      // Send connection request to desktop
      const response = await fetch(`${qrData.endpoint}/api/mobile/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          session_id: qrData.session_id,
          ...linkageData
        })
      });

      if (!response.ok) {
        throw new Error(`Connection failed: ${response.statusText}`);
      }

      const connectionResult = await response.json();
      console.log('Connection established:', connectionResult);

      // Update connection status
      this.connectionStatus = {
        connected: true,
        desktop_id: qrData.desktop_id,
        session_id: qrData.session_id,
        secure_endpoint: connectionResult.secure_endpoint,
        last_heartbeat: Date.now()
      };

      // Establish WebSocket connection for real-time communication
      await this.establishWebSocketConnection(qrData.endpoint, qrData.session_id);

      // Start heartbeat
      this.startHeartbeat();

      // Notify connection change
      if (this.onConnectionChange) {
        this.onConnectionChange(this.connectionStatus);
      }

      return true;
    } catch (error) {
      console.error('Failed to connect to desktop:', error);
      this.connectionStatus = {
        connected: false,
        error: error instanceof Error ? error.message : 'Unknown error'
      };

      if (this.onConnectionChange) {
        this.onConnectionChange(this.connectionStatus);
      }

      return false;
    }
  }

  // Establish WebSocket connection for real-time communication
  private async establishWebSocketConnection(endpoint: string, sessionId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = endpoint.replace('http', 'ws') + `/api/agent/ws?session_id=${sessionId}`;
      
      this.websocket = new WebSocket(wsUrl);

      this.websocket.onopen = () => {
        console.log('WebSocket connection established');
        this.reconnectAttempts = 0;
        resolve();
      };

      this.websocket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleWebSocketMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      this.websocket.onclose = () => {
        console.log('WebSocket connection closed');
        this.websocket = null;
        this.attemptReconnect();
      };

      this.websocket.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };
    });
  }

  // Handle incoming WebSocket messages
  private handleWebSocketMessage(message: unknown) {
    console.log('Received WebSocket message:', message);

    // Type guard to ensure message has the expected structure
    if (!this.isWebSocketMessage(message)) {
      console.warn('Received invalid WebSocket message format:', message);
      return;
    }

    switch (message.type) {
      case 'hrm_response':
        if (this.onHRMResponse && message.data) {
          this.onHRMResponse(message.data as HRMProcessingResponse);
        }
        break;
      case 'heartbeat':
        this.connectionStatus.last_heartbeat = Date.now();
        break;
      default:
        if (this.onMessage) {
          this.onMessage(message);
        }
    }
  }

  // Type guard for WebSocket messages
  private isWebSocketMessage(message: unknown): message is WebSocketMessage {
    return (
      typeof message === 'object' &&
      message !== null &&
      'type' in message &&
      typeof (message as Record<string, unknown>).type === 'string'
    );
  }

  // Send HRM processing request to desktop
  async sendHRMRequest(request: HRMProcessingRequest): Promise<HRMProcessingResponse | null> {
    if (!this.connectionStatus.connected || !this.connectionStatus.secure_endpoint) {
      throw new Error('Not connected to desktop');
    }

    try {
      const response = await fetch(`${this.connectionStatus.secure_endpoint}/api/hrm/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`HRM request failed: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error('HRM request failed:', error);
      return null;
    }
  }

  // Send message via WebSocket
  sendWebSocketMessage(message: unknown) {
    if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
      this.websocket.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }

  // Disconnect from desktop
  disconnect() {
    if (this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }

    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    this.connectionStatus = { connected: false };

    if (this.onConnectionChange) {
      this.onConnectionChange(this.connectionStatus);
    }
  }

  // Get current connection status
  getConnectionStatus(): ConnectionStatus {
    return { ...this.connectionStatus };
  }

  // Validate QR code data
  private validateQRCode(qrData: QRData): boolean {
    // Check required fields
    if (!qrData.version || !qrData.session_id || !qrData.desktop_id || !qrData.endpoint) {
      return false;
    }

    // Check expiration
    if (qrData.expires_at && Date.now() / 1000 > qrData.expires_at) {
      return false;
    }

    // Validate signature (simplified)
    // In production, implement proper cryptographic verification
    return !!(qrData.signature && qrData.signature.length > 0);
  }

  // Get wallet address (mock implementation)
  private getWalletAddress(): string {
    return localStorage.getItem('wallet_address') || 'mock_wallet_address';
  }

  // Get public key (mock implementation)
  private getPublicKey(): string {
    return localStorage.getItem('public_key') || 'mock_public_key';
  }

  // Get device capabilities
  private getDeviceCapabilities(): string[] {
    const capabilities = ['voice_processing', 'visual_processing', 'qr_scanning'];
    
    // Check for additional capabilities
    if (navigator.mediaDevices && typeof navigator.mediaDevices.getUserMedia === 'function') {
      capabilities.push('camera_access', 'microphone_access');
    }
    
    if ('geolocation' in navigator) {
      capabilities.push('location_access');
    }
    
    if ('vibrate' in navigator) {
      capabilities.push('haptic_feedback');
    }

    return capabilities;
  }

  // Sign linkage data (mock implementation)
  private signLinkageData(sessionId: string): string {
    // In production, implement proper cryptographic signing
    const data = `${this.generateDeviceId()}_${sessionId}_${Date.now()}`;
    return btoa(data); // Base64 encode as mock signature
  }

  // Start heartbeat to maintain connection
  private startHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }

    this.heartbeatInterval = window.setInterval(() => {
      if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
        this.sendWebSocketMessage({
          type: 'heartbeat',
          timestamp: Date.now(),
          device_id: this.generateDeviceId()
        });
      }
    }, 30000); // Send heartbeat every 30 seconds
  }

  // Attempt to reconnect WebSocket
  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('Max reconnection attempts reached');
      this.connectionStatus.connected = false;
      this.connectionStatus.error = 'Connection lost';
      
      if (this.onConnectionChange) {
        this.onConnectionChange(this.connectionStatus);
      }
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      if (this.connectionStatus.secure_endpoint && this.connectionStatus.session_id) {
        this.establishWebSocketConnection(
          this.connectionStatus.secure_endpoint,
          this.connectionStatus.session_id
        ).catch(error => {
          console.error('Reconnection failed:', error);
        });
      }
    }, delay);
  }
}

// Export singleton instance
export const desktopConnection = new DesktopConnectionService();
