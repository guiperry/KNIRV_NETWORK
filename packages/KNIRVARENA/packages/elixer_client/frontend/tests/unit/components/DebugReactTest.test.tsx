/**
 * Debug React Test - Minimal test to debug React rendering issues
 */

import * as React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

describe('Debug React Test', () => {
  it('should render a minimal React component', () => {
    // Simple functional component
    const TestComponent = () => {
      return <div data-testid="test-component">Hello World</div>;
    };

    // Try to render
    const { container } = render(<TestComponent />);
    
    // Debug output
    console.log('Container:', container);
    console.log('Container innerHTML:', container.innerHTML);
    
    // Check if element exists
    const testElement = screen.queryByTestId('test-component');
    console.log('Test element found:', testElement);
    
    expect(testElement).toBeInTheDocument();
    expect(screen.getByText('Hello World')).toBeInTheDocument();
  });
});