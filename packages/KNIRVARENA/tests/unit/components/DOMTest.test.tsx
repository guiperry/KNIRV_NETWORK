/**
 * Simple DOM Environment Test
 * Test to verify that the DOM environment is properly set up
 */

import * as React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// Simple test component
const TestComponent: React.FC = () => {
  return <div data-testid="test-component">Hello World</div>;
};

describe('DOM Environment Test', () => {
  it('should have a working DOM environment', () => {
    // Test basic DOM functionality
    expect(document).toBeDefined();
    expect(document.body).toBeDefined();
    expect(document.createElement).toBeDefined();
    
    // Test creating elements
    const div = document.createElement('div');
    expect(div).toBeDefined();
    expect(div.tagName).toBe('DIV');
  });

  it('should be able to render a simple React component', () => {
    // Try to render without custom container first
    try {
      render(<TestComponent />);
      expect(screen.getByTestId('test-component')).toBeInTheDocument();
      expect(screen.getByText('Hello World')).toBeInTheDocument();
    } catch (error) {
      console.error('Basic render failed:', error);
      throw error;
    }
  });

  it('should be able to render with explicit container', () => {
    // Create explicit container
    const container = document.createElement('div');
    document.body.appendChild(container);
    
    try {
      render(<TestComponent />, { container });
      expect(screen.getByTestId('test-component')).toBeInTheDocument();
    } catch (error) {
      console.error('Container render failed:', error);
      throw error;
    } finally {
      // Clean up
      if (container.parentNode) {
        container.parentNode.removeChild(container);
      }
    }
  });
});
