import {
  addToolReport,
  clearToolReports,
  createCombinedToolReport,
  getToolReports,
} from '../services/toolReports';

describe('tool reports', () => {
  beforeEach(() => {
    clearToolReports();
  });

  test('persists an execution record while redacting sensitive arguments', () => {
    addToolReport({
      tool: 'jwt-tool',
      execution: 'scan',
      status: 'completed',
      sessionID: 'sandbox-1',
      startedAt: '2026-08-27T10:00:00.000Z',
      completedAt: '2026-08-27T10:00:01.000Z',
      durationMs: 1000,
      args: { token: 'private-value', targetPath: '/workspace/app' },
      output: 'No issues found',
    });

    const [report] = getToolReports();
    expect(report.args).toEqual({ token: '[redacted]', targetPath: '/workspace/app' });
    expect(localStorage.getItem('knirvengine.tool-reports.v1')).toContain('jwt-tool');
  });

  test('creates a complete Markdown report from recorded tool results', () => {
    addToolReport({
      tool: 'semgrep', execution: 'analysis', status: 'failed', sessionID: 'sandbox-2',
      startedAt: '2026-08-27T10:00:00.000Z', completedAt: '2026-08-27T10:00:02.000Z',
      output: 'partial output', error: 'scan failed',
    });

    const markdown = createCombinedToolReport(getToolReports());
    expect(markdown).toContain('# KNIRVENGINE Tool Reports');
    expect(markdown).toContain('semgrep — failed');
    expect(markdown).toContain('scan failed');
    expect(markdown).toContain('partial output');
  });
});
