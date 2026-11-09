import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ErrorNodeModal } from '../../components/modals/ErrorNodeModal';
import * as engineLink from '../testlabLink';

jest.mock('../KNIRVSERVERClient', () => ({
  getKNIRVSERVERClient: () => ({ getErrorNodeTests: jest.fn().mockResolvedValue(null) }),
}));

const nrv = (overrides: Record<string, unknown>) => ({
  id: 'nrv-1',
  problemDescription: 'User-submitted error for SkillNode training',
  sourceID: 'KNIRV-CONTROLLER-user',
  inputType: 'Error',
  temporalContext: new Date(),
  severity: 'high',
  suggestedSolutionType: 'skill-training',
  status: 'Identified',
  ...overrides,
});

const renderModal = (nrvs: unknown[]) =>
  render(
    <ErrorNodeModal isOpen onClose={jest.fn()} nrvs={nrvs as never} selectedNRV={null} nrnBalance={100} onDeployAgent={jest.fn()} />
  );

describe('ErrorNodeModal → Testlab', () => {
  it('opens a network error node in Testlab', async () => {
    const open = jest.spyOn(engineLink, 'openInTestlab').mockResolvedValue(true);
    renderModal([nrv({ networkErrorNodeId: 'graph-err-7' })]);
    fireEvent.click(screen.getByText(/Error 1:/));
    fireEvent.click(await screen.findByRole('button', { name: /open in testlab/i }));
    await waitFor(() => expect(open).toHaveBeenCalledWith('graph-err-7'));
    expect(await screen.findByText(/opened in testlab/i)).toBeInTheDocument();
  });

  it('offers the browser engine when the desktop app does not answer', async () => {
    jest.spyOn(engineLink, 'openInTestlab').mockResolvedValue(false);
    renderModal([nrv({ networkErrorNodeId: 'graph-err-7' })]);
    fireEvent.click(screen.getByText(/Error 1:/));
    fireEvent.click(await screen.findByRole('button', { name: /open in testlab/i }));
    const link = await screen.findByRole('link', { name: /open it in the browser engine/i });
    expect(link).toHaveAttribute('href', expect.stringContaining('/dashboard?errorNode=graph-err-7'));
  });

  it('is disabled for errors that only exist in this client', async () => {
    renderModal([nrv({})]);
    fireEvent.click(screen.getByText(/Error 1:/));
    expect(await screen.findByRole('button', { name: /open in testlab/i })).toBeDisabled();
  });
});
