// EventEmitter Template for WASM Compilation
export class EventEmitter {
  private listeners: Map<string, Function[]> = new Map();
  private isActive: boolean = false;

  constructor() {
    this.isActive = true;
  }

  on(event: string, listener: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(listener);
  }

  emit(event: string, ...args: any[]): void {
    if (!this.isActive) {
      return;
    }
    
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      eventListeners.forEach(listener => {
        try {
          listener(...args);
        } catch (error) {
          // Handle listener errors gracefully
          console.error(`Error in event listener for ${event}:`, error);
        }
      });
    }
  }

  off(event: string, listener?: Function): void {
    if (!listener) {
      this.listeners.delete(event);
      return;
    }
    
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      const index = eventListeners.indexOf(listener);
      if (index > -1) {
        eventListeners.splice(index, 1);
      }
    }
  }

  removeAllListeners(): void {
    this.listeners.clear();
  }

  isReady(): boolean {
    return this.isActive;
  }

  shutdown(): void {
    this.isActive = false;
    this.removeAllListeners();
  }
}
