import React, { useState, useEffect } from 'react';
import { useDefaultAgentImage } from '../../hooks/useAssetPath';

/**
 * Component for displaying agent images with proper asset path resolution
 * Handles both Electron and web contexts automatically
 */
const AgentImage = ({ 
  src, 
  alt = "Agent Image", 
  className = "", 
  fallbackSrc = null,
  ...props 
}) => {
  const defaultAgentImage = useDefaultAgentImage();
  const [resolvedSrc, setResolvedSrc] = useState(src);
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    const resolveImageSrc = async () => {
      setIsLoading(true);
      setHasError(false);

      try {
        let finalSrc = src;

        // Handle different image source types
        if (!src || src === '/knirv-logo.png' || src === './knirv-logo.png' || src === '/Agentify_logo_2.png' || src === './Agentify_logo_2.png') {
          // Use default agent image
          finalSrc = defaultAgentImage;
        } else if (typeof window !== 'undefined' && window.electronAPI && src === 'ELECTRON_ASSET:agent.png') {
          // Resolve the packaged default agent image.
          try {
            finalSrc = await window.electronAPI.getAssetPath('agent.png');
          } catch (error) {
            console.error('Failed to resolve asset path:', error);
            finalSrc = defaultAgentImage;
          }
        } else if (src.startsWith('http://') || src.startsWith('https://')) {
          // External URL - use as is
          finalSrc = src;
        } else if (src.startsWith('./') || src.startsWith('/')) {
          // Relative or absolute path - check if it's the logo
          if (src.includes('knirv-logo.png') || src.includes('Agentify_logo_2.png')) {
            finalSrc = defaultAgentImage;
          } else {
            finalSrc = src;
          }
        }

        setResolvedSrc(finalSrc);
      } catch (error) {
        console.error('Error resolving image source:', error);
        setResolvedSrc(fallbackSrc || defaultAgentImage);
        setHasError(true);
      } finally {
        setIsLoading(false);
      }
    };

    resolveImageSrc();
  }, [src, defaultAgentImage, fallbackSrc]);

  const handleImageLoad = () => {
    setIsLoading(false);
    setHasError(false);
  };

  const handleImageError = () => {
    setIsLoading(false);
    setHasError(true);
    
    // Try fallback or default image
    const fallback = fallbackSrc || defaultAgentImage;
    if (resolvedSrc !== fallback) {
      setResolvedSrc(fallback);
    }
  };

  if (isLoading) {
    return (
      <div 
        className={`bg-slate-700 animate-pulse flex items-center justify-center ${className}`}
        {...props}
      >
        <div className="text-slate-400 text-xs">Loading...</div>
      </div>
    );
  }

  return (
    <img
      src={resolvedSrc}
      alt={alt}
      className={className}
      onLoad={handleImageLoad}
      onError={handleImageError}
      {...props}
    />
  );
};

export default AgentImage;
