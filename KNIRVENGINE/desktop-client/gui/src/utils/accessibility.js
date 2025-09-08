// Accessibility utilities for the KNIRVENGINE

/**
 * Focus management utilities
 */
export class FocusManager {
  constructor() {
    this.focusStack = [];
    this.trapStack = [];
  }

  // Save current focus and set new focus
  saveFocus() {
    const activeElement = document.activeElement;
    this.focusStack.push(activeElement);
    return activeElement;
  }

  // Restore previously saved focus
  restoreFocus() {
    const previousFocus = this.focusStack.pop();
    if (previousFocus && previousFocus.focus) {
      previousFocus.focus();
    }
  }

  // Focus first focusable element in container
  focusFirst(container) {
    const focusableElements = this.getFocusableElements(container);
    if (focusableElements.length > 0) {
      focusableElements[0].focus();
    }
  }

  // Focus last focusable element in container
  focusLast(container) {
    const focusableElements = this.getFocusableElements(container);
    if (focusableElements.length > 0) {
      focusableElements[focusableElements.length - 1].focus();
    }
  }

  // Get all focusable elements in container
  getFocusableElements(container) {
    const focusableSelectors = [
      'button:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      'a[href]',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]'
    ].join(', ');

    return Array.from(container.querySelectorAll(focusableSelectors))
      .filter(element => {
        return element.offsetWidth > 0 && 
               element.offsetHeight > 0 && 
               !element.hasAttribute('hidden');
      });
  }

  // Trap focus within container
  trapFocus(container) {
    const focusableElements = this.getFocusableElements(container);
    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    const handleKeyDown = (e) => {
      if (e.key === 'Tab') {
        if (e.shiftKey) {
          // Shift + Tab
          if (document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          }
        } else {
          // Tab
          if (document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      }
    };

    container.addEventListener('keydown', handleKeyDown);
    this.trapStack.push({ container, handler: handleKeyDown });

    // Focus first element
    firstElement.focus();
  }

  // Remove focus trap
  removeFocusTrap() {
    const trap = this.trapStack.pop();
    if (trap) {
      trap.container.removeEventListener('keydown', trap.handler);
    }
  }
}

/**
 * Keyboard navigation utilities
 */
export const KeyboardNavigation = {
  // Handle arrow key navigation in grids
  handleGridNavigation(e, currentIndex, itemsPerRow, totalItems) {
    let newIndex = currentIndex;

    switch (e.key) {
      case 'ArrowUp':
        e.preventDefault();
        newIndex = Math.max(0, currentIndex - itemsPerRow);
        break;
      case 'ArrowDown':
        e.preventDefault();
        newIndex = Math.min(totalItems - 1, currentIndex + itemsPerRow);
        break;
      case 'ArrowLeft':
        e.preventDefault();
        newIndex = Math.max(0, currentIndex - 1);
        break;
      case 'ArrowRight':
        e.preventDefault();
        newIndex = Math.min(totalItems - 1, currentIndex + 1);
        break;
      case 'Home':
        e.preventDefault();
        newIndex = 0;
        break;
      case 'End':
        e.preventDefault();
        newIndex = totalItems - 1;
        break;
    }

    return newIndex;
  },

  // Handle list navigation
  handleListNavigation(e, currentIndex, totalItems) {
    let newIndex = currentIndex;

    switch (e.key) {
      case 'ArrowUp':
        e.preventDefault();
        newIndex = currentIndex > 0 ? currentIndex - 1 : totalItems - 1;
        break;
      case 'ArrowDown':
        e.preventDefault();
        newIndex = currentIndex < totalItems - 1 ? currentIndex + 1 : 0;
        break;
      case 'Home':
        e.preventDefault();
        newIndex = 0;
        break;
      case 'End':
        e.preventDefault();
        newIndex = totalItems - 1;
        break;
    }

    return newIndex;
  }
};

/**
 * ARIA utilities
 */
export const AriaUtils = {
  // Generate unique IDs for ARIA relationships
  generateId(prefix = 'aria') {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  },

  // Announce to screen readers
  announce(message, priority = 'polite') {
    const announcer = document.createElement('div');
    announcer.setAttribute('aria-live', priority);
    announcer.setAttribute('aria-atomic', 'true');
    announcer.className = 'sr-only';
    announcer.textContent = message;

    document.body.appendChild(announcer);

    setTimeout(() => {
      document.body.removeChild(announcer);
    }, 1000);
  },

  // Set ARIA expanded state
  setExpanded(element, expanded) {
    element.setAttribute('aria-expanded', expanded.toString());
  },

  // Set ARIA selected state
  setSelected(element, selected) {
    element.setAttribute('aria-selected', selected.toString());
  },

  // Set ARIA pressed state
  setPressed(element, pressed) {
    element.setAttribute('aria-pressed', pressed.toString());
  }
};

/**
 * Color contrast utilities
 */
export const ColorUtils = {
  // Calculate relative luminance
  getLuminance(r, g, b) {
    const [rs, gs, bs] = [r, g, b].map(c => {
      c = c / 255;
      return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
  },

  // Calculate contrast ratio
  getContrastRatio(color1, color2) {
    const l1 = this.getLuminance(...color1);
    const l2 = this.getLuminance(...color2);
    const lighter = Math.max(l1, l2);
    const darker = Math.min(l1, l2);
    return (lighter + 0.05) / (darker + 0.05);
  },

  // Check if contrast meets WCAG standards
  meetsWCAG(color1, color2, level = 'AA') {
    const ratio = this.getContrastRatio(color1, color2);
    return level === 'AAA' ? ratio >= 7 : ratio >= 4.5;
  }
};

/**
 * Screen reader utilities
 */
export const ScreenReaderUtils = {
  // Hide element from screen readers
  hideFromScreenReader(element) {
    element.setAttribute('aria-hidden', 'true');
  },

  // Show element to screen readers
  showToScreenReader(element) {
    element.removeAttribute('aria-hidden');
  },

  // Create screen reader only text
  createSROnlyText(text) {
    const span = document.createElement('span');
    span.className = 'sr-only';
    span.textContent = text;
    return span;
  }
};

/**
 * Global focus manager instance
 */
export const focusManager = new FocusManager();

/**
 * Initialize accessibility features
 */
export const initializeAccessibility = () => {
  // Add skip link
  const skipLink = document.createElement('a');
  skipLink.href = '#main-content';
  skipLink.textContent = 'Skip to main content';
  skipLink.className = 'sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:bg-blue-600 focus:text-white focus:px-4 focus:py-2 focus:rounded-lg';
  document.body.insertBefore(skipLink, document.body.firstChild);

  // Add screen reader only styles
  const style = document.createElement('style');
  style.textContent = `
    .sr-only {
      position: absolute;
      width: 1px;
      height: 1px;
      padding: 0;
      margin: -1px;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      white-space: nowrap;
      border: 0;
    }
    
    .focus\\:not-sr-only:focus {
      position: static;
      width: auto;
      height: auto;
      padding: 0.5rem 1rem;
      margin: 0;
      overflow: visible;
      clip: auto;
      white-space: normal;
    }
  `;
  document.head.appendChild(style);

  // Handle escape key globally
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      // Close any open modals or dropdowns
      const event = new CustomEvent('escape-pressed');
      document.dispatchEvent(event);
    }
  });

  // Announce page changes for SPAs
  let lastUrl = location.href;
  new MutationObserver(() => {
    const url = location.href;
    if (url !== lastUrl) {
      lastUrl = url;
      setTimeout(() => {
        const title = document.title;
        AriaUtils.announce(`Navigated to ${title}`, 'assertive');
      }, 100);
    }
  }).observe(document, { subtree: true, childList: true });
};
