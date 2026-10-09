import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { NodeSubmissionForm, MAX_NODE_DESCRIPTION } from '../../../src/components/NodeSubmissionForm';

const describeAs = (text: string) => fireEvent.change(screen.getByLabelText(/Description/), { target: { value: text } });
const submitButton = () => screen.getByRole('button', { name: /Submit/ });

describe('NodeSubmissionForm', () => {
  it('submits what the user typed for a context node', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    const onDone = jest.fn();
    render(<NodeSubmissionForm kind="context" onSubmit={onSubmit} onDone={onDone} />);

    expect(submitButton()).toBeDisabled();
    fireEvent.change(screen.getByLabelText(/Context type/), { target: { value: 'tool' } });
    describeAs('  A jq tool that filters JSON  ');
    fireEvent.click(submitButton());

    await waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(onSubmit).toHaveBeenCalledWith({ kind: 'context', contextType: 'tool', description: 'A jq tool that filters JSON' });
  });

  it('requires an error type for error nodes and sends the severity', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    render(<NodeSubmissionForm kind="error" onSubmit={onSubmit} onDone={jest.fn()} />);

    describeAs('Build fails on node 22');
    expect(submitButton()).toBeDisabled();
    fireEvent.change(screen.getByLabelText(/Error type/), { target: { value: 'build failure' } });
    fireEvent.change(screen.getByLabelText(/Severity/), { target: { value: '5' } });
    fireEvent.click(submitButton());

    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith({
      kind: 'error', errorType: 'build failure', description: 'Build fails on node 22', severity: 5,
    }));
  });

  it('shows a registration failure and stays open', async () => {
    const onDone = jest.fn();
    render(<NodeSubmissionForm kind="idea" onSubmit={jest.fn().mockRejectedValue(new Error('Could not register on KNIRVGRAPH: down'))} onDone={onDone} />);

    describeAs('Shared skill marketplace');
    fireEvent.click(submitButton());

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not register on KNIRVGRAPH: down');
    expect(onDone).not.toHaveBeenCalled();
  });

  it('refuses blank or over-long descriptions', () => {
    render(<NodeSubmissionForm kind="idea" onSubmit={jest.fn()} onDone={jest.fn()} />);
    describeAs('   ');
    expect(submitButton()).toBeDisabled();
    expect(screen.getByLabelText(/Description/)).toHaveAttribute('maxLength', String(MAX_NODE_DESCRIPTION));
  });
});
