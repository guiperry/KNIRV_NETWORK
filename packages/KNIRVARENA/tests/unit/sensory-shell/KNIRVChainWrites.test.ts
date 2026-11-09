import { KNIRVChainIntegration } from '../../../src/sensory-shell/KNIRVChainIntegration';
import { submitChainWrite } from '../../../src/services/ChainWrites';

jest.mock('../../../src/services/ChainWrites', () => ({
  ...jest.requireActual('../../../src/services/ChainWrites'),
  submitChainWrite: jest.fn(),
}));

const submitMock = submitChainWrite as unknown as jest.Mock;
const RPC = 'https://testnet-gateway.knirv.com/api/chain';
const OWNER = 'knirv1owner';

describe('KNIRVChainIntegration writes', () => {
  beforeEach(() => submitMock.mockReset());

  it('registers a skill as a signed TOOL capability through the gateway', async () => {
    submitMock.mockResolvedValueOnce('hash-1');
    const chain = new KNIRVChainIntegration({ rpcUrl: RPC, onApprovalUri: undefined });

    const skillId = await chain.registerSkill({
      name: 'Skill One', skillType: 'TOOL', capabilities: [], requirements: {}, owner: OWNER, usageFee: '3',
      validationStatus: 'pending', performanceMetrics: { successRate: 0, averageLatency: 0, totalInvocations: 0, lastUpdated: 0 },
    });

    expect(skillId).toMatch(/^skill-/);
    const [write, options] = submitMock.mock.calls[0];
    expect(write.type).toBe('register_capability_txn');
    expect(write.sender).toBe(OWNER);
    const registration = JSON.parse(new TextDecoder().decode(write.data));
    expect(registration).toEqual(expect.objectContaining({ capabilityID: skillId, name: 'Skill One', capabilityType: 'TOOL', gasFeeNRN: 3 }));
    expect(options.gatewayUrl).toBe('https://testnet-gateway.knirv.com');
    expect(chain.getSkill(skillId)).toEqual(expect.objectContaining({ id: skillId, validationStatus: 'pending' }));
  });

  it('refuses a skill type the chain does not define', async () => {
    const chain = new KNIRVChainIntegration({ rpcUrl: RPC, onApprovalUri: undefined });
    await expect(chain.registerSkill({
      name: 'x', skillType: 'LLM', capabilities: [], requirements: {}, owner: OWNER, usageFee: '0',
      validationStatus: 'pending', performanceMetrics: { successRate: 0, averageLatency: 0, totalInvocations: 0, lastUpdated: 0 },
    })).rejects.toThrow('not a KNIRV Chain capability type');
    expect(submitMock).not.toHaveBeenCalled();
  });

  it('records a skill invocation as a signed invoke transaction paying the usage fee', async () => {
    submitMock.mockResolvedValueOnce('invoke-hash');
    const chain = new KNIRVChainIntegration({ rpcUrl: RPC, onApprovalUri: undefined });

    await expect(chain.invokeSkillOnChain('skill-1', OWNER, '5', {})).resolves.toBe('invoke-hash');

    const [write] = submitMock.mock.calls[0];
    expect(write.type).toBe('invoke_capability_txn');
    expect(write.sender).toBe(OWNER);
    expect(write.fee).toBe(5);
    const invocation = JSON.parse(new TextDecoder().decode(write.data));
    expect(invocation.contextRecord).toEqual(expect.objectContaining({ capabilityID: 'skill-1', interactionType: 'Invoke', initiator: OWNER }));
    expect(chain.getSkillInvocations('skill-1')).toEqual([expect.objectContaining({ transactionHash: 'invoke-hash', amountBurned: '5' })]);
  });

  it('still refuses NRN transfers, which the chain does not accept', async () => {
    const chain = new KNIRVChainIntegration({ rpcUrl: RPC, onApprovalUri: undefined });
    await expect(chain.transferNRN('a', 'b', '1')).rejects.toThrow('Transaction Chain');
    expect(submitMock).not.toHaveBeenCalled();
  });

  it('still refuses LLM model registration, which the chain has no registry for', async () => {
    const chain = new KNIRVChainIntegration({ rpcUrl: RPC, onApprovalUri: undefined });
    await expect(chain.registerLLMModel({ name: 'm' } as never)).rejects.toThrow('no LLM model registry');
    expect(submitMock).not.toHaveBeenCalled();
  });
});
