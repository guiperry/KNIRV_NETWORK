import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { DatasetSubmissionForm, parseRecords, MAX_DATASET_RECORDS } from '../../../src/components/DatasetSubmissionForm';

const targets = [{ errorNodeId: 'graph-err-1', label: 'Memory leak in loader' }];
const setRecords = (text: string) => fireEvent.change(screen.getByLabelText(/Records/), { target: { value: text } });

describe('parseRecords', () => {
  it('accepts an array or a single object and enforces the limits', () => {
    expect(parseRecords('[{"prompt":"p"}]').records).toHaveLength(1);
    expect(parseRecords('{"prompt":"p"}').records).toHaveLength(1);
    expect(parseRecords('not json').error).toMatch(/not valid JSON/);
    expect(parseRecords('[]').error).toMatch(/at least one/);
    expect(parseRecords('["text"]').error).toMatch(/JSON object/);
    expect(parseRecords(JSON.stringify(Array(MAX_DATASET_RECORDS + 1).fill({ text: 't' }))).error).toMatch(/at most/);
  });
});

describe('DatasetSubmissionForm', () => {
  it('explains that datasets need a registered error node', () => {
    render(<DatasetSubmissionForm targets={[]} onSubmit={jest.fn()} />);
    expect(screen.getByText(/Submit an error first/)).toBeInTheDocument();
  });

  it('pre-fills the format example and submits the edited records', async () => {
    const onSubmit = jest.fn().mockResolvedValue({
      error_node_id: 'graph-err-1', format: 'preference', records: 1,
      names: ['preference', 'preference.rejected'],
      datasets: [{ brackets: 12 }, { brackets: 9 }],
    });
    render(<DatasetSubmissionForm targets={targets} onSubmit={onSubmit} />);

    expect((screen.getByLabelText(/Records/) as HTMLTextAreaElement).value).toContain('"completion"');
    fireEvent.change(screen.getByLabelText(/Format/), { target: { value: 'preference' } });
    expect((screen.getByLabelText(/Records/) as HTMLTextAreaElement).value).toContain('"rejected"');

    setRecords('[{"prompt":"Sort by value","chosen":"sorted(items, key=v)","rejected":"items.sort()"}]');
    fireEvent.click(screen.getByRole('button', { name: /Submit dataset/ }));

    expect(await screen.findByRole('status')).toHaveTextContent('Stored 1 record as 21 brackets (preference, preference.rejected)');
    expect(onSubmit).toHaveBeenCalledWith('graph-err-1', 'preference', [{ prompt: 'Sort by value', chosen: 'sorted(items, key=v)', rejected: 'items.sort()' }]);
  });

  it('shows the server validation message', async () => {
    const onSubmit = jest.fn().mockRejectedValue({ response: { data: { error: 'record 0: invalid dataset submission: completion is required' } } });
    render(<DatasetSubmissionForm targets={targets} onSubmit={onSubmit} />);
    setRecords('[{"prompt":"p"}]');
    fireEvent.click(screen.getByRole('button', { name: /Submit dataset/ }));
    expect(await screen.findByRole('alert')).toHaveTextContent('completion is required');
  });

  it('blocks submission of invalid JSON', () => {
    render(<DatasetSubmissionForm targets={targets} onSubmit={jest.fn()} />);
    setRecords('[{');
    expect(screen.getByRole('button', { name: /Submit dataset/ })).toBeDisabled();
  });
});
