/**
 * @jest-environment jsdom
 */


import { renderHook, act } from '@testing-library/react';
import { useRouter } from 'next/router';
import { useNavigation } from './useNavigation';

// Mock the useRouter hook from next/router
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));

describe('useNavigation', () => {
  beforeEach(() => {
    // Reset the mock before each test
    useRouter.mockReset();
  });

  it('should initialize with the correct initial page', () => {
    // Arrange
    const mockRouter = {
      pathname: '/',
      isReady: true,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';

    // Act
    const { result } = renderHook(() => useNavigation(initialPage));

    // Assert
    expect(result.current.activePage).toBe(initialPage);
  });

  it('should update activePage and push to the correct route when handleNavigation is called', () => {
    // Arrange
    const mockRouter = {
      pathname: '/',
      isReady: true,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';
    const { result } = renderHook(() => useNavigation(initialPage));
    const newPage = 'inventory';

    // Act
    act(() => {
      result.current.handleNavigation(newPage);
    });

    // Assert
    expect(result.current.activePage).toBe(newPage);
    expect(mockRouter.push).toHaveBeenCalledWith(`/${newPage}`);
  });

  it('should update activePage when the route changes', () => {
    // Arrange
    let mockRouter = {
      pathname: '/',
      isReady: true,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';
    const { result, rerender } = renderHook(() => useNavigation(initialPage));

    // Act
    mockRouter = {
      ...mockRouter,
      pathname: '/inventory',
    };
    useRouter.mockReturnValue(mockRouter);
    rerender();

    // Assert
    expect(result.current.activePage).toBe('inventory');
  });

  it('should return the initial page when the path is root', () => {
    // Arrange
    const mockRouter = {
      pathname: '/',
      isReady: true,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';

    // Act
    const { result } = renderHook(() => useNavigation(initialPage));

    // Assert
    expect(result.current.activePage).toBe(initialPage);
  });

  it('should return the correct page from the URL', () => {
    // Arrange
    const mockRouter = {
      pathname: '/vault',
      isReady: true,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';

    // Act
    const { result } = renderHook(() => useNavigation(initialPage));

    // Assert
    expect(result.current.activePage).toBe('vault');
  });

  it('should return the initial page when router is not ready', () => {
    // Arrange
    const mockRouter = {
      pathname: '/vault',
      isReady: false,
      push: jest.fn(),
    };
    useRouter.mockReturnValue(mockRouter);
    const initialPage = 'home';

    // Act
    const { result } = renderHook(() => useNavigation(initialPage));

    // Assert
    expect(result.current.activePage).toBe(initialPage);
  });
});
