import React, { useEffect, useRef } from 'react';
import { X } from 'lucide-react';
import { focusManager, AriaUtils } from '../../utils/accessibility';

const AccessibleModal = ({ 
  isOpen, 
  onClose, 
  title, 
  children, 
  size = 'default',
  closeOnEscape = true,
  closeOnOverlayClick = true,
  initialFocus = null,
  className = '',
  ariaDescribedBy = null
}) => {
  const modalRef = useRef(null);
  const titleId = useRef(AriaUtils.generateId('modal-title'));
  const descriptionId = useRef(AriaUtils.generateId('modal-description'));

  useEffect(() => {
    if (isOpen) {
      // Save current focus
      focusManager.saveFocus();
      
      // Trap focus in modal
      if (modalRef.current) {
        focusManager.trapFocus(modalRef.current);
        
        // Focus initial element or first focusable element
        if (initialFocus) {
          const element = modalRef.current.querySelector(initialFocus);
          if (element) {
            element.focus();
          } else {
            focusManager.focusFirst(modalRef.current);
          }
        } else {
          focusManager.focusFirst(modalRef.current);
        }
      }

      // Prevent body scroll
      document.body.style.overflow = 'hidden';
      
      // Announce modal opening
      AriaUtils.announce(`${title} dialog opened`, 'assertive');
    } else {
      // Restore focus
      focusManager.restoreFocus();
      
      // Remove focus trap
      focusManager.removeFocusTrap();
      
      // Restore body scroll
      document.body.style.overflow = '';
    }

    return () => {
      if (isOpen) {
        focusManager.removeFocusTrap();
        document.body.style.overflow = '';
      }
    };
  }, [isOpen, title, initialFocus]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && closeOnEscape && isOpen) {
        e.preventDefault();
        onClose();
      }
    };

    const handleEscapePressed = () => {
      if (closeOnEscape && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.addEventListener('escape-pressed', handleEscapePressed);
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('escape-pressed', handleEscapePressed);
    };
  }, [isOpen, closeOnEscape, onClose]);

  const handleOverlayClick = (e) => {
    if (e.target === e.currentTarget && closeOnOverlayClick) {
      onClose();
    }
  };

  const getSizeClasses = () => {
    switch (size) {
      case 'small':
        return 'max-w-md';
      case 'large':
        return 'max-w-4xl';
      case 'extra-large':
        return 'max-w-6xl';
      case 'full':
        return 'max-w-full mx-4';
      default:
        return 'max-w-2xl';
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
      onClick={handleOverlayClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId.current}
      aria-describedby={ariaDescribedBy || descriptionId.current}
    >
      <div
        ref={modalRef}
        className={`bg-slate-800 rounded-lg shadow-xl w-full ${getSizeClasses()} max-h-[90vh] overflow-hidden ${className}`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <h2 
            id={titleId.current}
            className="text-xl font-semibold text-white"
          >
            {title}
          </h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors duration-200 p-1 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-800"
            aria-label={`Close ${title} dialog`}
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div 
          id={descriptionId.current}
          className="overflow-y-auto max-h-[calc(90vh-120px)]"
        >
          {children}
        </div>
      </div>
    </div>
  );
};

// Accessible form field component
export const AccessibleFormField = ({ 
  label, 
  id, 
  error, 
  required = false, 
  description = null,
  children,
  className = ''
}) => {
  const errorId = useRef(AriaUtils.generateId('error'));
  const descriptionId = useRef(AriaUtils.generateId('description'));

  return (
    <div className={`space-y-2 ${className}`}>
      <label 
        htmlFor={id}
        className="block text-sm font-medium text-slate-300"
      >
        {label}
        {required && (
          <span className="text-red-400 ml-1" aria-label="required">*</span>
        )}
      </label>
      
      {description && (
        <p 
          id={descriptionId.current}
          className="text-sm text-slate-400"
        >
          {description}
        </p>
      )}
      
      <div className="relative">
        {React.cloneElement(children, {
          id,
          'aria-required': required,
          'aria-invalid': !!error,
          'aria-describedby': [
            error ? errorId.current : null,
            description ? descriptionId.current : null
          ].filter(Boolean).join(' ') || undefined,
          className: `${children.props.className || ''} ${error ? 'border-red-500 focus:border-red-500 focus:ring-red-500' : ''}`
        })}
      </div>
      
      {error && (
        <p 
          id={errorId.current}
          className="text-sm text-red-400 flex items-center space-x-1"
          role="alert"
          aria-live="polite"
        >
          <span>⚠</span>
          <span>{error}</span>
        </p>
      )}
    </div>
  );
};

// Accessible button component
export const AccessibleButton = ({ 
  children, 
  variant = 'primary', 
  size = 'medium',
  disabled = false,
  loading = false,
  ariaLabel = null,
  ariaDescribedBy = null,
  onClick,
  className = '',
  ...props
}) => {
  const getVariantClasses = () => {
    switch (variant) {
      case 'secondary':
        return 'bg-slate-700 text-white border border-slate-600 hover:bg-slate-600 focus:ring-slate-500';
      case 'danger':
        return 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500';
      case 'success':
        return 'bg-green-600 text-white hover:bg-green-700 focus:ring-green-500';
      case 'ghost':
        return 'text-slate-300 hover:text-white hover:bg-slate-700 focus:ring-slate-500';
      default:
        return 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500';
    }
  };

  const getSizeClasses = () => {
    switch (size) {
      case 'small':
        return 'px-3 py-1.5 text-sm';
      case 'large':
        return 'px-6 py-3 text-lg';
      default:
        return 'px-4 py-2';
    }
  };

  return (
    <button
      onClick={onClick}
      disabled={disabled || loading}
      aria-label={ariaLabel}
      aria-describedby={ariaDescribedBy}
      className={`
        ${getVariantClasses()}
        ${getSizeClasses()}
        rounded-lg font-medium transition-all duration-200
        focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-slate-800
        disabled:opacity-50 disabled:cursor-not-allowed
        flex items-center justify-center space-x-2
        ${className}
      `}
      {...props}
    >
      {loading && (
        <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
      )}
      <span>{children}</span>
    </button>
  );
};

// Accessible toggle switch
export const AccessibleToggle = ({ 
  checked, 
  onChange, 
  label, 
  description = null,
  disabled = false,
  className = ''
}) => {
  const id = useRef(AriaUtils.generateId('toggle'));
  const descriptionId = useRef(AriaUtils.generateId('toggle-description'));

  return (
    <div className={`flex items-center justify-between ${className}`}>
      <div className="flex-1">
        <label 
          htmlFor={id.current}
          className="text-sm font-medium text-white cursor-pointer"
        >
          {label}
        </label>
        {description && (
          <p 
            id={descriptionId.current}
            className="text-sm text-slate-400 mt-1"
          >
            {description}
          </p>
        )}
      </div>
      
      <button
        id={id.current}
        type="button"
        role="switch"
        aria-checked={checked}
        aria-describedby={description ? descriptionId.current : undefined}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`
          relative inline-flex h-6 w-11 items-center rounded-full transition-colors
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-800
          disabled:opacity-50 disabled:cursor-not-allowed
          ${checked ? 'bg-blue-600' : 'bg-slate-600'}
        `}
      >
        <span
          className={`
            inline-block h-4 w-4 transform rounded-full bg-white transition-transform
            ${checked ? 'translate-x-6' : 'translate-x-1'}
          `}
        />
        <span className="sr-only">
          {checked ? 'Disable' : 'Enable'} {label}
        </span>
      </button>
    </div>
  );
};

export default AccessibleModal;
