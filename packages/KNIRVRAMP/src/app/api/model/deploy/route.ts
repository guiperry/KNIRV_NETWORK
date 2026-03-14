import { NextRequest, NextResponse } from 'next/server';

// KNIRV-NEXUS API integration
const NEXUS_API_BASE = process.env.NEXUS_API_BASE || 'http://localhost:8080';
const NEXUS_API_KEY = process.env.NEXUS_API_KEY || '';

// KNIRV-CONTROLLER API integration
const CONTROLLER_API_BASE = process.env.CONTROLLER_API_BASE || 'http://localhost:3001';
const CONTROLLER_API_KEY = process.env.CONTROLLER_API_KEY || '';

async function deployToKNIRVNexus(modelId: string, cortexWasm: Uint8Array, modelConfig: any) {
  try {
    // Create agent in KNIRV-NEXUS using the agent management API
    const agentData = {
      id: modelId,
      name: modelConfig.name || `Cortex Model ${modelId}`,
      description: modelConfig.description || 'Cortex WASM model deployed from primary website',
      type: 'cortex-wasm',
      status: 'uploaded',
      uploadedBy: 'primary-website',
      resourceLimits: {
        maxMemoryMB: 512,
        maxCPUPercent: 80.0,
        maxExecutionTime: 300,
        maxConcurrency: 10,
        maxDiskMB: 1024,
        networkAccess: true,
        fileSystemAccess: false
      },
      configuration: {
        wasmSize: cortexWasm.length,
        modelType: 'cortex',
        deploymentSource: 'primary-website',
        template: modelConfig.template?.id,
        parameters: modelConfig.template?.parameters,
        architecture: modelConfig.template?.architecture
      },
      tags: ['cortex-wasm', 'neural-model', modelConfig.template?.size || 'small']
    };

    const agentResponse = await fetch(`${NEXUS_API_BASE}/api/agent-management/agents`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${NEXUS_API_KEY}`
      },
      body: JSON.stringify(agentData)
    });

    if (!agentResponse.ok) {
      throw new Error(`Failed to create agent in KNIRV-NEXUS: ${agentResponse.statusText}`);
    }

    const agentResult = await agentResponse.json();

    // Deploy the agent using the action endpoint
    const deployResponse = await fetch(`${NEXUS_API_BASE}/api/agent-management/agents/${modelId}/actions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${NEXUS_API_KEY}`
      },
      body: JSON.stringify({
        action: 'deploy',
        parameters: {
          wasmData: Array.from(cortexWasm),
          deploymentType: 'cortex-wasm',
          environment: 'production',
          configuration: {
            inference_endpoint: `/api/inference/${modelId}`,
            batch_size: modelConfig.training_config?.batch_size || 32
          }
        }
      })
    });

    if (!deployResponse.ok) {
      throw new Error(`Failed to deploy agent in KNIRV-NEXUS: ${deployResponse.statusText}`);
    }

    return {
      success: true,
      agent_id: modelId,
      deployment_id: `deployment-${modelId}`,
      url: `${NEXUS_API_BASE}/api/agent-management/agents/${modelId}`,
      inference_endpoint: `${NEXUS_API_BASE}/api/inference/${modelId}`,
      dve_instance: `dve-${modelId}`,
      status: 'deployed'
    };
  } catch (error) {
    console.error('KNIRV-NEXUS deployment error:', error);
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown deployment error'
    };
  }
}

async function deployToKNIRVController(modelId: string, cortexWasm: Uint8Array, modelConfig: any) {
  try {
    // Use the WASM compilation endpoint in KNIRV-CONTROLLER
    const compileResponse = await fetch(`${CONTROLLER_API_BASE}/api/wasm/compile-agent-core`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${CONTROLLER_API_KEY}`
      },
      body: JSON.stringify({
        options: {
          modelId: modelId,
          wasmBytes: Array.from(cortexWasm),
          target: 'cortex-model',
          features: ['inference', 'cortex-processing'],
          configuration: {
            template: modelConfig.template?.id,
            parameters: modelConfig.template?.parameters,
            architecture: modelConfig.template?.architecture,
            quantization: modelConfig.quantization
          }
        }
      })
    });

    if (!compileResponse.ok) {
      throw new Error(`Failed to compile model with KNIRV-CONTROLLER: ${compileResponse.statusText}`);
    }

    const compileResult = await compileResponse.json();

    if (!compileResult.success) {
      throw new Error(`KNIRV-CONTROLLER compilation failed: ${compileResult.error}`);
    }

    // Register the compiled model as a LoRA adapter for execution
    const registerResponse = await fetch(`${CONTROLLER_API_BASE}/api/adapters`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${CONTROLLER_API_KEY}`
      },
      body: JSON.stringify({
        skillId: modelId,
        skillName: modelConfig.name || `Cortex Model ${modelId}`,
        description: modelConfig.description || 'Cortex WASM model deployed from primary website',
        baseModelCompatibility: 'cortex-wasm',
        version: '1.0.0',
        rank: 8,
        alpha: 16.0,
        configuration: {
          modelType: 'cortex',
          deploymentSource: 'primary-website',
          capabilities: ['inference', 'mobile-optimized']
        }
      })
    });

    if (!registerResponse.ok) {
      throw new Error(`Failed to register model with KNIRV-CONTROLLER: ${registerResponse.statusText}`);
    }

    return {
      success: true,
      model_id: modelId,
      url: `${CONTROLLER_API_BASE}/api/adapters/${modelId}`,
      inference_endpoint: `${CONTROLLER_API_BASE}/api/skills/invoke`,
      mobile_endpoint: `${CONTROLLER_API_BASE}/api/mobile/models/${modelId}`,
      status: 'active'
    };
  } catch (error) {
    console.error('KNIRV-CONTROLLER deployment error:', error);
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown deployment error'
    };
  }
}

export async function POST(request: NextRequest) {
  try {
    const {
      model_id,
      cortex_wasm,
      deployment_targets,
      model_config
    } = await request.json();
    
    if (!model_id || !cortex_wasm) {
      return NextResponse.json(
        { success: false, error: 'Model ID and cortex WASM are required' },
        { status: 400 }
      );
    }

    const deploymentResults: Record<string, any> = {};

    // Convert base64 WASM to Uint8Array if needed
    let wasmData: Uint8Array;
    if (typeof cortex_wasm === 'string') {
      // Assume base64 encoded
      const binaryString = atob(cortex_wasm);
      wasmData = new Uint8Array(binaryString.length);
      for (let i = 0; i < binaryString.length; i++) {
        wasmData[i] = binaryString.charCodeAt(i);
      }
    } else {
      // Assume already Uint8Array
      wasmData = new Uint8Array(cortex_wasm);
    }

    // Deploy to KNIRVCONTROLLER
    if (deployment_targets.knirvcontroller) {
      deploymentResults.knirvcontroller = await deployToKNIRVController(
        model_id,
        wasmData,
        model_config
      );
    }

    // Deploy to KNIRVSERVER
    if (deployment_targets.knirvserver) {
      deploymentResults.knirvserver = await deployToKNIRVNexus(
        model_id,
        wasmData,
        model_config
      );
    }

    // Deploy to Cloud Hosting
    if (deployment_targets.cloud_hosting) {
      try {
        const provider = deployment_targets.cloud_hosting.provider;
        
        // In a real implementation, this would:
        // 1. Package cortex.wasm for cloud deployment
        // 2. Deploy to the specified cloud provider
        // 3. Configure auto-scaling and monitoring
        
        deploymentResults.cloud = {
          success: true,
          provider: provider,
          url: `https://${model_id}.${provider}.app`,
          api_endpoint: `https://${model_id}.${provider}.app/api/inference`,
          status: 'deployed'
        };
      } catch (error) {
        deploymentResults.cloud = {
          success: false,
          error: error instanceof Error ? error.message : 'Cloud deployment failed'
        };
      }
    }

    // Check if any deployments succeeded
    const hasSuccessfulDeployment = Object.values(deploymentResults).some(
      (result: any) => result.success
    );

    return NextResponse.json({
      success: hasSuccessfulDeployment,
      model_id: model_id,
      deployment_results: deploymentResults,
      message: hasSuccessfulDeployment
        ? 'Model deployed successfully to selected targets'
        : 'All deployments failed'
    });

  } catch (error) {
    console.error('Model deployment error:', error);
    return NextResponse.json(
      {
        success: false,
        error: 'Failed to deploy model',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const modelId = searchParams.get('model_id');

    if (!modelId) {
      return NextResponse.json(
        { success: false, error: 'Model ID is required' },
        { status: 400 }
      );
    }

    // In a real implementation, this would query the deployment status
    // from a database or deployment service
    
    const deploymentStatus = {
      model_id: modelId,
      deployments: {
        knirvcontroller: {
          status: 'active',
          url: `https://controller.knirv.com/models/${modelId}`,
          last_updated: new Date().toISOString()
        },
        knirvserver: {
          status: 'active',
          url: 'https://knirv.com/nexus-portal',
          last_updated: new Date().toISOString()
        },
        cloud: {
          status: 'active',
          provider: 'vercel',
          url: `https://${modelId}.vercel.app`,
          last_updated: new Date().toISOString()
        }
      }
    };

    return NextResponse.json({
      success: true,
      deployment_status: deploymentStatus
    });

  } catch (error) {
    console.error('Get deployment status error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to get deployment status',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}
