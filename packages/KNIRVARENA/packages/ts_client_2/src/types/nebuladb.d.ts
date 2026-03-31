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

  export interface QueryResult<T = Record<string, unknown>> {
    rows: T[];
    rowCount: number;
    fields: Array<{ name: string; type: string }>;
  }

  export interface Transaction {
    query<T = Record<string, unknown>>(sql: string, params?: unknown[]): Promise<QueryResult<T>>;
    commit(): Promise<void>;
    rollback(): Promise<void>;
  }

  export class NebulaDB {
    constructor(config?: NebulaDBConfig);
    
    connect(): Promise<void>;
    disconnect(): Promise<void>;
    
    query<T = Record<string, unknown>>(sql: string, params?: unknown[]): Promise<QueryResult<T>>;

    transaction(): Promise<Transaction>;

    defineSchema(schema: Record<string, unknown>): Promise<void>;
    
    // Collection-like methods
    collection(name: string): Collection;
    
    // Database management
    createDatabase(name: string): Promise<void>;
    dropDatabase(name: string): Promise<void>;
    listDatabases(): Promise<string[]>;
    
    // Schema management
    createTable(name: string, schema: Record<string, unknown>): Promise<void>;
    dropTable(name: string): Promise<void>;
    listTables(): Promise<string[]>;
  }

  export interface Collection {
    insert(document: Record<string, unknown>): Promise<Record<string, unknown>>;
    find(query?: Record<string, unknown>): Promise<Record<string, unknown>[]>;
    findOne(query?: Record<string, unknown>): Promise<Record<string, unknown> | null>;
    update(query: Record<string, unknown>, update: Record<string, unknown>): Promise<number>;
    delete(query: Record<string, unknown>): Promise<number>;
    count(query?: Record<string, unknown>): Promise<number>;
  }

  export default NebulaDB;
}
