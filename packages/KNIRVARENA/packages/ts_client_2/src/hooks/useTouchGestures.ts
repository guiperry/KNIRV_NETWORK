/**
 * Touch Gestures Hook
 * Provides comprehensive touch gesture support for mobile interactions
 */

import { useEffect, useCallback, useRef, useState } from 'react';

interface TouchPoint {
  id: number;
  x: number;
  y: number;
  timestamp: number;
}

interface GestureState {
  isActive: boolean;
  startTime: number;
  startPoints: TouchPoint[];
  currentPoints: TouchPoint[];
  velocity: { x: number; y: number };
  distance: number;
  angle: number;
  scale: number;
  rotation: number;
}

interface SwipeGesture {
  direction: 'up' | 'down' | 'left' | 'right';
  distance: number;
  velocity: number;
  duration: number;
}

interface PinchGesture {
  scale: number;
  center: { x: number; y: number };
  velocity: number;
}

interface RotationGesture {
  angle: number;
  center: { x: number; y: number };
  velocity: number;
}

interface TapGesture {
  x: number;
  y: number;
  tapCount: number;
  timestamp: number;
}

interface TouchGestureOptions {
  enabled?: boolean;
  swipeThreshold?: number;
  pinchThreshold?: number;
  rotationThreshold?: number;
  tapTimeout?: number;
  doubleTapTimeout?: number;
  longPressTimeout?: number;
  preventDefault?: boolean;
  onSwipe?: (gesture: SwipeGesture) => void;
  onPinch?: (gesture: PinchGesture) => void;
  onRotation?: (gesture: RotationGesture) => void;
  onTap?: (gesture: TapGesture) => void;
  onDoubleTap?: (gesture: TapGesture) => void;
  onLongPress?: (gesture: TapGesture) => void;
  onPan?: (delta: { x: number; y: number }, velocity: { x: number; y: number }) => void;
}

export const useTouchGestures = (
  elementRef: React.RefObject<HTMLElement>,
  options: TouchGestureOptions = {}
) => {
  const {
    enabled = true,
    swipeThreshold = 50,
    pinchThreshold = 0.1,
    rotationThreshold = 10,
    tapTimeout = 300,
    doubleTapTimeout = 300,
    longPressTimeout = 500,
    preventDefault = true,
    onSwipe,
    onPinch,
    onRotation,
    onTap,
    onDoubleTap,
    onLongPress,
    onPan
  } = options;

  const gestureState = useRef<GestureState>({
    isActive: false,
    startTime: 0,
    startPoints: [],
    currentPoints: [],
    velocity: { x: 0, y: 0 },
    distance: 0,
    angle: 0,
    scale: 1,
    rotation: 0
  });

  const lastTap = useRef<{ x: number; y: number; timestamp: number } | null>(null);
  const longPressTimer = useRef<number | null>(null);
  const [isLongPressing, setIsLongPressing] = useState(false);

  // Helper functions
  const getTouchPoint = (touch: Touch): TouchPoint => ({
    id: touch.identifier,
    x: touch.clientX,
    y: touch.clientY,
    timestamp: Date.now()
  });

  const getDistance = (p1: TouchPoint, p2: TouchPoint): number => {
    const dx = p2.x - p1.x;
    const dy = p2.y - p1.y;
    return Math.sqrt(dx * dx + dy * dy);
  };

  const getAngle = (p1: TouchPoint, p2: TouchPoint): number => {
    return Math.atan2(p2.y - p1.y, p2.x - p1.x) * 180 / Math.PI;
  };

  const getCenter = useCallback((points: TouchPoint[]): { x: number; y: number } => {
    const x = points.reduce((sum, p) => sum + p.x, 0) / points.length;
    const y = points.reduce((sum, p) => sum + p.y, 0) / points.length;
    return { x, y };
  }, []);

  const getVelocity = useCallback((start: TouchPoint[], current: TouchPoint[]): { x: number; y: number } => {
    if (start.length === 0 || current.length === 0) return { x: 0, y: 0 };

    const startCenter = getCenter(start);
    const currentCenter = getCenter(current);
    const timeDiff = (current[0].timestamp - start[0].timestamp) / 1000; // Convert to seconds

    if (timeDiff === 0) return { x: 0, y: 0 };

    return {
      x: (currentCenter.x - startCenter.x) / timeDiff,
      y: (currentCenter.y - startCenter.y) / timeDiff
    };
  }, [getCenter]);

  // Clear long press timer
  const clearLongPressTimer = useCallback(() => {
    if (longPressTimer.current) {
      window.clearTimeout(longPressTimer.current);
      longPressTimer.current = null;
    }
    setIsLongPressing(false);
  }, []);

  // Handle touch start
  const handleTouchStart = useCallback((event: TouchEvent) => {
    if (!enabled) return;
    
    if (preventDefault) {
      event.preventDefault();
    }

    const touches = Array.from(event.touches).map(getTouchPoint);
    
    gestureState.current = {
      isActive: true,
      startTime: Date.now(),
      startPoints: touches,
      currentPoints: touches,
      velocity: { x: 0, y: 0 },
      distance: touches.length > 1 ? getDistance(touches[0], touches[1]) : 0,
      angle: touches.length > 1 ? getAngle(touches[0], touches[1]) : 0,
      scale: 1,
      rotation: 0
    };

    // Start long press timer for single touch
    if (touches.length === 1 && onLongPress) {
      longPressTimer.current = window.setTimeout(() => {
        setIsLongPressing(true);
        onLongPress({
          x: touches[0].x,
          y: touches[0].y,
          tapCount: 1,
          timestamp: Date.now()
        });
      }, longPressTimeout);
    }
  }, [enabled, preventDefault, onLongPress, longPressTimeout]);

  // Handle touch move
  const handleTouchMove = useCallback((event: TouchEvent) => {
    if (!enabled || !gestureState.current.isActive) return;
    
    if (preventDefault) {
      event.preventDefault();
    }

    clearLongPressTimer();

    const touches = Array.from(event.touches).map(getTouchPoint);
    const state = gestureState.current;
    
    state.currentPoints = touches;
    state.velocity = getVelocity(state.startPoints, touches);

    // Handle pan gesture
    if (touches.length === 1 && onPan) {
      const startPoint = state.startPoints[0];
      const currentPoint = touches[0];
      const delta = {
        x: currentPoint.x - startPoint.x,
        y: currentPoint.y - startPoint.y
      };
      onPan(delta, state.velocity);
    }

    // Handle pinch gesture
    if (touches.length === 2 && state.startPoints.length === 2) {
      const currentDistance = getDistance(touches[0], touches[1]);
      const startDistance = state.distance;
      
      if (startDistance > 0) {
        const scale = currentDistance / startDistance;
        state.scale = scale;
        
        if (onPinch && Math.abs(scale - 1) > pinchThreshold) {
          const center = getCenter(touches);
          onPinch({
            scale,
            center,
            velocity: Math.abs(state.velocity.x) + Math.abs(state.velocity.y)
          });
        }
      }

      // Handle rotation gesture
      if (onRotation) {
        const currentAngle = getAngle(touches[0], touches[1]);
        const startAngle = state.angle;
        let rotation = currentAngle - startAngle;
        
        // Normalize rotation to -180 to 180
        while (rotation > 180) rotation -= 360;
        while (rotation < -180) rotation += 360;
        
        state.rotation = rotation;
        
        if (Math.abs(rotation) > rotationThreshold) {
          const center = getCenter(touches);
          onRotation({
            angle: rotation,
            center,
            velocity: Math.abs(state.velocity.x) + Math.abs(state.velocity.y)
          });
        }
      }
    }
  }, [enabled, preventDefault, onPan, onPinch, onRotation, pinchThreshold, rotationThreshold, clearLongPressTimer, getVelocity, getCenter]);

  // Handle touch end
  const handleTouchEnd = useCallback((event: TouchEvent) => {
    if (!enabled || !gestureState.current.isActive) return;
    
    if (preventDefault) {
      event.preventDefault();
    }

    clearLongPressTimer();

    const state = gestureState.current;
    const duration = Date.now() - state.startTime;
    const remainingTouches = Array.from(event.touches).map(getTouchPoint);

    // Handle swipe gesture
    if (remainingTouches.length === 0 && state.startPoints.length === 1 && onSwipe) {
      const startPoint = state.startPoints[0];
      const endPoint = state.currentPoints[0];
      
      if (endPoint) {
        const deltaX = endPoint.x - startPoint.x;
        const deltaY = endPoint.y - startPoint.y;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        if (distance > swipeThreshold) {
          let direction: 'up' | 'down' | 'left' | 'right';
          
          if (Math.abs(deltaX) > Math.abs(deltaY)) {
            direction = deltaX > 0 ? 'right' : 'left';
          } else {
            direction = deltaY > 0 ? 'down' : 'up';
          }
          
          onSwipe({
            direction,
            distance,
            velocity: Math.abs(state.velocity.x) + Math.abs(state.velocity.y),
            duration
          });
        }
      }
    }

    // Handle tap gesture
    if (remainingTouches.length === 0 && state.startPoints.length === 1 && duration < tapTimeout && !isLongPressing) {
      const tapPoint = state.startPoints[0];
      const currentTime = Date.now();
      
      // Check for double tap
      if (lastTap.current && 
          currentTime - lastTap.current.timestamp < doubleTapTimeout &&
          Math.abs(tapPoint.x - lastTap.current.x) < 50 &&
          Math.abs(tapPoint.y - lastTap.current.y) < 50) {
        
        if (onDoubleTap) {
          onDoubleTap({
            x: tapPoint.x,
            y: tapPoint.y,
            tapCount: 2,
            timestamp: currentTime
          });
        }
        lastTap.current = null;
      } else {
        if (onTap) {
          onTap({
            x: tapPoint.x,
            y: tapPoint.y,
            tapCount: 1,
            timestamp: currentTime
          });
        }
        lastTap.current = {
          x: tapPoint.x,
          y: tapPoint.y,
          timestamp: currentTime
        };
      }
    }

    // Reset gesture state if no touches remain
    if (remainingTouches.length === 0) {
      gestureState.current.isActive = false;
    }
  }, [enabled, preventDefault, onSwipe, onTap, onDoubleTap, swipeThreshold, tapTimeout, doubleTapTimeout, isLongPressing, clearLongPressTimer]);

  // Set up event listeners
  useEffect(() => {
    if (!enabled || !elementRef.current) return;

    const element = elementRef.current;
    
    element.addEventListener('touchstart', handleTouchStart, { passive: !preventDefault });
    element.addEventListener('touchmove', handleTouchMove, { passive: !preventDefault });
    element.addEventListener('touchend', handleTouchEnd, { passive: !preventDefault });
    element.addEventListener('touchcancel', handleTouchEnd, { passive: !preventDefault });

    return () => {
      element.removeEventListener('touchstart', handleTouchStart);
      element.removeEventListener('touchmove', handleTouchMove);
      element.removeEventListener('touchend', handleTouchEnd);
      element.removeEventListener('touchcancel', handleTouchEnd);
      clearLongPressTimer();
    };
  }, [enabled, elementRef, handleTouchStart, handleTouchMove, handleTouchEnd, preventDefault, clearLongPressTimer]);

  return {
    isActive: gestureState.current.isActive,
    isLongPressing,
    gestureState: gestureState.current
  };
};
