import React, { useRef, useEffect } from 'react';
import { useRouter } from 'next/router';

interface ViewportProps {
  modelId?: string;
}

/**
 * Viewport component that displays a 3D model using an iframe
 * This approach uses the /api/view/[id] endpoint which returns a complete HTML page
 * with Three.js setup for rendering the 3D model
 */
const Viewport: React.FC<ViewportProps> = ({ modelId }) => {
  const router = useRouter();
  const iframeRef = useRef<HTMLIFrameElement>(null);
  
  // Use the modelId prop if provided, otherwise use the router query
  const id = modelId || (router.query.id as string);
  
  // The source URL for the iframe - this endpoint returns a complete HTML page with Three.js
  const viewerUrl = id ? `/api/view/${id}` : '';

  // Add loading and error states
  const [isLoading, setIsLoading] = React.useState(true);
  const [hasError, setHasError] = React.useState(false);

  // Handle iframe load event
  const handleIframeLoad = () => {
    setIsLoading(false);
    setHasError(false);
  };

  // Handle iframe error event
  const handleIframeError = () => {
    setIsLoading(false);
    setHasError(true);
    console.error(`Failed to load 3D model with ID: ${id}`);
  };

  // Update iframe src when modelId changes
  useEffect(() => {
    if (iframeRef.current && id) {
      setIsLoading(true);
      iframeRef.current.src = viewerUrl;
    }
  }, [id, viewerUrl]);

  return (
    <div style={{ width: '100%', height: '100%', minHeight: '400px', position: 'relative' }}>
      {isLoading && (
        <div style={{ 
          position: 'absolute', 
          top: '50%', 
          left: '50%', 
          transform: 'translate(-50%, -50%)',
          color: '#fff',
          background: 'rgba(0,0,0,0.5)',
          padding: '10px 20px',
          borderRadius: '5px',
          zIndex: 10
        }}>
          Loading 3D model...
        </div>
      )}
      
      {hasError && (
        <div style={{ 
          position: 'absolute', 
          top: '50%', 
          left: '50%', 
          transform: 'translate(-50%, -50%)',
          color: '#fff',
          background: 'rgba(255,0,0,0.7)',
          padding: '10px 20px',
          borderRadius: '5px',
          zIndex: 10
        }}>
          Error loading 3D model. Please try another object.
        </div>
      )}
      
      {id ? (
        <iframe 
          ref={iframeRef}
          src={viewerUrl}
          style={{ 
            width: '100%', 
            height: '100%', 
            border: 'none',
            minHeight: '400px'
          }}
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          title="3D Model Viewer"
          allowFullScreen
        />
      ) : (
        <div style={{ 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          height: '100%',
          color: '#888',
          background: '#0a0e17',
          minHeight: '400px'
        }}>
          Select an object to view
        </div>
      )}
    </div>
  );
};

export default Viewport;