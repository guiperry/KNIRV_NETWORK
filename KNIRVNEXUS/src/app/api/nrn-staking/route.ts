import { NextRequest, NextResponse } from 'next/server';

// Required for static export
export const dynamic = 'force-static';
export const revalidate = false;

interface NRNStaking {
  total_staked: number;
  apy: number;
  rewards_24h: number;
  validators_count: number;
  slashing_events: number;
  staking_distribution: Array<{
    range: string;
    count: number;
    percentage: number;
  }>;
  reward_history: Array<{
    date: string;
    amount: number;
    type: 'reward' | 'slash';
    reason?: string;
  }>;
  performance_metrics: {
    network_participation_rate: number;
    average_uptime: number;
    successful_validations: number;
    failed_validations: number;
  };
}

// Mock data for NRN staking
const mockNRNStaking: NRNStaking = {
  total_staked: 2500000,
  apy: 12.5,
  rewards_24h: 856.25,
  validators_count: 45,
  slashing_events: 0,
  staking_distribution: [
    { range: "0-10K NRN", count: 12, percentage: 26.7 },
    { range: "10K-50K NRN", count: 18, percentage: 40.0 },
    { range: "50K-100K NRN", count: 10, percentage: 22.2 },
    { range: "100K+ NRN", count: 5, percentage: 11.1 }
  ],
  reward_history: [
    { date: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), amount: 856.25, type: 'reward' },
    { date: new Date(Date.now() - 48 * 60 * 60 * 1000).toISOString(), amount: 892.50, type: 'reward' },
    { date: new Date(Date.now() - 72 * 60 * 60 * 1000).toISOString(), amount: 915.75, type: 'reward' }
  ],
  performance_metrics: {
    network_participation_rate: 94.5,
    average_uptime: 99.2,
    successful_validations: 15420,
    failed_validations: 892
  }
};

export async function GET(request: NextRequest) {
  try {
    // Simulate real-time staking updates
    const updatedStaking = {
      ...mockNRNStaking,
      total_staked: mockNRNStaking.total_staked + (Math.random() - 0.5) * 1000,
      rewards_24h: mockNRNStaking.rewards_24h + (Math.random() - 0.5) * 10,
      apy: Math.max(5, Math.min(20, mockNRNStaking.apy + (Math.random() - 0.5) * 0.1)),
      performance_metrics: {
        ...mockNRNStaking.performance_metrics,
        network_participation_rate: Math.max(90, Math.min(100, mockNRNStaking.performance_metrics.network_participation_rate + (Math.random() - 0.5) * 0.5)),
        average_uptime: Math.max(95, Math.min(100, mockNRNStaking.performance_metrics.average_uptime + (Math.random() - 0.5) * 0.2))
      }
    };

    return NextResponse.json({
      success: true,
      data: updatedStaking,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to fetch NRN staking data' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { action, parameters } = body;
    
    if (!action) {
      return NextResponse.json(
        { success: false, error: 'Action is required' },
        { status: 400 }
      );
    }

    let responseMessage = '';
    
    switch (action) {
      case 'update_apy':
        if (parameters?.apy !== undefined) {
          mockNRNStaking.apy = Math.max(0, Math.min(100, parameters.apy));
          responseMessage = `APY updated to ${mockNRNStaking.apy}%`;
        } else {
          return NextResponse.json(
            { success: false, error: 'APY value is required' },
            { status: 400 }
          );
        }
        break;
      
      case 'add_slash_event':
        if (parameters?.amount && parameters?.reason) {
          mockNRNStaking.slashing_events += 1;
          mockNRNStaking.reward_history.unshift({
            date: new Date().toISOString(),
            amount: -Math.abs(parameters.amount),
            type: 'slash',
            reason: parameters.reason
          });
          responseMessage = `Slash event recorded: ${parameters.reason}`;
        } else {
          return NextResponse.json(
            { success: false, error: 'Amount and reason are required for slash event' },
            { status: 400 }
          );
        }
        break;
      
      case 'add_reward':
        if (parameters?.amount !== undefined) {
          mockNRNStaking.rewards_24h += Math.abs(parameters.amount);
          mockNRNStaking.reward_history.unshift({
            date: new Date().toISOString(),
            amount: Math.abs(parameters.amount),
            type: 'reward'
          });
          responseMessage = `Reward of ${Math.abs(parameters.amount)} NRN added`;
        } else {
          return NextResponse.json(
            { success: false, error: 'Amount is required for reward' },
            { status: 400 }
          );
        }
        break;
      
      case 'update_validators':
        if (parameters?.count !== undefined) {
          mockNRNStaking.validators_count = Math.max(0, parameters.count);
          responseMessage = `Validator count updated to ${mockNRNStaking.validators_count}`;
        } else {
          return NextResponse.json(
            { success: false, error: 'Validator count is required' },
            { status: 400 }
          );
        }
        break;
      
      case 'update_performance_metrics':
        if (parameters?.metrics) {
          mockNRNStaking.performance_metrics = {
            ...mockNRNStaking.performance_metrics,
            ...parameters.metrics
          };
          responseMessage = 'Performance metrics updated';
        } else {
          return NextResponse.json(
            { success: false, error: 'Metrics are required' },
            { status: 400 }
          );
        }
        break;
      
      default:
        return NextResponse.json(
          { success: false, error: 'Invalid action' },
          { status: 400 }
        );
    }
    
    return NextResponse.json({
      success: true,
      data: mockNRNStaking,
      message: responseMessage,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to process NRN staking action' },
      { status: 500 }
    );
  }
}