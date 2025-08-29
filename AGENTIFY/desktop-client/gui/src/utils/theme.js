// Theme consistency utilities for the KNIRVENGINE

/**
 * Color palette with WCAG AA compliant contrast ratios
 */
export const colors = {
  // Primary colors
  primary: {
    50: '#eff6ff',
    100: '#dbeafe',
    200: '#bfdbfe',
    300: '#93c5fd',
    400: '#60a5fa',
    500: '#3b82f6',
    600: '#2563eb',
    700: '#1d4ed8',
    800: '#1e40af',
    900: '#1e3a8a',
  },
  
  // Secondary colors
  secondary: {
    50: '#f8fafc',
    100: '#f1f5f9',
    200: '#e2e8f0',
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748b',
    600: '#475569',
    700: '#334155',
    800: '#1e293b',
    900: '#0f172a',
  },
  
  // Success colors
  success: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#22c55e',
    600: '#16a34a',
    700: '#15803d',
    800: '#166534',
    900: '#14532d',
  },
  
  // Warning colors
  warning: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
  },
  
  // Error colors
  error: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444',
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
  },
  
  // Neutral colors (slate)
  neutral: {
    50: '#f8fafc',
    100: '#f1f5f9',
    200: '#e2e8f0',
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748b',
    600: '#475569',
    700: '#334155',
    800: '#1e293b',
    900: '#0f172a',
  }
};

/**
 * Semantic color mappings
 */
export const semanticColors = {
  background: {
    primary: colors.neutral[900],
    secondary: colors.neutral[800],
    tertiary: colors.neutral[700],
    overlay: 'rgba(0, 0, 0, 0.5)',
  },
  
  text: {
    primary: '#ffffff',
    secondary: colors.neutral[300],
    tertiary: colors.neutral[400],
    muted: colors.neutral[500],
    inverse: colors.neutral[900],
  },
  
  border: {
    primary: colors.neutral[700],
    secondary: colors.neutral[600],
    focus: colors.primary[500],
    error: colors.error[500],
    success: colors.success[500],
  },
  
  interactive: {
    primary: colors.primary[600],
    primaryHover: colors.primary[700],
    secondary: colors.neutral[700],
    secondaryHover: colors.neutral[600],
    danger: colors.error[600],
    dangerHover: colors.error[700],
    success: colors.success[600],
    successHover: colors.success[700],
  },
  
  status: {
    active: colors.success[500],
    inactive: colors.neutral[500],
    warning: colors.warning[500],
    error: colors.error[500],
    info: colors.primary[500],
  }
};

/**
 * Typography scale
 */
export const typography = {
  fontFamily: {
    sans: ['Inter', 'system-ui', 'sans-serif'],
    mono: ['JetBrains Mono', 'Consolas', 'monospace'],
  },
  
  fontSize: {
    xs: '0.75rem',
    sm: '0.875rem',
    base: '1rem',
    lg: '1.125rem',
    xl: '1.25rem',
    '2xl': '1.5rem',
    '3xl': '1.875rem',
    '4xl': '2.25rem',
  },
  
  fontWeight: {
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
  },
  
  lineHeight: {
    tight: '1.25',
    normal: '1.5',
    relaxed: '1.75',
  }
};

/**
 * Spacing scale
 */
export const spacing = {
  0: '0',
  1: '0.25rem',
  2: '0.5rem',
  3: '0.75rem',
  4: '1rem',
  5: '1.25rem',
  6: '1.5rem',
  8: '2rem',
  10: '2.5rem',
  12: '3rem',
  16: '4rem',
  20: '5rem',
  24: '6rem',
};

/**
 * Border radius scale
 */
export const borderRadius = {
  none: '0',
  sm: '0.125rem',
  base: '0.25rem',
  md: '0.375rem',
  lg: '0.5rem',
  xl: '0.75rem',
  '2xl': '1rem',
  full: '9999px',
};

/**
 * Shadow scale
 */
export const shadows = {
  sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
  base: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06)',
  md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
  lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
  xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
  '2xl': '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
  inner: 'inset 0 2px 4px 0 rgba(0, 0, 0, 0.06)',
};

/**
 * Animation durations
 */
export const animation = {
  duration: {
    fast: '150ms',
    normal: '200ms',
    slow: '300ms',
  },
  
  easing: {
    ease: 'ease',
    easeIn: 'ease-in',
    easeOut: 'ease-out',
    easeInOut: 'ease-in-out',
  }
};

/**
 * Component-specific theme utilities
 */
export const components = {
  button: {
    primary: {
      background: semanticColors.interactive.primary,
      backgroundHover: semanticColors.interactive.primaryHover,
      text: semanticColors.text.primary,
      border: 'transparent',
    },
    secondary: {
      background: semanticColors.interactive.secondary,
      backgroundHover: semanticColors.interactive.secondaryHover,
      text: semanticColors.text.primary,
      border: semanticColors.border.primary,
    },
    danger: {
      background: semanticColors.interactive.danger,
      backgroundHover: semanticColors.interactive.dangerHover,
      text: semanticColors.text.primary,
      border: 'transparent',
    },
  },
  
  input: {
    background: semanticColors.background.secondary,
    backgroundFocus: semanticColors.background.tertiary,
    text: semanticColors.text.primary,
    placeholder: semanticColors.text.muted,
    border: semanticColors.border.primary,
    borderFocus: semanticColors.border.focus,
    borderError: semanticColors.border.error,
  },
  
  card: {
    background: `${semanticColors.background.secondary}80`, // 50% opacity
    backgroundHover: `${semanticColors.background.secondary}CC`, // 80% opacity
    border: `${semanticColors.border.primary}80`, // 50% opacity
    borderHover: `${semanticColors.border.secondary}80`, // 50% opacity
  },
  
  modal: {
    overlay: semanticColors.background.overlay,
    background: semanticColors.background.secondary,
    border: semanticColors.border.primary,
  }
};

/**
 * Utility functions
 */
export const themeUtils = {
  // Get color with opacity
  withOpacity: (color, opacity) => {
    if (color.startsWith('#')) {
      const hex = color.slice(1);
      const r = parseInt(hex.slice(0, 2), 16);
      const g = parseInt(hex.slice(2, 4), 16);
      const b = parseInt(hex.slice(4, 6), 16);
      return `rgba(${r}, ${g}, ${b}, ${opacity})`;
    }
    return color;
  },
  
  // Get focus ring classes
  focusRing: (color = 'primary') => {
    const ringColor = colors[color]?.[500] || colors.primary[500];
    return `focus:outline-none focus:ring-2 focus:ring-${color}-500 focus:ring-offset-2 focus:ring-offset-slate-900`;
  },
  
  // Get status color
  getStatusColor: (status) => {
    switch (status) {
      case 'active':
      case 'success':
      case 'completed':
        return semanticColors.status.active;
      case 'warning':
      case 'pending':
        return semanticColors.status.warning;
      case 'error':
      case 'failed':
        return semanticColors.status.error;
      case 'info':
      case 'building':
        return semanticColors.status.info;
      default:
        return semanticColors.status.inactive;
    }
  },
  
  // Get gradient classes
  gradient: (from, to) => {
    return `bg-gradient-to-r from-${from} to-${to}`;
  }
};

/**
 * Dark mode theme (default)
 */
export const darkTheme = {
  colors,
  semanticColors,
  typography,
  spacing,
  borderRadius,
  shadows,
  animation,
  components,
  utils: themeUtils,
};

export default darkTheme;
