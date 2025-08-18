import React, { useEffect, useState } from 'react';
import { X, ArrowLeft, ArrowRight, Play, Pause, SkipForward } from 'lucide-react';

const OnboardingModal = ({
  step,
  isVisible,
  onNext,
  onPrevious,
  onSkip,
  onComplete,
  isAutoMode,
  onToggleAutoMode,
  totalSteps
}) => {
  const [position, setPosition] = useState({ top: 0, left: 0, height: 300, width: 400 });
  const [arrowDirection, setArrowDirection] = useState('bottom');

  // Store current highlighted element to manage highlighting properly
  const [currentHighlightedElement, setCurrentHighlightedElement] = useState(null);
  const [highlightTimer, setHighlightTimer] = useState(null);

  // Cleanup function to remove highlights
  const cleanupHighlight = (element) => {
    if (element) {
      console.log('Onboarding: Cleaning up highlight from element:', element);
      element.classList.remove('onboarding-highlight');

      // Remove inline styles we added
      element.style.border = '';
      element.style.borderRadius = '';
      element.style.boxShadow = '';

      // Reset any style changes we made
      if (element.style.position === 'relative' && element.dataset.originalPosition === 'static') {
        element.style.position = '';
        delete element.dataset.originalPosition;
      }
    }
  };

  // Effect to handle highlighting when step changes
  useEffect(() => {
    // Clear any existing timer
    if (highlightTimer) {
      clearTimeout(highlightTimer);
      setHighlightTimer(null);
    }

    // Clean up previous highlight immediately when step changes
    if (currentHighlightedElement) {
      cleanupHighlight(currentHighlightedElement);
      setCurrentHighlightedElement(null);
    }

    if (!isVisible || !step?.target) {
      return;
    }

    const updatePosition = () => {
      // Try multiple selectors if provided (comma-separated)
      const selectors = step.target.split(',').map(s => s.trim());
      let targetElement = null;

      for (const selector of selectors) {
        try {
          targetElement = document.querySelector(selector);
          if (targetElement) break;
        } catch (error) {
          console.warn(`Invalid CSS selector in onboarding: "${selector}"`, error);
          // Continue to next selector
        }
      }

      if (!targetElement) {
        console.warn(`Onboarding target not found for any selector: ${step.target}`);
        // Fallback to body for positioning
        targetElement = document.body;
      }

      const rect = targetElement.getBoundingClientRect();

      // Responsive modal sizing
      const isMobile = window.innerWidth <= 768;
      const isTablet = window.innerWidth <= 1024;

      const modalWidth = isMobile ? Math.min(350, window.innerWidth - 40) : isTablet ? 380 : 400;
      const arrowSize = 20;
      const padding = isMobile ? 10 : 20;
      const minModalHeight = isMobile ? 180 : 200;
      const maxModalHeight = Math.min(isMobile ? 500 : 600, window.innerHeight - (isMobile ? 60 : 100));

      // Calculate dynamic modal height based on content
      const baseHeight = isMobile ? 250 : 300;
      const featuresHeight = (step.features?.length || 0) * (isMobile ? 20 : 25);
      const descriptionHeight = Math.max(isMobile ? 40 : 60, (step.description?.length || 0) / (isMobile ? 3 : 2));
      let modalHeight = Math.min(maxModalHeight, Math.max(minModalHeight, baseHeight + featuresHeight + descriptionHeight));

      let top, left, arrow;

      // Determine best position based on available space
      const spaceAbove = rect.top;
      const spaceBelow = window.innerHeight - rect.bottom;
      const spaceLeft = rect.left;
      const spaceRight = window.innerWidth - rect.right;

      // Prioritize positioning that keeps modal fully in viewport
      if (spaceBelow >= modalHeight + arrowSize + padding) {
        // Position below
        top = rect.bottom + arrowSize + padding;
        left = Math.max(padding, Math.min(rect.left, window.innerWidth - modalWidth - padding));
        arrow = 'top';
      } else if (spaceAbove >= modalHeight + arrowSize + padding) {
        // Position above
        top = rect.top - modalHeight - arrowSize - padding;
        left = Math.max(padding, Math.min(rect.left, window.innerWidth - modalWidth - padding));
        arrow = 'bottom';
      } else if (spaceRight >= modalWidth + arrowSize + padding) {
        // Position to the right
        top = Math.max(padding, Math.min(rect.top, window.innerHeight - modalHeight - padding));
        left = rect.right + arrowSize + padding;
        arrow = 'left';
      } else if (spaceLeft >= modalWidth + arrowSize + padding) {
        // Position to the left
        top = Math.max(padding, Math.min(rect.top, window.innerHeight - modalHeight - padding));
        left = rect.left - modalWidth - arrowSize - padding;
        arrow = 'right';
      } else {
        // Fallback: position with maximum available space and adjust height if needed
        const availableHeight = window.innerHeight - 2 * padding;
        modalHeight = Math.min(modalHeight, availableHeight);

        // Center on screen
        top = Math.max(padding, (window.innerHeight - modalHeight) / 2);
        left = Math.max(padding, (window.innerWidth - modalWidth) / 2);
        arrow = 'none';
      }

      // Final viewport boundary checks
      top = Math.max(padding, Math.min(top, window.innerHeight - modalHeight - padding));
      left = Math.max(padding, Math.min(left, window.innerWidth - modalWidth - padding));

      setPosition({ top, left, height: modalHeight, width: modalWidth });
      setArrowDirection(arrow);
    };

    updatePosition();
    window.addEventListener('resize', updatePosition);
    window.addEventListener('scroll', updatePosition);

    return () => {
      window.removeEventListener('resize', updatePosition);
      window.removeEventListener('scroll', updatePosition);
    };
  }, [step, isVisible, currentHighlightedElement]);

  // Separate effect for managing highlighting to avoid conflicts
  useEffect(() => {
    if (!isVisible || !step?.target) {
      return;
    }

    // Delay highlighting to ensure DOM is ready and navigation is complete
    const timer = setTimeout(() => {
      // Try multiple selectors if provided (comma-separated)
      const selectors = step.target.split(',').map(s => s.trim());
      let targetElement = null;

      for (const selector of selectors) {
        try {
          targetElement = document.querySelector(selector);
          console.log(`Onboarding: Trying selector "${selector}":`, targetElement);
          if (targetElement) break;
        } catch (error) {
          console.warn(`Invalid CSS selector in onboarding: "${selector}"`, error);
          // Continue to next selector
        }
      }

      if (!targetElement) {
        console.warn(`Onboarding: No element found for any selector in: ${step.target}`);
        console.log('Available elements with data-page:', document.querySelectorAll('[data-page]'));
        console.log('Available elements with main-header class:', document.querySelectorAll('.main-header'));
        console.log('All buttons:', document.querySelectorAll('button'));
        console.log('All navigation elements:', document.querySelectorAll('nav, [role="navigation"]'));

        // Fallback: try to find any reasonable element to highlight
        const fallbackSelectors = [
          'h1', // Main title
          '.main-header', // Header area
          'nav', // Navigation
          '[role="navigation"]', // Navigation role
          'button[data-page]', // Any navigation button
          'button' // Any button
        ];

        for (const fallbackSelector of fallbackSelectors) {
          targetElement = document.querySelector(fallbackSelector);
          if (targetElement) {
            console.log(`Onboarding: Using fallback selector "${fallbackSelector}":`, targetElement);
            break;
          }
        }
      }

      // Only highlight if we found a valid target (not body fallback)
      if (targetElement && targetElement !== document.body) {
        // Remove any existing highlights from other elements first
        document.querySelectorAll('.onboarding-highlight').forEach(el => {
          if (el !== targetElement) {
            cleanupHighlight(el);
          }
        });

        // Store original position if we need to modify it
        const computedStyle = window.getComputedStyle(targetElement);
        if (computedStyle.position === 'static') {
          targetElement.dataset.originalPosition = 'static';
          targetElement.style.position = 'relative';
        }

        // Add highlight with enhanced visibility
        targetElement.classList.add('onboarding-highlight');

        // Also add a very visible border as backup
        targetElement.style.border = '4px solid #3b82f6';
        targetElement.style.borderRadius = '8px';
        targetElement.style.boxShadow = '0 0 20px rgba(59, 130, 246, 0.8)';

        setCurrentHighlightedElement(targetElement);

        // Debug logging
        console.log(`Onboarding: Highlighted element for step ${step.stepNumber}:`, {
          selector: step.target,
          element: targetElement,
          className: targetElement.className,
          rect: targetElement.getBoundingClientRect()
        });

        // Scroll target into view if needed, but only if it's not already visible
        const targetRect = targetElement.getBoundingClientRect();
        const isTargetVisible = targetRect.top >= 0 && targetRect.bottom <= window.innerHeight;

        if (!isTargetVisible) {
          targetElement.scrollIntoView({
            behavior: 'smooth',
            block: 'center',
            inline: 'center'
          });
        }
      }
    }, 300); // Increased delay to ensure navigation and DOM updates are complete

    setHighlightTimer(timer);

    // Don't cleanup on dependency change - let the next effect handle it
    return () => {
      // Only clear the timer, don't remove highlights here
      clearTimeout(timer);
    };
  }, [step?.target, isVisible]);

  // Cleanup effect when component unmounts or onboarding ends
  useEffect(() => {
    return () => {
      // Clear any pending timer
      if (highlightTimer) {
        clearTimeout(highlightTimer);
      }

      // Clean up all highlights when component unmounts
      document.querySelectorAll('.onboarding-highlight').forEach(el => {
        cleanupHighlight(el);
      });
    };
  }, [highlightTimer]);

  if (!isVisible || !step) return null;

  const isFirstStep = step.stepNumber === 1;
  const isLastStep = step.stepNumber === totalSteps;

  return (
    <>
      {/* Overlay */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-40" />
      
      {/* Modal */}
      <div
        className={`fixed z-50 bg-slate-800 border border-slate-600 rounded-lg shadow-2xl transition-all duration-300 flex flex-col ${
          arrowDirection !== 'none' ? 'onboarding-modal-with-arrow' : ''
        }`}
        style={{
          top: `${position.top}px`,
          left: `${position.left}px`,
          height: `${position.height}px`,
          width: `${position.width || (window.innerWidth <= 768 ? Math.min(350, window.innerWidth - 40) : 400)}px`,
          maxHeight: `${Math.min(window.innerWidth <= 768 ? 500 : 600, window.innerHeight - (window.innerWidth <= 768 ? 60 : 100))}px`,
          '--arrow-direction': arrowDirection
        }}
      >
        {/* Arrow */}
        {arrowDirection !== 'none' && (
          <div className={`onboarding-arrow onboarding-arrow-${arrowDirection}`} />
        )}

        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-600">
          <div className="flex items-center space-x-3">
            <div className="flex items-center space-x-2">
              <span className="text-blue-400 font-semibold">
                Step {step.stepNumber} of {totalSteps}
              </span>
              <div className="flex space-x-1">
                {Array.from({ length: totalSteps }, (_, i) => (
                  <div
                    key={i}
                    className={`w-2 h-2 rounded-full ${
                      i < step.stepNumber ? 'bg-blue-400' : 'bg-slate-600'
                    }`}
                  />
                ))}
              </div>
            </div>
          </div>
          <button
            onClick={onSkip}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content - Scrollable */}
        <div className="flex-1 overflow-y-auto p-4">
          <h3 className="text-lg font-semibold text-white mb-2">
            {step.title}
          </h3>
          <p className="text-slate-300 text-sm mb-4 leading-relaxed">
            {step.description}
          </p>

          {step.features && (
            <div className="mb-4">
              <h4 className="text-sm font-medium text-blue-400 mb-2">Key Features:</h4>
              <ul className="text-xs text-slate-400 space-y-1">
                {step.features.map((feature, index) => (
                  <li key={index} className="flex items-start">
                    <span className="text-blue-400 mr-2">•</span>
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {/* Footer - Fixed at bottom */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 border-t border-slate-600">
          <div className="flex items-center space-x-2">
            <button
              onClick={onToggleAutoMode}
              className={`flex items-center space-x-1 px-2 py-1 rounded text-xs transition-colors ${
                isAutoMode 
                  ? 'bg-blue-600 text-white' 
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {isAutoMode ? <Pause size={12} /> : <Play size={12} />}
              <span>{isAutoMode ? 'Auto' : 'Manual'}</span>
            </button>
          </div>

          <div className="flex items-center space-x-2">
            {!isFirstStep && (
              <button
                onClick={onPrevious}
                className="flex items-center space-x-1 px-3 py-1 bg-slate-700 text-slate-300 rounded hover:bg-slate-600 transition-colors text-sm"
              >
                <ArrowLeft size={14} />
                <span>Back</span>
              </button>
            )}
            
            {isLastStep ? (
              <button
                onClick={onComplete}
                className="flex items-center space-x-1 px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 transition-colors text-sm"
              >
                <span>Complete</span>
              </button>
            ) : (
              <button
                onClick={onNext}
                className="flex items-center space-x-1 px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors text-sm"
              >
                <span>Next</span>
                <ArrowRight size={14} />
              </button>
            )}
          </div>
        </div>
      </div>
    </>
  );
};

export default OnboardingModal;
