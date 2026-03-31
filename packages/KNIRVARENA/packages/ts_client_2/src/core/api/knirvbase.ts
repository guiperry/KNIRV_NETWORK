import { Request, Response } from 'express';
import type { DB, Options } from '@knirvcorp/knirvbase-ts';
import * as path from 'path';
import * as os from 'os';

// In-memory storage for DB instances per user/session
const dbInstances = new Map<string, DB>();

// Get App Data directory based on OS
function getAppDataDir(): string {
  const platform = os.platform();
  let baseDir: string;
  
  switch (platform) {
    case 'win32':
      baseDir = process.env.APPDATA || path.join(os.homedir(), 'AppData', 'Roaming');
      break;
    case 'darwin':
      baseDir = path.join(os.homedir(), 'Library', 'Application Support');
      break;
    default: // linux
      baseDir = process.env.XDG_DATA_HOME || path.join(os.homedir(), '.local', 'share');
      break;
  }
  
  return path.join(baseDir, 'KNIRV', 'KNIRVARENA', 'data');
}

// Get or create DB instance for a session
function getDBInstance(sessionId: string, dataDir?: string): DB {
  if (dbInstances.has(sessionId)) {
    return dbInstances.get(sessionId)!;
  }

  const defaultDataDir = getAppDataDir();
  const options: Options = {
    dataDir: dataDir || defaultDataDir,
    distributedEnabled: false,
    distributedNetworkID: undefined,
    distributedBootstrapPeers: undefined
  };

  const db = new DB(options);
  dbInstances.set(sessionId, db);
  return db;
}

/**
 * Initialize KNIRVBASE database for a session
 */
export const initializeDatabase = async (req: Request, res: Response) => {
  try {
    const { sessionId, dataDir, distributedEnabled, networkId, bootstrapPeers } = req.body;

    if (!sessionId) {
      return res.status(400).json({ error: 'sessionId is required' });
    }

    const options: Options = {
      dataDir: dataDir || getAppDataDir(),
      distributedEnabled: distributedEnabled || false,
      distributedNetworkID: networkId,
      distributedBootstrapPeers: bootstrapPeers
    };

    const db = new DB(options);
    await db.initialize();
    dbInstances.set(sessionId, db);

    res.json({ 
      success: true, 
      message: 'KNIRVBASE database initialized successfully',
      dataDir: options.dataDir
    });
  } catch (error) {
    console.error('Failed to initialize KNIRVBASE:', error);
    res.status(500).json({ 
      error: 'Failed to initialize KNIRVBASE',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Insert document into collection
 */
export const insertDocument = async (req: Request, res: Response) => {
  try {
    const { sessionId, collection, document } = req.body;

    if (!sessionId || !collection || !document) {
      return res.status(400).json({ error: 'sessionId, collection, and document are required' });
    }

    const db = getDBInstance(sessionId);
    const collectionObj = db.collection(collection);
    const result = await collectionObj.insert(document);

    res.json({ success: true, document: result });
  } catch (error) {
    console.error('Failed to insert document:', error);
    res.status(500).json({ 
      error: 'Failed to insert document',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Find document by ID
 */
export const findDocument = async (req: Request, res: Response) => {
  try {
    const { sessionId, collection, id } = req.params;

    if (!sessionId || !collection || !id) {
      return res.status(400).json({ error: 'sessionId, collection, and id are required' });
    }

    const db = getDBInstance(sessionId);
    const collectionObj = db.collection(collection);
    const document = await collectionObj.find(id);

    if (!document) {
      return res.status(404).json({ error: 'Document not found' });
    }

    res.json({ success: true, document });
  } catch (error) {
    console.error('Failed to find document:', error);
    res.status(500).json({ 
      error: 'Failed to find document',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Find all documents in collection
 */
export const findAllDocuments = async (req: Request, res: Response) => {
  try {
    const { sessionId, collection } = req.params;

    if (!sessionId || !collection) {
      return res.status(400).json({ error: 'sessionId and collection are required' });
    }

    const db = getDBInstance(sessionId);
    const collectionObj = db.collection(collection);
    const documents = await collectionObj.findAll();

    res.json({ success: true, documents });
  } catch (error) {
    console.error('Failed to find all documents:', error);
    res.status(500).json({ 
      error: 'Failed to find all documents',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Update document by ID
 */
export const updateDocument = async (req: Request, res: Response) => {
  try {
    const { sessionId, collection, id } = req.params;
    const { update } = req.body;

    if (!sessionId || !collection || !id || !update) {
      return res.status(400).json({ error: 'sessionId, collection, id, and update are required' });
    }

    const db = getDBInstance(sessionId);
    const collectionObj = db.collection(collection);
    const updatedCount = await collectionObj.update(id, update);

    res.json({ success: true, updatedCount });
  } catch (error) {
    console.error('Failed to update document:', error);
    res.status(500).json({ 
      error: 'Failed to update document',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Delete document by ID
 */
export const deleteDocument = async (req: Request, res: Response) => {
  try {
    const { sessionId, collection, id } = req.params;

    if (!sessionId || !collection || !id) {
      return res.status(400).json({ error: 'sessionId, collection, and id are required' });
    }

    const db = getDBInstance(sessionId);
    const collectionObj = db.collection(collection);
    const deletedCount = await collectionObj.delete(id);

    res.json({ success: true, deletedCount });
  } catch (error) {
    console.error('Failed to delete document:', error);
    res.status(500).json({ 
      error: 'Failed to delete document',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Get database information
 */
export const getDatabaseInfo = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;

    if (!sessionId) {
      return res.status(400).json({ error: 'sessionId is required' });
    }

    const db = getDBInstance(sessionId);
    const dataDir = getAppDataDir();

    res.json({ 
      success: true, 
      sessionId,
      dataDir,
      initialized: true
    });
  } catch (error) {
    console.error('Failed to get database info:', error);
    res.status(500).json({ 
      error: 'Failed to get database info',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Close database session
 */
export const closeDatabase = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;

    if (!sessionId) {
      return res.status(400).json({ error: 'sessionId is required' });
    }

    const db = dbInstances.get(sessionId);
    if (db) {
      await db.shutdown();
      dbInstances.delete(sessionId);
    }

    res.json({ success: true, message: 'Database session closed' });
  } catch (error) {
    console.error('Failed to close database:', error);
    res.status(500).json({ 
      error: 'Failed to close database',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};

/**
 * Get App Data directory path
 */
export const getAppDataPath = async (req: Request, res: Response) => {
  try {
    const appDataDir = getAppDataDir();
    res.json({ 
      success: true, 
      appDataDir,
      platform: os.platform(),
      homedir: os.homedir()
    });
  } catch (error) {
    console.error('Failed to get App Data path:', error);
    res.status(500).json({ 
      error: 'Failed to get App Data path',
      details: error instanceof Error ? error.message : String(error)
    });
  }
};