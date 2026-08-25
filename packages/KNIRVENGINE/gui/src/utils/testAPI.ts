// Test API utilities for integration testing

export interface ADKTestResult {
  success: boolean;
  message: string;
  data?: unknown;
  error?: string;
}

export interface AgentTestConfig {
  name: string;
  template: string;
  config: Record<string, unknown>;
}

/**
 * Test ADK integration functionality
 */
export async function testADKIntegration(): Promise<ADKTestResult> {
  try {
    // Test basic API connectivity
    const response = await fetch('/api/v1/health');
    
    if (!response.ok) {
      return {
        success: false,
        message: 'API health check failed',
        error: `HTTP ${response.status}: ${response.statusText}`
      };
    }

    const healthData = await response.json();
    
    return {
      success: true,
      message: 'ADK integration test passed',
      data: healthData
    };
  } catch (error) {
    return {
      success: false,
      message: 'ADK integration test failed',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

/**
 * Test agent creation functionality
 */
export async function testAgentCreation(config: AgentTestConfig): Promise<ADKTestResult> {
  try {
    const response = await fetch('/api/v1/agents', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config)
    });

    if (!response.ok) {
      return {
        success: false,
        message: 'Agent creation failed',
        error: `HTTP ${response.status}: ${response.statusText}`
      };
    }

    const agentData = await response.json();
    
    return {
      success: true,
      message: 'Agent creation test passed',
      data: agentData
    };
  } catch (error) {
    return {
      success: false,
      message: 'Agent creation test failed',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

/**
 * Test agent inference functionality
 */
export async function testAgentInference(agentId: string, input: string): Promise<ADKTestResult> {
  try {
    const response = await fetch(`/api/v1/agents/${agentId}/inference`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ input })
    });

    if (!response.ok) {
      return {
        success: false,
        message: 'Agent inference failed',
        error: `HTTP ${response.status}: ${response.statusText}`
      };
    }

    const inferenceData = await response.json();
    
    return {
      success: true,
      message: 'Agent inference test passed',
      data: inferenceData
    };
  } catch (error) {
    return {
      success: false,
      message: 'Agent inference test failed',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

/**
 * Test agent plugin building functionality
 */
export async function testAgentPluginBuild(config: AgentTestConfig): Promise<ADKTestResult> {
  try {
    const response = await fetch('/api/v1/agents/build', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config)
    });

    if (!response.ok) {
      return {
        success: false,
        message: 'Agent plugin build failed',
        error: `HTTP ${response.status}: ${response.statusText}`
      };
    }

    const buildData = await response.json();
    
    return {
      success: true,
      message: 'Agent plugin build test passed',
      data: buildData
    };
  } catch (error) {
    return {
      success: false,
      message: 'Agent plugin build test failed',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

/**
 * Run comprehensive ADK test suite
 */
export async function runADKTestSuite(): Promise<ADKTestResult[]> {
  const results: ADKTestResult[] = [];

  // Test 1: Basic connectivity
  results.push(await testADKIntegration());

  // Test 2: Agent creation
  const testConfig: AgentTestConfig = {
    name: 'Test Agent',
    template: 'basic',
    config: {
      description: 'A test agent for integration testing',
      capabilities: ['test']
    }
  };
  results.push(await testAgentCreation(testConfig));

  // Test 3: Plugin building
  results.push(await testAgentPluginBuild(testConfig));

  return results;
}
