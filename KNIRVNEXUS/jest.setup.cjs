require('@testing-library/jest-dom')

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter() {
    return {
      push: jest.fn(),
      replace: jest.fn(),
      prefetch: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
    }
  },
  useSearchParams() {
    return new URLSearchParams()
  },
  usePathname() {
    return '/'
  },
}))

// Mock Next.js image
jest.mock('next/image', () => ({
  __esModule: true,
  default: (props) => {
    const React = require('react');
    // eslint-disable-next-line @next/next/no-img-element
    return React.createElement('img', props);
  },
}))

// Mock Lucide React icons
jest.mock('lucide-react', () => {
  const React = require('react');
  return {
    __esModule: true,
    ...Object.fromEntries(
      [
        'Activity', 'AlertTriangle', 'ArrowUpRight', 'BarChart3', 'Bell', 'Bot', 'Calendar',
        'CheckCircle', 'ChevronDown', 'ChevronLeft', 'ChevronRight', 'Clock', 'Cpu', 'CreditCard',
        'Database', 'Download', 'Edit', 'Eye', 'FileText', 'Globe', 'HardDrive', 'Lock', 'Network',
        'Play', 'Plus', 'QrCode', 'RefreshCw', 'Server', 'Settings', 'Shield', 'Square', 'Timer',
        'Trash2', 'TrendingUp', 'Unlock', 'Upload', 'User', 'Wifi', 'WifiOff', 'Zap'
      ].map(iconName => [
        iconName,
        (props) => React.createElement('div', { 'data-testid': `${iconName}-icon`, ...props })
      ])
    ),
  };
})

// Mock QR Code component
jest.mock('qrcode.react', () => {
  const React = require('react');
  return {
    QRCodeSVG: ({ value, ...props }) =>
      React.createElement('div', { 'data-testid': 'qr-code', 'data-value': value, ...props }, `QR Code: ${value}`)
  };
})

// Mock socket.io-client
jest.mock('socket.io-client', () => ({
  io: jest.fn(() => ({
    on: jest.fn(),
    off: jest.fn(),
    emit: jest.fn(),
    connect: jest.fn(),
    disconnect: jest.fn(),
    connected: false,
  })),
}))

// Mock fetch globally
global.fetch = jest.fn()

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // deprecated
    removeListener: jest.fn(), // deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
})

// Mock ResizeObserver
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}))

// Mock IntersectionObserver
global.IntersectionObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}))

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
}
global.localStorage = localStorageMock

// Mock console methods to reduce noise in tests
global.console = {
  ...console,
  log: jest.fn(),
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
}
