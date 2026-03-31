/**
 * Keyboard Navigation Hook
 * Provides comprehensive keyboard navigation support for accessibility
 */

import { useEffect, useCallback, useRef, useState } from 'react';

interface KeyboardNavigationOptions {
  enabled?: boolean;
  focusableSelectors?: string[];
  skipSelectors?: string[];
  wrapAround?: boolean;
  onNavigate?: (element: HTMLElement, direction: 'next' | 'previous') => void;
  onActivate?: (element: HTMLElement) => void;
  onEscape?: () => void;
}

export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  altKey?: boolean;
  shiftKey?: boolean;
  metaKey?: boolean;
  action: () => void;
  description: string;
  category?: string;
}

const DEFAULT_FOCUSABLE_SELECTORS = [
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'a[href]',
  '[tabindex]:not([tabindex="-1"])',
  '[role="button"]:not([disabled])',
  '[role="link"]:not([disabled])',
  '[role="menuitem"]:not([disabled])',
  '[role="tab"]:not([disabled])',
  '[contenteditable="true"]'
];

export const useKeyboardNavigation = (
  containerRef: React.RefObject<HTMLElement>,
  options: KeyboardNavigationOptions = {}
) => {
  const {
    enabled = true,
    focusableSelectors = DEFAULT_FOCUSABLE_SELECTORS,
    skipSelectors = [],
    wrapAround = true,
    onNavigate,
    onActivate,
    onEscape
  } = options;

  const [currentFocusIndex, setCurrentFocusIndex] = useState(-1);
  const [focusableElements, setFocusableElements] = useState<HTMLElement[]>([]);

  // Update focusable elements when container changes
  const updateFocusableElements = useCallback(() => {
    if (!containerRef.current || !enabled) {
      setFocusableElements([]);
      return;
    }

    const selector = focusableSelectors.join(', ');
    const elements = Array.from(containerRef.current.querySelectorAll(selector)) as HTMLElement[];
    
    // Filter out elements that should be skipped
    const filteredElements = elements.filter(element => {
      // Skip hidden elements
      if (element.offsetParent === null) return false;
      
      // Skip elements with skip selectors
      if (skipSelectors.some(skipSelector => element.matches(skipSelector))) return false;
      
      // Skip elements inside disabled containers
      if (element.closest('[disabled], [aria-disabled="true"]')) return false;
      
      return true;
    });

    setFocusableElements(filteredElements);
  }, [containerRef, enabled, focusableSelectors, skipSelectors]);

  // Navigate to specific element
  const navigateToElement = useCallback((index: number) => {
    if (index < 0 || index >= focusableElements.length) return;
    
    const element = focusableElements[index];
    element.focus();
    setCurrentFocusIndex(index);
    
    if (onNavigate) {
      const direction = index > currentFocusIndex ? 'next' : 'previous';
      onNavigate(element, direction);
    }
  }, [focusableElements, currentFocusIndex, onNavigate]);

  // Navigate to next element
  const navigateNext = useCallback(() => {
    if (focusableElements.length === 0) return;
    
    let nextIndex = currentFocusIndex + 1;
    
    if (nextIndex >= focusableElements.length) {
      nextIndex = wrapAround ? 0 : focusableElements.length - 1;
    }
    
    navigateToElement(nextIndex);
  }, [currentFocusIndex, focusableElements.length, wrapAround, navigateToElement]);

  // Navigate to previous element
  const navigatePrevious = useCallback(() => {
    if (focusableElements.length === 0) return;
    
    let prevIndex = currentFocusIndex - 1;
    
    if (prevIndex < 0) {
      prevIndex = wrapAround ? focusableElements.length - 1 : 0;
    }
    
    navigateToElement(prevIndex);
  }, [currentFocusIndex, focusableElements.length, wrapAround, navigateToElement]);

  // Activate current element
  const activateCurrentElement = useCallback(() => {
    if (currentFocusIndex >= 0 && currentFocusIndex < focusableElements.length) {
      const element = focusableElements[currentFocusIndex];
      
      if (onActivate) {
        onActivate(element);
      } else {
        // Default activation behavior
        if (element.tagName === 'BUTTON' || element.getAttribute('role') === 'button') {
          element.click();
        } else if (element.tagName === 'A') {
          element.click();
        } else if (element.tagName === 'INPUT' && element.getAttribute('type') === 'checkbox') {
          element.click();
        }
      }
    }
  }, [currentFocusIndex, focusableElements, onActivate]);

  // Handle keyboard events
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;

    switch (event.key) {
      case 'Tab':
        event.preventDefault();
        if (event.shiftKey) {
          navigatePrevious();
        } else {
          navigateNext();
        }
        break;
        
      case 'ArrowDown':
      case 'ArrowRight':
        event.preventDefault();
        navigateNext();
        break;
        
      case 'ArrowUp':
      case 'ArrowLeft':
        event.preventDefault();
        navigatePrevious();
        break;
        
      case 'Enter':
      case ' ':
        event.preventDefault();
        activateCurrentElement();
        break;
        
      case 'Escape':
        if (onEscape) {
          event.preventDefault();
          onEscape();
        }
        break;
        
      case 'Home':
        event.preventDefault();
        navigateToElement(0);
        break;
        
      case 'End':
        event.preventDefault();
        navigateToElement(focusableElements.length - 1);
        break;
    }
  }, [enabled, navigateNext, navigatePrevious, activateCurrentElement, onEscape, navigateToElement, focusableElements.length]);

  // Set up event listeners
  useEffect(() => {
    if (!enabled || !containerRef.current) return;

    const container = containerRef.current;
    
    // Update focusable elements initially and on mutations
    updateFocusableElements();
    
    // Set up mutation observer to track DOM changes
    const observer = new MutationObserver(updateFocusableElements);
    observer.observe(container, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['disabled', 'tabindex', 'aria-disabled', 'hidden']
    });

    // Add keyboard event listener
    container.addEventListener('keydown', handleKeyDown);
    
    // Track focus changes
    const handleFocusIn = (event: FocusEvent) => {
      const target = event.target as HTMLElement;
      const index = focusableElements.indexOf(target);
      if (index >= 0) {
        setCurrentFocusIndex(index);
      }
    };
    
    container.addEventListener('focusin', handleFocusIn);

    return () => {
      observer.disconnect();
      container.removeEventListener('keydown', handleKeyDown);
      container.removeEventListener('focusin', handleFocusIn);
    };
  }, [enabled, containerRef, handleKeyDown, updateFocusableElements, focusableElements]);

  return {
    focusableElements,
    currentFocusIndex,
    navigateNext,
    navigatePrevious,
    navigateToElement,
    activateCurrentElement,
    updateFocusableElements
  };
};

export const useKeyboardShortcuts = (shortcuts: KeyboardShortcut[]) => {
  const [isEnabled, setIsEnabled] = useState(true);
  const shortcutsRef = useRef(shortcuts);

  // Update shortcuts ref when shortcuts change
  useEffect(() => {
    shortcutsRef.current = shortcuts;
  }, [shortcuts]);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!isEnabled) return;

    // Don't trigger shortcuts when typing in inputs
    const target = event.target as HTMLElement;
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.contentEditable === 'true') {
      return;
    }

    const matchingShortcut = shortcutsRef.current.find(shortcut => {
      return (
        shortcut.key.toLowerCase() === event.key.toLowerCase() &&
        !!shortcut.ctrlKey === event.ctrlKey &&
        !!shortcut.altKey === event.altKey &&
        !!shortcut.shiftKey === event.shiftKey &&
        !!shortcut.metaKey === event.metaKey
      );
    });

    if (matchingShortcut) {
      event.preventDefault();
      matchingShortcut.action();
    }
  }, [isEnabled]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return {
    isEnabled,
    setIsEnabled,
    shortcuts: shortcutsRef.current
  };
};

// Helper function to create keyboard shortcut descriptions
export const formatShortcut = (shortcut: KeyboardShortcut): string => {
  const parts: string[] = [];
  
  if (shortcut.ctrlKey) parts.push('Ctrl');
  if (shortcut.altKey) parts.push('Alt');
  if (shortcut.shiftKey) parts.push('Shift');
  if (shortcut.metaKey) parts.push('Cmd');
  
  parts.push(shortcut.key.toUpperCase());
  
  return parts.join(' + ');
};

// Helper function to group shortcuts by category
export const groupShortcutsByCategory = (shortcuts: KeyboardShortcut[]) => {
  return shortcuts.reduce((groups, shortcut) => {
    const category = shortcut.category || 'General';
    if (!groups[category]) {
      groups[category] = [];
    }
    groups[category].push(shortcut);
    return groups;
  }, {} as Record<string, KeyboardShortcut[]>);
};
