// Browser-compatible EventEmitter implementation
export class EventEmitter {
  private events: Map<string, Function[]> = new Map();

  on(event: string, listener: Function): this {
    if (!this.events.has(event)) {
      this.events.set(event, []);
    }
    this.events.get(event)!.push(listener);
    return this;
  }

  off(event: string, listener?: Function): this {
    if (!listener) {
      // Remove all listeners for this event
      this.events.delete(event);
    } else {
      const listeners = this.events.get(event);
      if (listeners) {
        const index = listeners.indexOf(listener);
        if (index > -1) {
          listeners.splice(index, 1);
          // Clean up empty event arrays
          if (listeners.length === 0) {
            this.events.delete(event);
          }
        }
      }
    }
    return this;
  }

  emit(event: string, ...args: any[]): boolean {
    const listeners = this.events.get(event);
    if (listeners) {
      // Create a copy to avoid issues with concurrent modification during iteration
      const listenersCopy = [...listeners];
      listenersCopy.forEach(listener => {
        try {
          listener(...args);
        } catch (error) {
          console.error('EventEmitter error:', error);
        }
      });
      return true;
    }
    return false;
  }

  once(event: string, listener: Function): this {
    const onceWrapper = (...args: any[]) => {
      this.off(event, onceWrapper);
      listener(...args);
    };
    return this.on(event, onceWrapper);
  }

  removeAllListeners(event?: string): this {
    if (event) {
      this.events.delete(event);
    } else {
      this.events.clear();
    }
    return this;
  }

  listenerCount(event: string): number {
    const listeners = this.events.get(event);
    return listeners ? listeners.length : 0;
  }

  listeners(event: string): Function[] {
    return this.events.get(event) || [];
  }

  eventNames(): string[] {
    return Array.from(this.events.keys());
  }
}
