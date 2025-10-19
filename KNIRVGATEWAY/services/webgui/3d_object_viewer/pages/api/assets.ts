// pages/api/assets.ts
import { NextApiRequest, NextApiResponse } from 'next';

// Use the environment variable for the backend URL
const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:3001'; // Default fallback

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    res.setHeader('Allow', ['GET']);
    return res.status(405).json({ message: 'Method Not Allowed' });
  }

  try {
    // Fetch assets from the backend server API endpoint
    const response = await fetch(`${BACKEND_URL}/api/assets`, { // <-- Use BACKEND_URL
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // Add any other necessary headers
      },
    });

    if (!response.ok) {
       // Log the error from the backend response for better debugging
       const errorText = await response.text();
       console.error(`Backend server error fetching assets: ${response.status} ${response.statusText}`, errorText);
       // Return a specific error status based on the backend response if desired, or a generic 502
       return res.status(response.status < 500 ? response.status : 502).json({ message: `Failed to fetch assets from backend: ${response.statusText}` });
    }

    const data = await response.json();
    return res.status(200).json(data);

  } catch (error: unknown) {
    console.error('Error fetching assets via Next.js API route:', error);
    // Return a 500 Internal Server Error if the fetch itself fails
    const message = error instanceof Error ? error.message : 'Unknown error occurred';
    return res.status(500).json({ message: `Failed to communicate with backend: ${message}` });
  }
}
