import { knirvEngineBrowserLink, knirvEngineDeepLink, openInKnirvEngine } from '../knirvEngineLink';

describe('knirvEngineLink', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('builds desktop and browser links for the network error-node id', () => {
    expect(knirvEngineDeepLink('err/1 a')).toBe('knirvengine://error-node/err%2F1%20a');
    expect(knirvEngineBrowserLink('err-1', 'http://localhost:8123')).toBe('http://localhost:8123/dashboard?errorNode=err-1');
  });

  it('launches the knirvengine:// link and reports it handled when the page loses focus', async () => {
    const clicked: string[] = [];
    const click = jest.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      clicked.push(this.href);
    });
    const win = new EventTarget();
    const result = openInKnirvEngine('err-1', { win: win as unknown as Window });
    win.dispatchEvent(new Event('blur'));
    await expect(result).resolves.toBe(true);
    expect(clicked).toEqual(['knirvengine://error-node/err-1']);
    click.mockRestore();
  });

  it('reports the link unhandled when no desktop engine answers', async () => {
    const click = jest.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined);
    const result = openInKnirvEngine('err-1', { waitMs: 500, win: new EventTarget() as unknown as Window });
    jest.advanceTimersByTime(500);
    await expect(result).resolves.toBe(false);
    click.mockRestore();
  });
});
