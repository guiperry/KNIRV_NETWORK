/**
 * Browser API Polyfills for Node.js
 * 
 * This file provides polyfills for browser APIs that libp2p expects
 * but are not available in Node.js 18.
 */

// CustomEvent polyfill
if (typeof globalThis.CustomEvent === 'undefined') {
  globalThis.CustomEvent = class CustomEvent extends Event {
    constructor(type, options = {}) {
      super(type, options);
      this.detail = options.detail || null;
    }
  };
}

// Event polyfill
if (typeof globalThis.Event === 'undefined') {
  globalThis.Event = class Event {
    constructor(type, options = {}) {
      this.type = type;
      this.bubbles = options.bubbles || false;
      this.cancelable = options.cancelable || false;
      this.composed = options.composed || false;
      this.defaultPrevented = false;
      this.eventPhase = 0;
      this.isTrusted = false;
      this.target = null;
      this.currentTarget = null;
      this.timeStamp = Date.now();
    }
    
    preventDefault() {
      this.defaultPrevented = true;
    }
    
    stopPropagation() {
      // No-op in Node.js
    }
    
    stopImmediatePropagation() {
      // No-op in Node.js
    }
  };
}

// EventTarget polyfill
if (typeof globalThis.EventTarget === 'undefined') {
  globalThis.EventTarget = class EventTarget {
    constructor() {
      this._listeners = new Map();
    }
    
    addEventListener(type, listener, options = {}) {
      if (!this._listeners.has(type)) {
        this._listeners.set(type, new Set());
      }
      this._listeners.get(type).add(listener);
    }
    
    removeEventListener(type, listener) {
      if (this._listeners.has(type)) {
        this._listeners.get(type).delete(listener);
      }
    }
    
    dispatchEvent(event) {
      if (this._listeners.has(event.type)) {
        const listeners = Array.from(this._listeners.get(event.type));
        for (const listener of listeners) {
          try {
            if (typeof listener === 'function') {
              listener.call(this, event);
            } else if (listener && typeof listener.handleEvent === 'function') {
              listener.handleEvent(event);
            }
          } catch (error) {
            console.error('Error in event listener:', error);
          }
        }
      }
      return !event.defaultPrevented;
    }
  };
}

// AbortController polyfill (if not available)
if (typeof globalThis.AbortController === 'undefined') {
  try {
    // Use require for synchronous loading in Node.js
    const { AbortController } = require('abort-controller');
    globalThis.AbortController = AbortController;
  } catch (error) {
    console.warn('[Polyfills] AbortController polyfill failed to load:', error.message);
  }
}

// TextEncoder/TextDecoder polyfills
if (typeof globalThis.TextEncoder === 'undefined') {
  try {
    // Use require for synchronous loading in Node.js
    const { TextEncoder, TextDecoder } = require('util');
    globalThis.TextEncoder = TextEncoder;
    globalThis.TextDecoder = TextDecoder;
  } catch (error) {
    console.warn('[Polyfills] TextEncoder/TextDecoder polyfill failed to load:', error.message);
  }
}

// Performance polyfill
if (typeof globalThis.performance === 'undefined') {
  globalThis.performance = {
    now: () => {
      const [seconds, nanoseconds] = process.hrtime();
      return seconds * 1000 + nanoseconds / 1000000;
    },
    timeOrigin: Date.now() - process.uptime() * 1000
  };
}

// Console polyfill enhancements
if (typeof globalThis.console === 'undefined') {
  globalThis.console = console;
}

// Promise.withResolvers polyfill (Node.js 22+ feature)
if (typeof Promise.withResolvers === 'undefined') {
  Promise.withResolvers = function() {
    let resolve, reject;
    const promise = new Promise((res, rej) => {
      resolve = res;
      reject = rej;
    });
    return { promise, resolve, reject };
  };
}

console.log('[Polyfills] Browser API polyfills loaded for Node.js compatibility');

export default {};
