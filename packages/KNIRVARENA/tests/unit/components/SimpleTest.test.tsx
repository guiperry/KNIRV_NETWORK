import * as React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// Simple component to test basic rendering
const SimpleComponent: React.FC = () => {
  return <div>Hello World</div>;
};

describe('Simple Test', () => {
  it('should render a simple component', () => {
    render(<SimpleComponent />);
    expect(screen.getByText('Hello World')).toBeInTheDocument();
  });
});
