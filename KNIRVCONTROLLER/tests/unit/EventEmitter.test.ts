import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { EventEmitter } from '../../src/sensory-shell/EventEmitter';

describe('EventEmitter', () => {
  let emitter: EventEmitter;

  beforeEach(() => {
    emitter = new EventEmitter();
  });

  describe('Basic Event Handling', () => {
    it('should register and emit events', () => {
      const callback = jest.fn();
      emitter.on('test', callback);
      emitter.emit('test', 'data');
      
      expect(callback).toHaveBeenCalledWith('data');
    });

    it('should handle multiple listeners for the same event', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      emitter.on('test', callback1);
      emitter.on('test', callback2);
      emitter.emit('test', 'data');
      
      expect(callback1).toHaveBeenCalledWith('data');
      expect(callback2).toHaveBeenCalledWith('data');
    });

    it('should handle events with multiple arguments', () => {
      const callback = jest.fn();
      emitter.on('test', callback);
      emitter.emit('test', 'arg1', 'arg2', 'arg3');
      
      expect(callback).toHaveBeenCalledWith('arg1', 'arg2', 'arg3');
    });

    it('should handle events with no arguments', () => {
      const callback = jest.fn();
      emitter.on('test', callback);
      emitter.emit('test');
      
      expect(callback).toHaveBeenCalledWith();
    });

    it('should not call listeners for different events', () => {
      const callback = jest.fn();
      emitter.on('test1', callback);
      emitter.emit('test2', 'data');
      
      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('Event Removal', () => {
    it('should remove specific listener', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      emitter.on('test', callback1);
      emitter.on('test', callback2);
      emitter.off('test', callback1);
      emitter.emit('test', 'data');
      
      expect(callback1).not.toHaveBeenCalled();
      expect(callback2).toHaveBeenCalledWith('data');
    });

    it('should remove all listeners for an event', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      emitter.on('test', callback1);
      emitter.on('test', callback2);
      emitter.off('test');
      emitter.emit('test', 'data');
      
      expect(callback1).not.toHaveBeenCalled();
      expect(callback2).not.toHaveBeenCalled();
    });

    it('should handle removing non-existent listener gracefully', () => {
      const callback = jest.fn();
      
      expect(() => {
        emitter.off('test', callback);
      }).not.toThrow();
    });

    it('should handle removing from non-existent event gracefully', () => {
      expect(() => {
        emitter.off('nonexistent');
      }).not.toThrow();
    });
  });

  describe('Once Listeners', () => {
    it('should call once listener only once', () => {
      const callback = jest.fn();
      emitter.once('test', callback);
      
      emitter.emit('test', 'data1');
      emitter.emit('test', 'data2');
      
      expect(callback).toHaveBeenCalledTimes(1);
      expect(callback).toHaveBeenCalledWith('data1');
    });

    it('should remove once listener after first call', () => {
      const callback = jest.fn();
      emitter.once('test', callback);
      
      emitter.emit('test', 'data');
      
      // Verify listener was removed
      expect(emitter.listenerCount('test')).toBe(0);
    });

    it('should handle multiple once listeners', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      emitter.once('test', callback1);
      emitter.once('test', callback2);
      
      emitter.emit('test', 'data');
      
      expect(callback1).toHaveBeenCalledWith('data');
      expect(callback2).toHaveBeenCalledWith('data');
      expect(emitter.listenerCount('test')).toBe(0);
    });
  });

  describe('Listener Management', () => {
    it('should return correct listener count', () => {
      expect(emitter.listenerCount('test')).toBe(0);
      
      emitter.on('test', () => {});
      expect(emitter.listenerCount('test')).toBe(1);
      
      emitter.on('test', () => {});
      expect(emitter.listenerCount('test')).toBe(2);
    });

    it('should return correct listener count for non-existent event', () => {
      expect(emitter.listenerCount('nonexistent')).toBe(0);
    });

    it('should update listener count after removal', () => {
      const callback = jest.fn();
      emitter.on('test', callback);
      expect(emitter.listenerCount('test')).toBe(1);
      
      emitter.off('test', callback);
      expect(emitter.listenerCount('test')).toBe(0);
    });

    it('should return all event names', () => {
      emitter.on('event1', () => {});
      emitter.on('event2', () => {});
      emitter.on('event3', () => {});
      
      const eventNames = emitter.eventNames();
      expect(eventNames).toContain('event1');
      expect(eventNames).toContain('event2');
      expect(eventNames).toContain('event3');
      expect(eventNames).toHaveLength(3);
    });

    it('should return empty array when no events registered', () => {
      const eventNames = emitter.eventNames();
      expect(eventNames).toEqual([]);
    });
  });

  describe('Error Handling', () => {
    it('should handle listener errors gracefully', () => {
      const errorCallback = jest.fn(() => {
        throw new Error('Test error');
      });
      const normalCallback = jest.fn();
      
      emitter.on('test', errorCallback);
      emitter.on('test', normalCallback);
      
      expect(() => {
        emitter.emit('test', 'data');
      }).not.toThrow();
      
      expect(errorCallback).toHaveBeenCalled();
      expect(normalCallback).toHaveBeenCalled();
    });

    it('should continue executing other listeners after error', () => {
      const callback1 = jest.fn(() => {
        throw new Error('Error in callback1');
      });
      const callback2 = jest.fn();
      const callback3 = jest.fn();
      
      emitter.on('test', callback1);
      emitter.on('test', callback2);
      emitter.on('test', callback3);
      
      emitter.emit('test', 'data');
      
      expect(callback1).toHaveBeenCalled();
      expect(callback2).toHaveBeenCalled();
      expect(callback3).toHaveBeenCalled();
    });
  });

  describe('Edge Cases', () => {
    it('should handle undefined event names', () => {
      expect(() => {
        emitter.on(undefined as unknown as string, () => {});
      }).not.toThrow();
    });

    it('should handle null callbacks', () => {
      expect(() => {
        emitter.on('test', null as unknown as (...args: unknown[]) => void);
      }).not.toThrow();
    });

    it('should handle undefined callbacks', () => {
      expect(() => {
        emitter.on('test', undefined as unknown as (...args: unknown[]) => void);
      }).not.toThrow();
    });

    it('should handle emitting to event with no listeners', () => {
      expect(() => {
        emitter.emit('nonexistent', 'data');
      }).not.toThrow();
    });

    it('should handle complex event data', () => {
      const callback = jest.fn();
      const complexData = {
        user: { id: 123, name: 'Test' },
        metadata: { timestamp: Date.now() },
        array: [1, 2, 3]
      };
      
      emitter.on('test', callback);
      emitter.emit('test', complexData);
      
      expect(callback).toHaveBeenCalledWith(complexData);
    });

    it('should handle rapid event emissions', () => {
      const callback = jest.fn();
      emitter.on('test', callback);
      
      for (let i = 0; i < 1000; i++) {
        emitter.emit('test', i);
      }
      
      expect(callback).toHaveBeenCalledTimes(1000);
    });

    it('should handle listener that removes itself', () => {
      const callback = jest.fn(() => {
        emitter.off('test', callback);
      });
      
      emitter.on('test', callback);
      emitter.emit('test', 'data');
      
      expect(callback).toHaveBeenCalledTimes(1);
      expect(emitter.listenerCount('test')).toBe(0);
    });

    it('should handle listener that adds new listeners', () => {
      const newCallback = jest.fn();
      const callback = jest.fn(() => {
        emitter.on('test', newCallback);
      });
      
      emitter.on('test', callback);
      emitter.emit('test', 'data1');
      emitter.emit('test', 'data2');
      
      expect(callback).toHaveBeenCalledTimes(2);
      expect(newCallback).toHaveBeenCalledTimes(1);
      expect(newCallback).toHaveBeenCalledWith('data2');
    });
  });

  describe('Memory Management', () => {
    it('should clean up listeners properly', () => {
      const callbacks = [];
      
      // Add many listeners
      for (let i = 0; i < 100; i++) {
        const callback = jest.fn();
        callbacks.push(callback);
        emitter.on('test', callback);
      }
      
      expect(emitter.listenerCount('test')).toBe(100);
      
      // Remove all listeners
      emitter.off('test');
      
      expect(emitter.listenerCount('test')).toBe(0);
      expect(emitter.eventNames()).not.toContain('test');
    });

    it('should handle removing listeners during emission', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn(() => {
        emitter.off('test', callback1);
      });
      const callback3 = jest.fn();
      
      emitter.on('test', callback1);
      emitter.on('test', callback2);
      emitter.on('test', callback3);
      
      emitter.emit('test', 'data');
      
      expect(callback1).toHaveBeenCalled();
      expect(callback2).toHaveBeenCalled();
      expect(callback3).toHaveBeenCalled();
      expect(emitter.listenerCount('test')).toBe(2);
    });
  });
});
