import { useState, useCallback, useEffect } from 'react';

export const useKnirvMenu = () => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Add keyboard toggle for testing
  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      if (event.key === ' ' || event.key === 'Enter') {
        event.preventDefault();
        console.log('Keyboard toggle triggered');
        setIsExpanded(prev => !prev);
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => window.removeEventListener('keydown', handleKeyPress);
  }, []);

  const toggleExpansion = useCallback(() => {
    console.log('toggleExpansion called, current state:', isExpanded);
    setIsExpanded(prev => {
      console.log('Setting isExpanded from', prev, 'to', !prev);
      return !prev;
    });
  }, [isExpanded]);

  const expand = useCallback(() => {
    setIsExpanded(true);
  }, []);

  const collapse = useCallback(() => {
    setIsExpanded(false);
  }, []);

  return {
    isExpanded,
    toggleExpansion,
    expand,
    collapse
  };
};
