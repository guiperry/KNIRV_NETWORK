// /home/gperry/Documents/GitHub/3dObjectViewer_TS/pages/index.tsx
import React, { useState, useEffect } from 'react';
import Link from 'next/link'; // Optional: if you want links

// Define the expected structure of an object from your API
interface DisplayObject {
  id: string;
  name: string;
  object_type: string;
  // Add other fields you might want to display
}

export default function HomePage() {
  const [objects, setObjects] = useState<DisplayObject[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Fetch the list of objects from your Next.js API route
    // This route will internally fetch from your backend server
    fetch('/api/objects') // Fetch from the Next.js API route
      .then(res => {
        if (!res.ok) {
          throw new Error(`Failed to fetch objects: ${res.statusText}`);
        }
        return res.json();
      })
      .then((data: DisplayObject[]) => {
        setObjects(data);
        setIsLoading(false);
      })
      .catch(err => {
        console.error("Error fetching objects:", err);
        setError(err.message || 'Failed to load objects.');
        setIsLoading(false);
      });
  }, []); // Empty dependency array means this runs once on mount

  return (
    <div>
      <h1>Welcome to the 3D Object Viewer</h1>

      <h2>Available Objects:</h2>
      {isLoading && <p>Loading objects...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
      {!isLoading && !error && (
        <ul>
          {objects.length > 0 ? (
            objects.map(obj => (
              <li key={obj.id}>
                {/* Optional: Link to a viewer page */}
                <Link href={`/view/${obj.id}`}>
                  {obj.name} ({obj.object_type})
                </Link>
                {/* Or just display info */}
                {/* {obj.name} ({obj.object_type}) */}
              </li>
            ))
          ) : (
            <p>No objects found.</p>
          )}
        </ul>
      )}

      {/* You can add links to other parts of your app if needed */}
      {/* <Link href="/some-other-page">Go to Other Page</Link> */}
    </div>
  );
}
