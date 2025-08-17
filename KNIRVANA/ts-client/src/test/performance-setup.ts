import { beforeAll, afterAll, beforeEach, afterEach } from 'vitest'

// Performance monitoring utilities
interface PerformanceMetrics {
  startTime: number
  endTime?: number
  duration?: number
  memoryUsage?: {
    initial: number
    final: number
    peak: number
  }
  renderCount?: number
  frameRate?: number
}

class PerformanceMonitor {
  private metrics: Map<string, PerformanceMetrics> = new Map()
  private observers: PerformanceObserver[] = []

  startMeasurement(name: string): void {
    const startTime = performance.now()
    const initialMemory = this.getMemoryUsage()
    
    this.metrics.set(name, {
      startTime,
      memoryUsage: {
        initial: initialMemory,
        final: 0,
        peak: initialMemory
      },
      renderCount: 0
    })

    // Start performance mark
    performance.mark(`${name}-start`)
  }

  endMeasurement(name: string): PerformanceMetrics | null {
    const metric = this.metrics.get(name)
    if (!metric) return null

    const endTime = performance.now()
    const finalMemory = this.getMemoryUsage()
    
    metric.endTime = endTime
    metric.duration = endTime - metric.startTime
    
    if (metric.memoryUsage) {
      metric.memoryUsage.final = finalMemory
    }

    // End performance mark and measure
    performance.mark(`${name}-end`)
    performance.measure(name, `${name}-start`, `${name}-end`)

    return metric
  }

  getMetrics(name: string): PerformanceMetrics | null {
    return this.metrics.get(name) || null
  }

  getAllMetrics(): Map<string, PerformanceMetrics> {
    return new Map(this.metrics)
  }

  clearMetrics(): void {
    this.metrics.clear()
    performance.clearMarks()
    performance.clearMeasures()
  }

  private getMemoryUsage(): number {
    // In Node.js environment, use process.memoryUsage()
    if (typeof process !== 'undefined' && process.memoryUsage) {
      return process.memoryUsage().heapUsed
    }
    
    // In browser environment, use performance.memory if available
    if (typeof performance !== 'undefined' && (performance as any).memory) {
      return (performance as any).memory.usedJSHeapSize
    }
    
    // Fallback
    return 0
  }

  startFrameRateMonitoring(name: string): void {
    let frameCount = 0
    let lastTime = performance.now()
    
    const measureFrameRate = () => {
      frameCount++
      const currentTime = performance.now()
      
      if (currentTime - lastTime >= 1000) { // Every second
        const metric = this.metrics.get(name)
        if (metric) {
          metric.frameRate = frameCount
          metric.renderCount = (metric.renderCount || 0) + frameCount
        }
        
        frameCount = 0
        lastTime = currentTime
      }
      
      requestAnimationFrame(measureFrameRate)
    }
    
    requestAnimationFrame(measureFrameRate)
  }

  observeLongTasks(): void {
    if (typeof PerformanceObserver !== 'undefined') {
      try {
        const observer = new PerformanceObserver((list) => {
          const entries = list.getEntries()
          entries.forEach((entry) => {
            if (entry.duration > 50) { // Tasks longer than 50ms
              console.warn(`Long task detected: ${entry.duration}ms`)
            }
          })
        })
        
        observer.observe({ entryTypes: ['longtask'] })
        this.observers.push(observer)
      } catch (e) {
        // PerformanceObserver might not be available in test environment
      }
    }
  }

  observeLayoutShifts(): void {
    if (typeof PerformanceObserver !== 'undefined') {
      try {
        const observer = new PerformanceObserver((list) => {
          const entries = list.getEntries()
          entries.forEach((entry: any) => {
            if (entry.value > 0.1) { // Significant layout shift
              console.warn(`Layout shift detected: ${entry.value}`)
            }
          })
        })
        
        observer.observe({ entryTypes: ['layout-shift'] })
        this.observers.push(observer)
      } catch (e) {
        // Layout shift observation might not be available
      }
    }
  }

  disconnect(): void {
    this.observers.forEach(observer => observer.disconnect())
    this.observers = []
  }
}

// Global performance monitor instance
const performanceMonitor = new PerformanceMonitor()

// Performance test utilities
export const performanceUtils = {
  monitor: performanceMonitor,
  
  // Measure function execution time
  measureFunction: async <T>(name: string, fn: () => Promise<T> | T): Promise<T> => {
    performanceMonitor.startMeasurement(name)
    const result = await fn()
    performanceMonitor.endMeasurement(name)
    return result
  },

  // Measure component render time
  measureRender: (name: string, renderFn: () => void): void => {
    performanceMonitor.startMeasurement(name)
    renderFn()
    // End measurement after next frame
    requestAnimationFrame(() => {
      performanceMonitor.endMeasurement(name)
    })
  },

  // Assert performance thresholds
  assertPerformance: (name: string, thresholds: {
    maxDuration?: number
    maxMemoryIncrease?: number
    minFrameRate?: number
  }): void => {
    const metrics = performanceMonitor.getMetrics(name)
    if (!metrics) {
      throw new Error(`No metrics found for "${name}"`)
    }

    if (thresholds.maxDuration && metrics.duration && metrics.duration > thresholds.maxDuration) {
      throw new Error(`Performance test "${name}" exceeded duration threshold: ${metrics.duration}ms > ${thresholds.maxDuration}ms`)
    }

    if (thresholds.maxMemoryIncrease && metrics.memoryUsage) {
      const memoryIncrease = metrics.memoryUsage.final - metrics.memoryUsage.initial
      if (memoryIncrease > thresholds.maxMemoryIncrease) {
        throw new Error(`Performance test "${name}" exceeded memory threshold: ${memoryIncrease} bytes > ${thresholds.maxMemoryIncrease} bytes`)
      }
    }

    if (thresholds.minFrameRate && metrics.frameRate && metrics.frameRate < thresholds.minFrameRate) {
      throw new Error(`Performance test "${name}" below frame rate threshold: ${metrics.frameRate} fps < ${thresholds.minFrameRate} fps`)
    }
  },

  // Generate performance report
  generateReport: (): string => {
    const allMetrics = performanceMonitor.getAllMetrics()
    let report = 'Performance Test Report\n'
    report += '========================\n\n'

    allMetrics.forEach((metrics, name) => {
      report += `Test: ${name}\n`
      report += `Duration: ${metrics.duration?.toFixed(2) || 'N/A'}ms\n`
      
      if (metrics.memoryUsage) {
        const increase = metrics.memoryUsage.final - metrics.memoryUsage.initial
        report += `Memory Usage: ${(increase / 1024 / 1024).toFixed(2)}MB increase\n`
      }
      
      if (metrics.frameRate) {
        report += `Frame Rate: ${metrics.frameRate} fps\n`
      }
      
      if (metrics.renderCount) {
        report += `Render Count: ${metrics.renderCount}\n`
      }
      
      report += '\n'
    })

    return report
  }
}

// Mock high-resolution timer for consistent testing
const mockPerformanceNow = (() => {
  let mockTime = 0
  return {
    now: () => mockTime,
    advance: (ms: number) => { mockTime += ms },
    reset: () => { mockTime = 0 }
  }
})()

beforeAll(() => {
  // Setup performance monitoring
  performanceMonitor.observeLongTasks()
  performanceMonitor.observeLayoutShifts()
  
  // Mock performance.now for deterministic testing
  const originalNow = performance.now
  performance.now = mockPerformanceNow.now
  
  // Store original for restoration
  ;(global as any).__originalPerformanceNow = originalNow
  
  // Set performance test environment
  process.env.NODE_ENV = 'test'
  process.env.PERFORMANCE_TEST = 'true'
})

afterAll(() => {
  // Restore original performance.now
  if ((global as any).__originalPerformanceNow) {
    performance.now = (global as any).__originalPerformanceNow
  }
  
  // Disconnect observers
  performanceMonitor.disconnect()
  
  // Generate final report
  const report = performanceUtils.generateReport()
  console.log(report)
})

beforeEach(() => {
  // Reset performance metrics for each test
  performanceMonitor.clearMetrics()
  mockPerformanceNow.reset()
})

afterEach(() => {
  // Clean up any test-specific performance monitoring
})

// Export for use in tests
export { mockPerformanceNow }
