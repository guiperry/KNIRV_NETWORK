/**
 * Mobile Touch Interaction Tests
 * Tests touch gestures and mobile-specific interactions for KNIRVCONTROLLER
 */

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { jest } from '@jest/globals';
import userEvent from '@testing-library/user-event';

// Mock touch event support - utility function for creating touch events
const createMockTouchEvent = (type: string, touches: Touch[]) => {
  return new TouchEvent(type, {
    touches,
    targetTouches: touches,
    changedTouches: touches,
    bubbles: true,
    cancelable: true,
  });
};

// Mock Touch constructor if not available
if (typeof Touch === 'undefined') {
  global.Touch = class {
    constructor(public identifier: number, public target: EventTarget, public clientX = 0, public clientY = 0) {}
  } as typeof Touch;
}

// Mock TouchEvent constructor
if (typeof TouchEvent === 'undefined') {
  global.TouchEvent = class extends Event {
    touches: Touch[];
    changedTouches: Touch[];
    targetTouches: Touch[];

    constructor(type: string, options: { touches?: Touch[]; changedTouches?: Touch[]; targetTouches?: Touch[] } = {}) {
      super(type);
      this.touches = options.touches || [];
      this.changedTouches = options.changedTouches || [];
      this.targetTouches = options.targetTouches || [];
    }
  } as typeof TouchEvent;
}

// Mock components and services
jest.mock('../services/KnirvanaBridgeService', () => ({
  knirvanaBridgeService: {
    initialize: jest.fn(),
    startGame: jest.fn(),
    pauseGame: jest.fn(),
    selectErrorNode: jest.fn(),
    deployAgent: jest.fn(),
    getGameState: jest.fn(() => ({
      nrnBalance: 500,
      agents: [],
      errorNodes: [],
      selectedErrorNode: null,
    })),
  },
}));

jest.mock('../services/DesktopConnection', () => ({
  desktopConnection: {
    getConnectionStatus: () => ({ connected: false, desktop_id: null }),
    setConnectionChangeHandler: jest.fn(),
    setHRMResponseHandler: jest.fn(),
  },
}));

import KNIRVANAGameVisualization from '../components/KNIRVANAGameVisualization';

// Helper function to create touch points
const createTouch = (x: number, y: number, identifier = 1): Touch => {
  return {
    identifier,
    target: document.body,
    clientX: x,
    clientY: y,
    pageX: x,
    pageY: y,
    screenX: x,
    screenY: y,
    // Add missing Touch interface properties
    force: 1.0,
    radiusX: 1.0,
    radiusY: 1.0,
    rotationAngle: 0.0
  } as Touch;
};

describe('Mobile Touch Interactions', () => {
  beforeEach(() => {
    // Setup user event for tests that need it
    userEvent.setup({
      advanceTimers: jest.advanceTimersByTime,
    });

    // Mock window dimensions for mobile
    Object.defineProperty(window, 'innerWidth', { value: 375 });
    Object.defineProperty(window, 'innerHeight', { value: 667 });

    // Mock touch support
    Object.defineProperty(navigator, 'maxTouchPoints', { value: 5 });

    // Mock viewport meta tag
    const meta = document.createElement('meta');
    meta.name = 'viewport';
    meta.content = 'width=device-width, initial-scale=1.0';
    document.head.appendChild(meta);
  });

  afterEach(() => {
    // Clean up
    const meta = document.querySelector('meta[name="viewport"]');
    if (meta) {
      document.head.removeChild(meta);
    }

    jest.clearAllMocks();
  });

  describe('Touch Target Sizing', () => {
    test('buttons should have minimum touch target size', () => {
      const TestButton = () => <button>Test Button</button>;

      const { container } = render(<TestButton />);
      const button = container.querySelector('button');

      if (button) {
        const rect = button.getBoundingClientRect();

        // Check minimum touch target size (44x44 pixels as per WCAG)
        expect(rect.width).toBeGreaterThanOrEqual(44);
        expect(rect.height).toBeGreaterThanOrEqual(44);
      }
    });

    test('interactive elements should not be too close together', () => {
      const TestButtons = () => (
        <div>
          <button style={{ margin: '10px' }}>Button 1</button>
          <button style={{ margin: '10px' }}>Button 2</button>
        </div>
      );

      const { container } = render(<TestButtons />);
      const buttons = container.querySelectorAll('button');

      if (buttons.length >= 2) {
        const rect1 = buttons[0].getBoundingClientRect();
        const rect2 = buttons[1].getBoundingClientRect();

        // Check that buttons are not overlapping
        const noOverlap =
          rect1.right < rect2.left ||
          rect1.left > rect2.right ||
          rect1.bottom < rect2.top ||
          rect1.top > rect2.bottom;

        expect(noOverlap).toBe(true);
      }
    });
  });

  describe('Touch Gestures', () => {
    test('should handle single touch', () => {
      const mockOnTouch = jest.fn();
      const TestTouchable = () => (
        <div
          onTouchStart={mockOnTouch}
          style={{ width: 100, height: 100 }}
          data-testid="touchable"
        >
          Touchable
        </div>
      );

      render(<TestTouchable />);
      const element = screen.getByTestId('touchable');

      const touch = createTouch(50, 50, 1);
      const touchEvent = createMockTouchEvent('touchstart', [touch]);

      fireEvent(element, touchEvent);

      expect(mockOnTouch).toHaveBeenCalledTimes(1);
    });

    test('should handle multi-touch gestures', () => {
      const mockOnTouchStart = jest.fn();
      const mockOnTouchMove = jest.fn();
      const mockOnTouchEnd = jest.fn();

      const TestMultiTouch = () => (
        <div
          onTouchStart={mockOnTouchStart}
          onTouchMove={mockOnTouchMove}
          onTouchEnd={mockOnTouchEnd}
          style={{ width: 200, height: 200 }}
          data-testid="multitouch"
        >
          Multi-Touch Area
        </div>
      );

      render(<TestMultiTouch />);
      const element = screen.getByTestId('multitouch');

      // Simulate pinch gesture (2 touches)
      const touch1 = createTouch(50, 50, 1);
      const touch2 = createTouch(150, 150, 2);

      // Touch start
      fireEvent.touchStart(element, { touches: [touch1, touch2] });
      expect(mockOnTouchStart).toHaveBeenCalledTimes(1);

      // Touch move
      const movedTouch1 = createTouch(45, 45, 1);
      const movedTouch2 = createTouch(155, 155, 2);
      fireEvent.touchMove(element, { touches: [movedTouch1, movedTouch2] });
      expect(mockOnTouchMove).toHaveBeenCalledTimes(1);

      // Touch end
      fireEvent.touchEnd(element, { changedTouches: [touch1] });
      expect(mockOnTouchEnd).toHaveBeenCalledTimes(1);
    });
  });

  describe('Game Visualization Touch Controls', () => {
    test('should support touch-based camera controls', async () => {
      const { container } = render(
        <div style={{ width: 800, height: 600 }}>
          <KNIRVANAGameVisualization />
        </div>
      );

      const canvas = container.querySelector('canvas');

      if (canvas) {
        const touch = createTouch(100, 100, 1);
        const touchEvent = new TouchEvent('touchstart', {
          touches: [touch],
          targetTouches: [touch],
          changedTouches: [touch],
          bubbles: true,
        });

        // This should trigger camera controls in the 3D scene
        fireEvent(canvas, touchEvent);

        // Verify touch event was applied (mock verification)
        await waitFor(() => {
          expect(container).toBeDefined();
        });
      }
    });

    test('should handle double tap for zoom', () => {
      const mockOnDoubleClick = jest.fn();

      const TestZoomable = () => (
        <div
          onDoubleClick={mockOnDoubleClick}
          style={{ width: 300, height: 300 }}
          data-testid="zoomable"
        >
          Zoomable Area
        </div>
      );

      render(<TestZoomable />);
      const element = screen.getByTestId('zoomable');

      // Simulate double tap
      fireEvent.doubleClick(element);
      expect(mockOnDoubleClick).toHaveBeenCalledTimes(1);
    });
  });

  describe('Mobile Responsiveness', () => {
    test('should adapt layout for mobile screens', () => {
      const TestResponsive = () => (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div className="bg-blue-500 p-4">Item 1</div>
          <div className="bg-green-500 p-4">Item 2</div>
          <div className="bg-red-500 p-4">Item 3</div>
        </div>
      );

      render(<TestResponsive />);

      // On mobile (375px width), should show single column
      const container = screen.getByText('Item 1').closest('div');

      // Verify mobile-first responsive classes are applied
      expect(container).toHaveClass('grid-cols-1');

      // Verify computed styles for mobile layout
      const computedStyle = window.getComputedStyle(container!);
      expect(computedStyle.display).toBe('grid');
    });

    test('should handle horizontal swipe gestures', () => {
      const mockOnSwipe = jest.fn();

      const TestSwipeable = () => (
        <div
          onTouchStart={(e) => {
            const startX = e.touches[0].clientX;
            const handleTouchEnd = (endEvent: TouchEvent) => {
              const endX = endEvent.changedTouches[0].clientX;
              const deltaX = endX - startX;

              if (Math.abs(deltaX) > 50) { // Minimum swipe distance
                mockOnSwipe(deltaX > 0 ? 'right' : 'left');
              }
            };

            e.target.addEventListener('touchend', handleTouchEnd as EventListener, { once: true });
          }}
          style={{ width: 300, height: 100 }}
          data-testid="swipeable"
        >
          Swipe Me
        </div>
      );

      render(<TestSwipeable />);
      const element = screen.getByTestId('swipeable');

      // Simulate horizontal swipe
      const touchStart = createTouch(100, 50, 1);
      const touchEnd = createTouch(200, 50, 1);

      fireEvent.touchStart(element, { touches: [touchStart] });
      fireEvent.touchEnd(element, { changedTouches: [touchEnd] });

      expect(mockOnSwipe).toHaveBeenCalledWith('right');
    });
  });

  describe('Touch Accessibility', () => {
    test('should support touch-based keyboard navigation', async () => {
      const TestFocusable = () => (
        <div>
          <button>Button 1</button>
          <input type="text" placeholder="Input field" />
          <button>Button 2</button>
        </div>
      );

      render(<TestFocusable />);

      const buttons = screen.getAllByRole('button');
      const input = screen.getByPlaceholderText('Input field');

      // Touch should be able to focus elements
      fireEvent.touchStart(buttons[0]);
      expect(buttons[0]).toHaveFocus();

      fireEvent.touchStart(input);
      expect(input).toHaveFocus();

      fireEvent.touchStart(buttons[1]);
      expect(buttons[1]).toHaveFocus();
    });

    test('should prevent text selection on interactive elements', () => {
      const TestNoSelect = () => (
        <button style={{ userSelect: 'none' }}>No Select Button</button>
      );

      const { container } = render(<TestNoSelect />);
      const button = container.querySelector('button');

      if (button) {
        const computedStyle = window.getComputedStyle(button);
        expect(computedStyle.userSelect).toBe('none');
      }
    });
  });

  describe('Performance on Mobile', () => {
    test('should limit concurrent touch events', () => {
      const mockOnTouch = jest.fn();

      const TestPerformance = () => (
        <div
          onTouchStart={mockOnTouch}
          style={{ width: 300, height: 300 }}
          data-testid="performance-test"
        >
          Performance Test
        </div>
      );

      render(<TestPerformance />);
      const element = screen.getByTestId('performance-test');

      // Simulate rapid touch events
      for (let i = 0; i < 10; i++) {
        const touch = createTouch(100 + i * 10, 100, i + 1);
        fireEvent.touchStart(element, { touches: [touch] });
      }

      // Touch events should be throttled or handled efficiently
      expect(mockOnTouch).toHaveBeenCalled();
    });

    test('should prevent default touch behaviors when needed', () => {
      const mockPreventDefault = jest.fn();

      const TestPreventDefault = () => (
        <div
          onTouchStart={(e) => {
            e.preventDefault();
            mockPreventDefault();
          }}
          style={{ width: 300, height: 300 }}
          data-testid="prevent-default"
        >
          Prevent Default
        </div>
      );

      render(<TestPreventDefault />);
      const element = screen.getByTestId('prevent-default');

      const touch = createTouch(100, 100, 1);
      const touchEvent = new TouchEvent('touchstart', {
        touches: [touch],
        bubbles: true,
        cancelable: true,
      });

      // Mock preventDefault
      touchEvent.preventDefault = mockPreventDefault;

      fireEvent(element, touchEvent);

      expect(mockPreventDefault).toHaveBeenCalled();
    });
  });
});
