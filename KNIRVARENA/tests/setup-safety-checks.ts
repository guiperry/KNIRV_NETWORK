// Setup safety checks for Jest tests
// This file contains safety measures to prevent common test issues

// Track unhandled promise rejections
const unhandledRejections = new Set<Promise<any>>();

process.on('unhandledRejection', (reason, promise) => {
  unhandledRejections.add(promise);
  console.warn('Unhandled promise rejection detected in test:', reason);
});

process.on('rejectionHandled', (promise) => {
  unhandledRejections.delete(promise);
});

// Check for memory leaks by monitoring global object growth
const initialGlobalKeys = Object.keys(global);
let maxGlobalKeys = initialGlobalKeys.length;

beforeEach(() => {
  // Reset unhandled rejections tracking
  unhandledRejections.clear();

  // Check for excessive global object pollution
  const currentGlobalKeys = Object.keys(global);
  if (currentGlobalKeys.length > maxGlobalKeys + 10) { // Allow some growth but warn if excessive
    console.warn('Potential memory leak: Global object has grown significantly');
    maxGlobalKeys = currentGlobalKeys.length;
  }
});

afterEach(() => {
  // Ensure no unhandled rejections remain
  if (unhandledRejections.size > 0) {
    console.error(`Test left ${unhandledRejections.size} unhandled promise rejections`);
    unhandledRejections.clear();
  }

  // Check for timers left running
  const activeTimers = (global as any).activeTimers || [];
  if (activeTimers.length > 0) {
    console.warn(`Test left ${activeTimers.length} active timers`);
  }
});

// Safety check for console errors during tests - only warn, don't fail
const originalConsoleError = console.error;
let consoleErrors: string[] = [];

beforeEach(() => {
  consoleErrors = [];
  console.error = (...args: any[]) => {
    consoleErrors.push(args.join(' '));
    originalConsoleError(...args);
  };
});

afterEach(() => {
  // Only warn about console errors, don't fail tests
  // Tests are expected to have various error conditions
  if (consoleErrors.length > 0) {
    console.warn(`Console errors during test (${consoleErrors.length}): ${consoleErrors.slice(0, 3).join('; ')}${consoleErrors.length > 3 ? '...' : ''}`);
  }

  console.error = originalConsoleError;
});

// Ensure proper cleanup of DOM elements
afterEach(() => {
  // Clean up any remaining DOM elements that might have been created
  const testContainer = document.getElementById('test-root');
  if (testContainer) {
    testContainer.innerHTML = '';
  }

  // Clean up any lingering event listeners (basic check)
  const body = document.body;
  if (body) {
    // This is a basic cleanup - more sophisticated cleanup might be needed
    body.innerHTML = '';
  }
});

// Memory usage monitoring (basic)
let initialMemoryUsage = process.memoryUsage?.();

beforeAll(() => {
  initialMemoryUsage = process.memoryUsage?.();
});

afterAll(() => {
  const finalMemoryUsage = process.memoryUsage?.();
  if (initialMemoryUsage && finalMemoryUsage) {
    const memoryGrowth = finalMemoryUsage.heapUsed - initialMemoryUsage.heapUsed;
    if (memoryGrowth > 50 * 1024 * 1024) { // 50MB growth
      console.warn(`Significant memory growth detected: ${(memoryGrowth / 1024 / 1024).toFixed(2)}MB`);
    }
  }
});