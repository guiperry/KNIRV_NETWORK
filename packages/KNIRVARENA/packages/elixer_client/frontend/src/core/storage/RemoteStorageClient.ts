/**
 * Remote Storage Client for KNIRVBASE
 * Provides browser-compatible storage by communicating with backend server
 */

export interface StorageOptions {
  sessionId: string;
  baseUrl?: string;
}

export interface Document {
  id: string;
  [key: string]: any;
}

export interface CollectionResponse {
  success: boolean;
  documents?: Document[];
  document?: Document;
  updatedCount?: number;
  deletedCount?: number;
  error?: string;
  details?: string;
}

export interface DatabaseInfo {
  success: boolean;
  sessionId: string;
  dataDir: string;
  initialized: boolean;
}

export interface InitializeResponse {
  success: boolean;
  message: string;
  dataDir: string;
  error?: string;
  details?: string;
}

export class RemoteStorageClient {
  private sessionId: string;
  private baseUrl: string;

  constructor(options: StorageOptions) {
    this.sessionId = options.sessionId;
    this.baseUrl = options.baseUrl || (typeof window !== 'undefined' ? 
      `${window.location.protocol}//${window.location.host.replace(':3000', ':3001')}` : 
      'http://localhost:3001');
  }

  private async makeRequest(endpoint: string, options: RequestInit = {}): Promise<any> {
    const url = `${this.baseUrl}/api/knirvbase${endpoint}`;
    
    const defaultHeaders = {
      'Content-Type': 'application/json',
    };

    const response = await fetch(url, {
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Initialize the remote database session
   */
  async initialize(dataDir?: string): Promise<InitializeResponse> {
    return this.makeRequest('/initialize', {
      method: 'POST',
      body: JSON.stringify({
        sessionId: this.sessionId,
        dataDir,
        distributedEnabled: false
      }),
    });
  }

  /**
   * Insert a document into a collection
   */
  async insert(collection: string, document: Document): Promise<Document> {
    const response: CollectionResponse = await this.makeRequest('/insert', {
      method: 'POST',
      body: JSON.stringify({
        sessionId: this.sessionId,
        collection,
        document,
      }),
    });

    if (!response.success || !response.document) {
      throw new Error(response.error || 'Failed to insert document');
    }

    return response.document;
  }

  /**
   * Find a document by ID
   */
  async find(collection: string, id: string): Promise<Document | null> {
    const response: CollectionResponse = await this.makeRequest(`/${this.sessionId}/${collection}/${id}`);

    if (!response.success) {
      if (response.error?.includes('not found')) {
        return null;
      }
      throw new Error(response.error || 'Failed to find document');
    }

    return response.document || null;
  }

  /**
   * Find all documents in a collection
   */
  async findAll(collection: string): Promise<Document[]> {
    const response: CollectionResponse = await this.makeRequest(`/${this.sessionId}/${collection}`);

    if (!response.success) {
      throw new Error(response.error || 'Failed to find all documents');
    }

    return response.documents || [];
  }

  /**
   * Update a document by ID
   */
  async update(collection: string, id: string, update: Partial<Document>): Promise<number> {
    const response: CollectionResponse = await this.makeRequest(`/${this.sessionId}/${collection}/${id}`, {
      method: 'PUT',
      body: JSON.stringify({
        update,
      }),
    });

    if (!response.success) {
      throw new Error(response.error || 'Failed to update document');
    }

    return response.updatedCount || 0;
  }

  /**
   * Delete a document by ID
   */
  async delete(collection: string, id: string): Promise<number> {
    const response: CollectionResponse = await this.makeRequest(`/${this.sessionId}/${collection}/${id}`, {
      method: 'DELETE',
    });

    if (!response.success) {
      throw new Error(response.error || 'Failed to delete document');
    }

    return response.deletedCount || 0;
  }

  /**
   * Get database information
   */
  async getInfo(): Promise<DatabaseInfo> {
    const response = await this.makeRequest(`/${this.sessionId}/info`);

    if (!response.success) {
      throw new Error(response.error || 'Failed to get database info');
    }

    return response;
  }

  /**
   * Close the database session
   */
  async close(): Promise<void> {
    await this.makeRequest(`/${this.sessionId}`, {
      method: 'DELETE',
    });
  }

  /**
   * Get App Data directory path (for information)
   */
  async getAppDataPath(): Promise<{ success: boolean; appDataDir: string; platform: string; homedir: string }> {
    return this.makeRequest('/appdata');
  }
}