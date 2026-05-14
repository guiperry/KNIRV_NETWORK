/**
 * User Feedback Collection Service
 * Collects and manages user feedback, bug reports, and feature requests
 */

interface FeedbackItem {
  id: string;
  type: 'bug' | 'feature' | 'improvement' | 'general';
  title: string;
  description: string;
  category: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  status: 'submitted' | 'acknowledged' | 'in-progress' | 'resolved' | 'closed';
  userAgent: string;
  url: string;
  timestamp: number;
  userId?: string;
  email?: string;
  attachments?: string[];
  metadata?: Record<string, unknown>;
}

interface FeedbackStats {
  total: number;
  byType: Record<string, number>;
  byPriority: Record<string, number>;
  byStatus: Record<string, number>;
  averageResponseTime: number;
  satisfactionScore: number;
}

export class UserFeedbackService {
  private feedback: FeedbackItem[] = [];
  private isInitialized = false;

  constructor() {
    this.initialize();
  }

  /**
   * Initialize the feedback service
   */
  private async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      // Load existing feedback from storage
      const stored = localStorage.getItem('knirv-feedback');
      if (stored) {
        this.feedback = JSON.parse(stored);
      }

      this.isInitialized = true;
      console.log('User feedback service initialized');
    } catch (error) {
      console.error('Failed to initialize feedback service:', error);
    }
  }

  /**
   * Submit new feedback
   */
  async submitFeedback(feedback: Omit<FeedbackItem, 'id' | 'timestamp' | 'userAgent' | 'url' | 'status'>): Promise<string> {
    const feedbackItem: FeedbackItem = {
      ...feedback,
      id: this.generateId(),
      timestamp: Date.now(),
      userAgent: navigator.userAgent,
      url: window.location.href,
      status: 'submitted'
    };

    this.feedback.push(feedbackItem);
    await this.saveFeedback();

    // Send to analytics if available
    this.trackFeedbackSubmission(feedbackItem);

    console.log('Feedback submitted:', feedbackItem.id);
    return feedbackItem.id;
  }

  /**
   * Submit bug report with automatic system info
   */
  async submitBugReport(
    title: string,
    description: string,
    stepsToReproduce?: string[],
    expectedBehavior?: string,
    actualBehavior?: string
  ): Promise<string> {
    const systemInfo = await this.collectSystemInfo();
    
    return this.submitFeedback({
      type: 'bug',
      title,
      description,
      category: 'bug-report',
      priority: 'medium',
      metadata: {
        stepsToReproduce,
        expectedBehavior,
        actualBehavior,
        systemInfo,
        consoleErrors: this.getRecentConsoleErrors(),
        performanceMetrics: this.getPerformanceSnapshot()
      }
    });
  }

  /**
   * Submit feature request
   */
  async submitFeatureRequest(
    title: string,
    description: string,
    useCase: string,
    priority: FeedbackItem['priority'] = 'medium'
  ): Promise<string> {
    return this.submitFeedback({
      type: 'feature',
      title,
      description,
      category: 'feature-request',
      priority,
      metadata: {
        useCase,
        userContext: this.getUserContext()
      }
    });
  }

  /**
   * Submit satisfaction rating
   */
  async submitSatisfactionRating(
    rating: number,
    category: string,
    comments?: string
  ): Promise<string> {
    return this.submitFeedback({
      type: 'general',
      title: `Satisfaction Rating: ${rating}/5`,
      description: comments || '',
      category: 'satisfaction',
      priority: 'low',
      metadata: {
        rating,
        category
      }
    });
  }

  /**
   * Get feedback by ID
   */
  getFeedback(id: string): FeedbackItem | undefined {
    return this.feedback.find(item => item.id === id);
  }

  /**
   * Get all feedback with optional filters
   */
  getAllFeedback(filters?: {
    type?: FeedbackItem['type'];
    status?: FeedbackItem['status'];
    priority?: FeedbackItem['priority'];
    category?: string;
    dateRange?: { start: number; end: number };
  }): FeedbackItem[] {
    let filtered = [...this.feedback];

    if (filters) {
      if (filters.type) {
        filtered = filtered.filter(item => item.type === filters.type);
      }
      if (filters.status) {
        filtered = filtered.filter(item => item.status === filters.status);
      }
      if (filters.priority) {
        filtered = filtered.filter(item => item.priority === filters.priority);
      }
      if (filters.category) {
        filtered = filtered.filter(item => item.category === filters.category);
      }
      if (filters.dateRange) {
        filtered = filtered.filter(item => 
          item.timestamp >= filters.dateRange!.start && 
          item.timestamp <= filters.dateRange!.end
        );
      }
    }

    return filtered.sort((a, b) => b.timestamp - a.timestamp);
  }

  /**
   * Get feedback statistics
   */
  getFeedbackStats(): FeedbackStats {
    const total = this.feedback.length;
    
    const byType = this.feedback.reduce((acc, item) => {
      acc[item.type] = (acc[item.type] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const byPriority = this.feedback.reduce((acc, item) => {
      acc[item.priority] = (acc[item.priority] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const byStatus = this.feedback.reduce((acc, item) => {
      acc[item.status] = (acc[item.status] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    const satisfactionRatings = this.feedback
      .filter(item => item.category === 'satisfaction' && item.metadata?.rating)
      .map(item => Number(item.metadata!.rating));

    const satisfactionScore = satisfactionRatings.length > 0
      ? satisfactionRatings.reduce((sum: number, rating: number) => sum + rating, 0) / satisfactionRatings.length
      : 0;

    return {
      total,
      byType,
      byPriority,
      byStatus,
      averageResponseTime: this.calculateAverageResponseTime(),
      satisfactionScore
    };
  }

  /**
   * Export feedback data
   */
  exportFeedback(format: 'json' | 'csv' = 'json'): string {
    if (format === 'csv') {
      return this.convertToCSV(this.feedback);
    }
    return JSON.stringify(this.feedback, null, 2);
  }

  /**
   * Clear all feedback (admin function)
   */
  async clearAllFeedback(): Promise<void> {
    this.feedback = [];
    await this.saveFeedback();
    console.log('All feedback cleared');
  }

  /**
   * Generate unique ID
   */
  private generateId(): string {
    return `feedback-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Save feedback to storage
   */
  private async saveFeedback(): Promise<void> {
    try {
      localStorage.setItem('knirv-feedback', JSON.stringify(this.feedback));
    } catch (error) {
      console.error('Failed to save feedback:', error);
    }
  }

  /**
   * Collect system information for bug reports
   */
  private async collectSystemInfo(): Promise<Record<string, unknown>> {
    const info: Record<string, unknown> = {
      userAgent: navigator.userAgent,
      platform: navigator.platform,
      language: navigator.language,
      cookieEnabled: navigator.cookieEnabled,
      onLine: navigator.onLine,
      screen: {
        width: screen.width,
        height: screen.height,
        colorDepth: screen.colorDepth
      },
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight
      },
      url: window.location.href,
      timestamp: new Date().toISOString()
    };

    // Add memory info if available
    if ('memory' in performance) {
      const perfWithMemory = performance as Performance & {
        memory?: {
          usedJSHeapSize: number;
          totalJSHeapSize: number;
          jsHeapSizeLimit: number;
        };
      };
      if (perfWithMemory.memory) {
        info.memory = {
          usedJSHeapSize: perfWithMemory.memory.usedJSHeapSize,
          totalJSHeapSize: perfWithMemory.memory.totalJSHeapSize,
          jsHeapSizeLimit: perfWithMemory.memory.jsHeapSizeLimit
        };
      }
    }

    // Add connection info if available
    if ('connection' in navigator) {
      const navWithConnection = navigator as Navigator & {
        connection?: {
          effectiveType: string;
          downlink?: number;
          rtt?: number;
        };
      };
      if (navWithConnection.connection) {
        info.connection = {
          effectiveType: navWithConnection.connection.effectiveType,
          downlink: navWithConnection.connection.downlink,
          rtt: navWithConnection.connection.rtt
        };
      }
    }

    return info;
  }

  /**
   * Get recent console errors
   */
  private getRecentConsoleErrors(): string[] {
    // This would require setting up error listeners
    // For now, return empty array
    return [];
  }

  /**
   * Get performance snapshot
   */
  private getPerformanceSnapshot(): Record<string, unknown> {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
    
    return {
      loadTime: navigation ? navigation.loadEventEnd - navigation.startTime : 0,
      domContentLoaded: navigation ? navigation.domContentLoadedEventEnd - navigation.startTime : 0,
      firstPaint: this.getFirstPaint(),
      memoryUsage: 'memory' in performance ?
        ((performance as Performance & { memory?: { usedJSHeapSize: number } }).memory?.usedJSHeapSize || 0) : 0
    };
  }

  /**
   * Get first paint timing
   */
  private getFirstPaint(): number {
    const paintEntries = performance.getEntriesByType('paint');
    const firstPaint = paintEntries.find(entry => entry.name === 'first-paint');
    return firstPaint ? firstPaint.startTime : 0;
  }

  /**
   * Get user context
   */
  private getUserContext(): Record<string, unknown> {
    return {
      sessionDuration: Date.now() - (performance.timing?.navigationStart || Date.now()),
      pageViews: this.getPageViews(),
      features: this.getUsedFeatures()
    };
  }

  /**
   * Get page views (mock implementation)
   */
  private getPageViews(): number {
    const stored = localStorage.getItem('knirv-page-views');
    return stored ? parseInt(stored, 10) : 1;
  }

  /**
   * Get used features (mock implementation)
   */
  private getUsedFeatures(): string[] {
    const stored = localStorage.getItem('knirv-used-features');
    return stored ? JSON.parse(stored) : [];
  }

  /**
   * Track feedback submission for analytics
   */
  private trackFeedbackSubmission(feedback: FeedbackItem): void {
    // This would integrate with analytics service
    console.log('Feedback tracked:', {
      type: feedback.type,
      category: feedback.category,
      priority: feedback.priority
    });
  }

  /**
   * Calculate average response time
   */
  private calculateAverageResponseTime(): number {
    const resolvedFeedback = this.feedback.filter(item => 
      item.status === 'resolved' || item.status === 'closed'
    );

    if (resolvedFeedback.length === 0) return 0;

    const totalTime = resolvedFeedback.reduce((sum, _item) => {
      // Mock response time calculation
      return sum + (24 * 60 * 60 * 1000); // 24 hours in ms
    }, 0);

    return totalTime / resolvedFeedback.length;
  }

  /**
   * Convert feedback to CSV format
   */
  private convertToCSV(data: FeedbackItem[]): string {
    if (data.length === 0) return '';

    const headers = ['ID', 'Type', 'Title', 'Description', 'Category', 'Priority', 'Status', 'Timestamp', 'URL'];
    const rows = data.map(item => [
      item.id,
      item.type,
      item.title,
      item.description,
      item.category,
      item.priority,
      item.status,
      new Date(item.timestamp).toISOString(),
      item.url
    ]);

    return [headers, ...rows]
      .map(row => row.map(cell => `"${cell}"`).join(','))
      .join('\n');
  }
}
