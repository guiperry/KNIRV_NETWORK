// Type declarations for nebuladb
declare module 'nebuladb' {
  export interface NebulaDBConfig {
    host?: string;
    port?: number;
    database?: string;
    username?: string;
    password?: string;
    ssl?: boolean;
    timeout?: number;
  }

  export interface QueryResult<T = any> {
    rows: T[];
    rowCount: number;
    fields: Array<{ name: string; type: string }>;
  }

  export interface Transaction {
    query<T = any>(sql: string, params?: any[]): Promise<QueryResult<T>>;
    commit(): Promise<void>;
    rollback(): Promise<void>;
  }

  export class NebulaDB {
    constructor(config?: NebulaDBConfig);
    
    connect(): Promise<void>;
    disconnect(): Promise<void>;
    
    query<T = any>(sql: string, params?: any[]): Promise<QueryResult<T>>;
    
    transaction(): Promise<Transaction>;
    
    defineSchema(schema: Record<string, any>): Promise<void>;
    
    // Collection-like methods
    collection(name: string): Collection;
    
    // Database management
    createDatabase(name: string): Promise<void>;
    dropDatabase(name: string): Promise<void>;
    listDatabases(): Promise<string[]>;
    
    // Schema management
    createTable(name: string, schema: Record<string, any>): Promise<void>;
    dropTable(name: string): Promise<void>;
    listTables(): Promise<string[]>;
  }

  export interface Collection {
    insert(document: Record<string, any>): Promise<any>;
    find(query?: Record<string, any>): Promise<any[]>;
    findOne(query?: Record<string, any>): Promise<any>;
    update(query: Record<string, any>, update: Record<string, any>): Promise<number>;
    delete(query: Record<string, any>): Promise<number>;
    count(query?: Record<string, any>): Promise<number>;
  }

  export default NebulaDB;
}
