import { useState, useCallback } from 'react';

export const useKnirvMenu = () => {
  const [isExpanded, setIsExpanded] = useState(false);
  
  // Add debug logging
  useEffect(() => {
    console.log('useKnirvMenu hook initialized, isExpanded:', isExpanded);
  }, []);

  const toggleExpansion = useCallback(() => {
    setIsExpanded(prev => {
      const newValue = !prev;
      console.log('toggleExpansion called, changing from', prev, 'to', newValue);
      return newValue;
    });
  }, []);

  const expand = useCallback(() => {
    console.log('expand called');
    setIsExpanded(true);
  }, []);

  const collapse = useCallback(() => {
    console.log('collapse called');
    setIsExpanded(false);
  }, []);

  return {
    isExpanded,
    toggleExpansion,
    expand,
    collapse
  };
};
