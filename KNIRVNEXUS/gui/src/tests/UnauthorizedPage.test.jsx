import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { BrowserRouter } from 'react-router-dom';
import UnauthorizedPage from '../components/UnauthorizedPage';

describe('UnauthorizedPage', () => {
  const renderUnauthorizedPage = () => {
    return render(
      <BrowserRouter>
        <UnauthorizedPage />
      </BrowserRouter>
    );
  };

  test('should render unauthorized message', () => {
    renderUnauthorizedPage();

    // Check for unauthorized message
    expect(screen.getByText('Access Denied')).toBeInTheDocument();
    expect(screen.getByText(/You don't have permission to access this page/)).toBeInTheDocument();
  });

  test('should render with proper styling classes', () => {
    const { container } = renderUnauthorizedPage();

    // Check that the component renders and has the main container
    expect(container.querySelector('.min-h-screen')).toBeInTheDocument();
    expect(container.querySelector('.bg-gradient-to-br')).toBeInTheDocument();

    // Check that the inner card div exists
    expect(container.querySelector('.bg-slate-800\\/90')).toBeInTheDocument();
    expect(container.querySelector('.backdrop-blur-sm')).toBeInTheDocument();
  });
});
