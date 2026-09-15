import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PanelErrorBoundary } from '../panel-error-boundary';
import { PanelLoading } from '../panel-loading';
import { PanelEmpty } from '../panel-empty';

const Boom: React.FC = () => {
  throw new Error('panel exploded');
};

const Calm: React.FC = () => <div>panel content</div>;

describe('Panel primitives (FE-3)', () => {
  it('renders children when no error is thrown', () => {
    render(
      <PanelErrorBoundary>
        <Calm />
      </PanelErrorBoundary>
    );
    expect(screen.getByText('panel content')).toBeInTheDocument();
  });

  it('shows the boundary retry UI instead of a crashed tree when a child throws', () => {
    const saved = console.error;
    console.error = jest.fn();
    render(
      <PanelErrorBoundary>
        <Boom />
      </PanelErrorBoundary>
    );
    expect(screen.getByText('Panel Error')).toBeInTheDocument();
    expect(screen.getByText('panel exploded')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    console.error = saved;
  });

  it('recovers on Retry when the child stops throwing', async () => {
    const saved = console.error;
    console.error = jest.fn();
    let shouldThrow = true;

    const Flaky: React.FC = () => {
      if (shouldThrow) {
        throw new Error('flaky failure');
      }
      return <div>recovered</div>;
    };

    render(
      <PanelErrorBoundary>
        <Flaky />
      </PanelErrorBoundary>
    );
    expect(screen.getByText('Panel Error')).toBeInTheDocument();

    shouldThrow = false;
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(screen.getByText('recovered')).toBeInTheDocument();
    console.error = saved;
  });

  it('PanelLoading renders a spinner and message', () => {
    const { container } = render(<PanelLoading message="Fetching telemetry" />);
    expect(screen.getByText('Fetching telemetry')).toBeInTheDocument();
    expect(container.querySelector('[data-slot="panel-loading"]')).toBeTruthy();
  });

  it('PanelEmpty renders title, description and action', async () => {
    const onAction = jest.fn();
    const { container } = render(
      <PanelEmpty
        title="No DVE nodes"
        description="Connect a DVE to get started"
        actionLabel="Connect"
        onAction={onAction}
      />
    );
    expect(screen.getByText('No DVE nodes')).toBeInTheDocument();
    expect(screen.getByText('Connect a DVE to get started')).toBeInTheDocument();
    expect(container.querySelector('[data-slot="panel-empty"]')).toBeTruthy();
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }));
    expect(onAction).toHaveBeenCalledTimes(1);
  });
});