import { useState, useEffect } from 'react';
import { resolveImagePath } from '../utils/imageUrls';

/**
 * Custom hook to resolve asset paths for Electron or web context
 * @param {string} imagePath - The image path to resolve
 * @returns {string} The resolved image path
 */
export const useAssetPath = (imagePath) => {
  const [resolvedPath, setResolvedPath] = useState(imagePath);

  useEffect(() => {
    const resolvePath = async () => {
      try {
        const resolved = await resolveImagePath(imagePath);
        setResolvedPath(resolved);
      } catch (error) {
        console.error('Failed to resolve asset path:', error);
        // Keep the original path as fallback
        setResolvedPath(imagePath);
      }
    };

    resolvePath();
  }, [imagePath]);

  return resolvedPath;
};

/**
 * Custom hook specifically for the app logo
 * @returns {string} The resolved app logo path
 */
export const useAppLogo = () => {
  const [logoPath, setLogoPath] = useState('./Agentify_logo_2.png');

  useEffect(() => {
    const resolveLogo = async () => {
      try {
        if (typeof window !== 'undefined' && window.electronAPI) {
          const resolved = await window.electronAPI.getAssetPath('Agentify_logo_2.png');
          setLogoPath(resolved);
        }
      } catch (error) {
        console.error('Failed to resolve logo path:', error);
        // Keep the fallback path
      }
    };

    resolveLogo();
  }, []);

  return logoPath;
};

/**
 * Custom hook specifically for the default agent image
 * @returns {string} The resolved default agent image path
 */
export const useDefaultAgentImage = () => {
  const [imagePath, setImagePath] = useState('./Agentify_logo_2.png');

  useEffect(() => {
    const resolveImage = async () => {
      try {
        if (typeof window !== 'undefined' && window.electronAPI) {
          const resolved = await window.electronAPI.getAssetPath('Agentify_logo_2.png');
          setImagePath(resolved);
        }
      } catch (error) {
        console.error('Failed to resolve default agent image path:', error);
        // Keep the fallback path
      }
    };

    resolveImage();
  }, []);

  return imagePath;
};
