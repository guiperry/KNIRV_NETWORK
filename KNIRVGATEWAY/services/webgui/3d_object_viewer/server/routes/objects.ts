// server/routes/objects.ts
import * as http from 'http';
import { objects3D } from '../assets';
import { logger } from '../utils';

export function handleObjectsRequest(req: http.IncomingMessage, res: http.ServerResponse): void {
  // Set CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  if (req.method !== 'GET') {
    res.writeHead(405, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Method not allowed' }));
    return;
  }

  try {
    // Convert objects3D object to array
    const objectsArray = Object.values(objects3D).map(obj => ({
      id: obj.id,
      name: obj.name,
      object_type: obj.object_type,
      size: obj.size,
      last_modified: obj.last_modified
    }));

    logger.log(`Returning ${objectsArray.length} objects`);
    
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(objectsArray));
  } catch (error) {
    logger.error(`Error handling objects request: ${error}`);
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Internal server error' }));
  }
}

export function handleObjectRequest(req: http.IncomingMessage, res: http.ServerResponse, id: string): void {
  // Set CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  if (req.method !== 'GET') {
    res.writeHead(405, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Method not allowed' }));
    return;
  }

  try {
    const object = objects3D[id];
    
    if (!object) {
      logger.log(`Object with ID ${id} not found`);
      res.writeHead(404, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Object not found' }));
      return;
    }

    logger.log(`Returning object: ${object.name}`);
    
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(object));
  } catch (error) {
    logger.error(`Error handling object request: ${error}`);
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Internal server error' }));
  }
}