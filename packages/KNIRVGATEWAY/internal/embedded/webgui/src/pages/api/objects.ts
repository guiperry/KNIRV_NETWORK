// Example: pages/api/objects.ts

import { NextApiRequest, NextApiResponse } from 'next';

// Use the environment variable for the backend URL
// Use gateway-proxied backend URL — the gateway serves these pages and
// proxies /api/v1/* to the real backend via backend.sock.
const BACKEND_URL = '/api/v1';

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ message: 'Method not allowed' });
  }

  try {
    // Fetch objects from the *backend* server API endpoint
    const response = await fetch(`${BACKEND_URL}/api/objects`, { // <-- Use BACKEND_URL
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
       // Log the error from the backend response for better debugging
       const errorText = await response.text();
       console.error(`Backend server error: ${response.status} ${response.statusText}`, errorText);
       throw new Error(`Backend server returned ${response.status}`);
    }

    const data = await response.json();
    return res.status(200).json(data);
  } catch (error: unknown) {
    console.error('Error fetching objects via Next.js API route:', error);
    const message = error instanceof Error ? error.message : 'Unknown error occurred';

    // Keep the mock data for development if you still want it as a fallback
    const mockObjects = [
      { id: 'mock-1', name: 'Mock Cube', object_type: 'glb' },
      { id: 'mock-2', name: 'Mock Sphere', object_type: 'glb' },
    ];
    // Indicate that mock data is being returned
    return res.status(500).json({ message: `Failed to fetch from backend: ${message}. Returning mock data.`, data: mockObjects });
    // Or just return an error:
    // return res.status(500).json({ message: `Failed to fetch objects from backend: ${error.message}` });
  }
}