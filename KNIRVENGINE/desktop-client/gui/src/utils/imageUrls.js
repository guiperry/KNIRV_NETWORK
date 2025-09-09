/**
 * Default agent image - KNIRV logo
 * @type {string}
 */
export const defaultAgentImage = (() => {
  // Check if we're running in Electron
  if (typeof window !== 'undefined' && window.electronAPI) {
    // In Electron, we'll get the asset path dynamically
    return 'ELECTRON_ASSET:knirv-logo.png';
  } else {
    // In web context (development)
    return './knirv-logo.png';
  }
})();

/**
 * Sample image URLs for agent creation
 * These images are used as default profile pictures when creating new agents
 * @type {string[]}
 */
export const sampleAgentImages = [
  (() => {
    if (typeof window !== 'undefined' && window.electronAPI) {
      return 'ELECTRON_ASSET:knirv-logo.png';
    } else {
      return './knirv-logo.png';
    }
  })(), // Default KNIRV logo as first option
  'https://images.pexels.com/photos/5380664/pexels-photo-5380664.jpeg?auto=compress&cs=tinysrgb&w=400',
  'https://images.pexels.com/photos/5380617/pexels-photo-5380617.jpeg?auto=compress&cs=tinysrgb&w=400',
  'https://images.pexels.com/photos/5380613/pexels-photo-5380613.jpeg?auto=compress&cs=tinysrgb&w=400',
  'https://images.pexels.com/photos/5380665/pexels-photo-5380665.jpeg?auto=compress&cs=tinysrgb&w=400',
  'https://images.pexels.com/photos/5380668/pexels-photo-5380668.jpeg?auto=compress&cs=tinysrgb&w=400',
  'https://images.pexels.com/photos/5380671/pexels-photo-5380671.jpeg?auto=compress&cs=tinysrgb&w=400'
];

/**
 * Get a random image URL from the sample images
 * @returns {string} A randomly selected image URL
 */
export const getRandomAgentImage = () => {
  const randomIndex = Math.floor(Math.random() * sampleAgentImages.length);
  return sampleAgentImages[randomIndex];
};

/**
 * Get the default agent image (KNIRV logo)
 * @returns {string} The default agent image URL
 */
export const getDefaultAgentImage = () => {
  return defaultAgentImage;
};

/**
 * Get the application logo (KNIRV logo)
 * @returns {string} The application logo URL
 */
export const getAppLogo = () => {
  return defaultAgentImage;
};

/**
 * Resolve asset path for Electron or web context
 * @param {string} imagePath - The image path to resolve
 * @returns {Promise<string>} The resolved image path
 */
export const resolveImagePath = async (imagePath) => {
  if (typeof window !== 'undefined' && window.electronAPI && imagePath.startsWith('ELECTRON_ASSET:')) {
    const filename = imagePath.replace('ELECTRON_ASSET:', '');
    try {
      return await window.electronAPI.getAssetPath(filename);
    } catch (error) {
      console.error('Failed to get asset path:', error);
      return './knirv-logo.png'; // Fallback
    }
  }
  return imagePath;
};