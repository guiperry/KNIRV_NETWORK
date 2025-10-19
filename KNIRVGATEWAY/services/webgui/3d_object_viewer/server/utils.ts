// utils.ts - Utility classes and functions

// Mutex class to simulate Go's sync.Mutex (basic implementation for locking)
export class Mutex {
    private locked = false;
    private waiting: (() => void)[] = [];

    async lock(): Promise<void> {
        return new Promise<void>(resolve => {
            if (!this.locked) {
                this.locked = true;
                resolve();
            } else {
                this.waiting.push(resolve);
            }
        });
    }

    unlock(): void {
        if (!this.locked) {
            return; // Or throw error, depending on desired behavior
        }
        this.locked = false;
        const nextResolve = this.waiting.shift();
        if (nextResolve) {
            this.locked = true; // Lock before resolving the next waiter
            nextResolve();
        }
    }
}

// Custom logger for server-side logging (adapted from GUILogger)
export class Logger {
    private buffer: string[] = [];
    private mutex: Mutex = new Mutex();

    // Log adds a message to the logger
    log(message: string): void {
        const timestamp = new Date().toLocaleTimeString('en-US', { hour12: false });
        const logMessage = `>>> [${timestamp}] ${message}`;

        console.log(logMessage); // Print to console for debugging

        // Store in buffer with mutex protection
        Promise.resolve().then(async () => {
            await this.mutex.lock();
            try {
                this.buffer.push(logMessage);
                if (this.buffer.length > 100) {
                    this.buffer = this.buffer.slice(this.buffer.length - 100);
                }
            } finally {
                this.mutex.unlock();
            }
        });
    }
    
    // Warning log method
    warn(message: string): void {
        const warningMessage = `WARNING: ${message}`;
        this.log(warningMessage);
        console.warn(warningMessage);
    }
    
    // Error log method
    error(message: string): void {
        const errorMessage = `ERROR: ${message}`;
        this.log(errorMessage);
        console.error(errorMessage);
    }

    // Get all logs
    getLogs(): string[] {
        return [...this.buffer];
    }
}

// Create and export a singleton logger instance
export const logger = new Logger();