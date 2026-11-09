import { KNIRVChainIntegration } from '../../../src/sensory-shell/KNIRVChainIntegration';

const RPC = 'https://testnet-gateway.knirv.com/api/chain';

const respond = (status: number, body?: unknown): Response =>
  ({ ok: status >= 200 && status < 300, status, json: async () => body }) as unknown as Response;

describe('KNIRVChainIntegration (KNIRVGATEWAY chain routes)', () => {
  let fetchMock: jest.Mock;
  let chain: KNIRVChainIntegration;

  const urls = (): string[] => fetchMock.mock.calls.map((call) => String(call[0]));

  beforeEach(() => {
    fetchMock = jest.fn();
    (global as unknown as { fetch: jest.Mock }).fetch = fetchMock;
    chain = new KNIRVChainIntegration({ rpcUrl: RPC });
  });

  afterEach(async () => {
    await chain.disconnect();
  });

  it('connects through the gateway and loads skills as TOOL capabilities', async () => {
    fetchMock
      .mockResolvedValueOnce(respond(200, { status: 'ok' }))
      .mockResolvedValueOnce(respond(200, { chain_id: 'knirv-1', height: 5, block_hash: 'ab', timestamp: 1 }))
      .mockResolvedValueOnce(respond(200, {
        capabilities: [
          { id: 'skill-1', name: 'Skill One', capabilityType: 'TOOL', owner: 'knirv1owner', gasFeeNRN: 3, timestamp: 9 },
        ],
      }));

    await chain.initialize();

    expect(urls()).toEqual([
      `${RPC}/health`,
      `${RPC}/chain/head`,
      `${RPC}/mcp/capability/list?type=TOOL`,
    ]);
    expect(chain.isChainConnected()).toBe(true);
    expect(chain.getSkills()).toEqual([
      expect.objectContaining({
        id: 'skill-1',
        name: 'Skill One',
        skillType: 'TOOL',
        owner: 'knirv1owner',
        usageFee: '3',
        validationStatus: 'pending',
      }),
    ]);
  });

  it('fails to initialize when the chain health route is down', async () => {
    fetchMock.mockResolvedValueOnce(respond(503));

    await expect(chain.initialize()).rejects.toThrow('health check failed: HTTP 503');
    expect(chain.isChainConnected()).toBe(false);
  });

  it('verifies a skill by reading the capability route', async () => {
    fetchMock.mockResolvedValueOnce(respond(200, { status: 'success', capability: { id: 'skill-1' } }));

    await expect(chain.verifySkill('skill-1')).resolves.toBe(true);
    expect(urls()).toEqual([`${RPC}/mcp/capability/get?id=skill-1`]);
  });

  it('reports a missing skill as not verified (404)', async () => {
    fetchMock.mockResolvedValueOnce(respond(404));

    await expect(chain.verifySkill('missing')).resolves.toBe(false);
  });

  it('reports a failed verification request as not verified', async () => {
    fetchMock.mockRejectedValueOnce(new Error('network down'));

    await expect(chain.verifySkill('skill-1')).resolves.toBe(false);
  });

  it('reads an account NRN balance from the balance route', async () => {
    fetchMock.mockResolvedValueOnce(respond(200, { address: 'knirv1abc', balance: 42 }));

    await expect(chain.getNRNBalance('knirv1abc')).resolves.toBe('42');
    expect(urls()).toEqual([`${RPC}/accounts/balance?address=knirv1abc`]);
  });

  it('throws when the balance route fails rather than reporting zero', async () => {
    fetchMock.mockResolvedValueOnce(respond(500));

    await expect(chain.getNRNBalance('knirv1abc')).rejects.toThrow('HTTP 500');
  });

  it('derives consensus from the chain head', async () => {
    fetchMock.mockResolvedValueOnce(respond(200, { chain_id: 'knirv-1', height: 9, block_hash: 'cafe', timestamp: 1 }));

    const consensus = await chain.getNetworkConsensus();

    expect(urls()).toEqual([`${RPC}/chain/head`]);
    expect(consensus).toEqual(expect.objectContaining({ chainId: 'knirv-1', blockHeight: 9, blockHash: 'cafe' }));
  });

  it.each([
    ['registerLLMModel', () => chain.registerLLMModel({ name: 'x' } as never)],
    ['transferNRN', () => chain.transferNRN('a', 'b', '1')],
  ])('%s is rejected without calling the network', async (_name, call) => {
    await expect(call()).rejects.toThrow('is not available');
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('has no LLM registry to read', () => {
    expect(chain.getLLMModels()).toEqual([]);
    expect(chain.getLLMModel('anything')).toBeNull();
  });
});
