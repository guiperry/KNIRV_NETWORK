import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import VectorCard from '../components/VectorCard';
import { NRVVector } from '../services/api';

const mockVector: NRVVector = {
  id: 'vector_123456789',
  source_peer: 'peer_abcdef123',
  target_hash: 'hash_987654321abcdef',
  coordinates: [1.2345, 2.3456, 3.4567, 4.5678],
  confidence: 0.85,
  timestamp: '2024-01-15T10:30:00Z',
  metadata: {
    type: 'skill_vector',
    priority: 'high',
    category: 'nlp'
  }
};

const renderVectorCard = (vector: NRVVector = mockVector) => {
  return render(
    <BrowserRouter>
      <VectorCard vector={vector} />
    </BrowserRouter>
  );
};

describe('VectorCard Component', () => {
  it('should render vector information correctly', () => {
    renderVectorCard();

    // Check vector ID display
    expect(screen.getByText('Vector vector_1')).toBeInTheDocument();
    
    // Check confidence percentage
    expect(screen.getByText('Confidence: 85.0%')).toBeInTheDocument();
    
    // Check target hash (truncated)
    expect(screen.getByText('hash_987654321...def')).toBeInTheDocument();
    
    // Check source peer (truncated)
    expect(screen.getByText('peer_abc...123')).toBeInTheDocument();
  });

  it('should format coordinates correctly', () => {
    renderVectorCard();

    // Check that coordinates are displayed with proper formatting
    expect(screen.getByText('[1.23, 2.35, 3.46]')).toBeInTheDocument();
  });

  it('should format timestamp correctly', () => {
    renderVectorCard();

    // Check that timestamp is formatted as locale string
    const timestampElement = screen.getByText(/1\/15\/2024/);
    expect(timestampElement).toBeInTheDocument();
  });

  it('should be clickable and link to vector details', () => {
    renderVectorCard();

    const linkElement = screen.getByRole('link');
    expect(linkElement).toHaveAttribute('href', '/vector/vector_123456789');
  });

  it('should handle different confidence levels', () => {
    const lowConfidenceVector = { ...mockVector, confidence: 0.23 };
    renderVectorCard(lowConfidenceVector);

    expect(screen.getByText('Confidence: 23.0%')).toBeInTheDocument();
  });

  it('should handle vectors with fewer coordinates', () => {
    const shortVector = { ...mockVector, coordinates: [1.1, 2.2] };
    renderVectorCard(shortVector);

    expect(screen.getByText('[1.10, 2.20]')).toBeInTheDocument();
  });

  it('should truncate long hashes correctly', () => {
    const longHashVector = {
      ...mockVector,
      target_hash: 'very_long_hash_that_should_be_truncated_properly_123456789abcdef'
    };
    renderVectorCard(longHashVector);

    expect(screen.getByText('very_long_ha...cdef')).toBeInTheDocument();
  });

  it('should display vector icon', () => {
    renderVectorCard();

    // Check for the Target icon (vector icon)
    const iconElement = screen.getByTestId('vector-icon') || 
                       document.querySelector('[data-lucide="target"]');
    expect(iconElement).toBeTruthy();
  });

  it('should have proper styling classes', () => {
    renderVectorCard();

    const linkElement = screen.getByRole('link');
    expect(linkElement).toHaveClass('block');
    expect(linkElement).toHaveClass('bg-gray-700/30');
    expect(linkElement).toHaveClass('hover:bg-gray-700/50');
  });

  it('should handle edge case with very high confidence', () => {
    const perfectVector = { ...mockVector, confidence: 1.0 };
    renderVectorCard(perfectVector);

    expect(screen.getByText('Confidence: 100.0%')).toBeInTheDocument();
  });

  it('should handle edge case with zero confidence', () => {
    const zeroConfidenceVector = { ...mockVector, confidence: 0.0 };
    renderVectorCard(zeroConfidenceVector);

    expect(screen.getByText('Confidence: 0.0%')).toBeInTheDocument();
  });

  it('should display all required vector information sections', () => {
    renderVectorCard();

    // Check for main sections
    expect(screen.getByText(/Target:/)).toBeInTheDocument();
    expect(screen.getByText(/Source:/)).toBeInTheDocument();
    expect(screen.getByText(/Coordinates:/)).toBeInTheDocument();
    expect(screen.getByText(/Confidence:/)).toBeInTheDocument();
  });
});
