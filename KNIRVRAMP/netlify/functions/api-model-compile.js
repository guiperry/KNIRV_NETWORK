// Auto-generated Netlify function from Next.js API route
// Original route: /api/model/compile
// Generated: 2025-09-24T01:08:10.221Z

// NextResponse/NextRequest converted to native Netlify response format

import { 
  cortexModelCompiler, 
  ModelCompilationRequest, 
  TrainingProgress 
} from '@/lib/cortex-compiler/CortexModelCompiler';

// Store active compilation sessions
const activeCompilations = new Map<string, {
  progress: TrainingProgress | null;
  result: any | null;
  status: 'running' | 'completed' | 'failed';
}>();

// CORS headers for all responses
const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization',
  'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS'
};

async function POST(event, context) {
  // Extract route parameters if this is a dynamic route
  if (event.pathParameters) {
    event.params = event.pathParameters;
  }

  // Parse request body if present
  let requestBody = {};
  if (event.body) {
    try {
      requestBody = JSON.parse(event.body);
    } catch (e) {
      requestBody = event.body;
    }
  }
  

  try {
    const compilationRequest: ModelCompilationRequest = requestBody;
    
    // Generate compilation ID
    const compilationId = `comp_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    // Initialize compilation session
    activeCompilations.set(compilationId, {
      progress: null,
      result: null,
      status: 'running'
    });

    // Start compilation in background
    cortexModelCompiler.compileModel(
      compilationRequest,
      (progress) => {
        const session = activeCompilations.get(compilationId);
        if (session) {
          session.progress = progress;
          activeCompilations.set(compilationId, session);
        }
      }
    ).then(result => {
      const session = activeCompilations.get(compilationId);
      if (session) {
        session.result = result;
        session.status = result.success ? 'completed' : 'failed';
        activeCompilations.set(compilationId, session);
      }
    }).catch(error => {
      const session = activeCompilations.get(compilationId);
      if (session) {
        session.result = { success: false, message: error.message };
        session.status = 'failed';
        activeCompilations.set(compilationId, session);
      }
    });

    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: true,
      compilation_id,
      message: 'Model compilation started'})
    };

  } catch (error) {
    console.error('Model compilation error:', error);
    return {
      statusCode: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, 
        error: 'Failed to start model compilation',
        details: error instanceof Error ? error.message : 'Unknown error'})
    };
  }
}

async function GET(event, context) {
  // Extract route parameters if this is a dynamic route
  if (event.pathParameters) {
    event.params = event.pathParameters;
  }

  // Parse request body if present
  let requestBody = {};
  if (event.body) {
    try {
      requestBody = JSON.parse(event.body);
    } catch (e) {
      requestBody = event.body;
    }
  }
  

  try {
    const { searchParams } = new URL(`https://${event.headers.host}${event.path}`);
    const compilationId = searchParams.get('id');

    if (!compilationId) {
      return {
      statusCode: 400,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, error: 'Compilation ID is required'})
    };
    }

    const session = activeCompilations.get(compilationId);
    
    if (!session) {
      return {
      statusCode: 404,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, error: 'Compilation session not found'})
    };
    }

    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: true,
      compilation_id,
      status: session.status,
      progress: session.progress,
      result: session.result})
    };

  } catch (error) {
    console.error('Get compilation status error:', error);
    return {
      statusCode: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, 
        error: 'Failed to get compilation status',
        details: error instanceof Error ? error.message : 'Unknown error'})
    };
  }
}

async function DELETE(event, context) {
  // Extract route parameters if this is a dynamic route
  if (event.pathParameters) {
    event.params = event.pathParameters;
  }

  // Parse request body if present
  let requestBody = {};
  if (event.body) {
    try {
      requestBody = JSON.parse(event.body);
    } catch (e) {
      requestBody = event.body;
    }
  }
  

  try {
    const { searchParams } = new URL(`https://${event.headers.host}${event.path}`);
    const compilationId = searchParams.get('id');

    if (!compilationId) {
      return {
      statusCode: 400,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, error: 'Compilation ID is required'})
    };
    }

    // Cancel compilation if running
    if (cortexModelCompiler.isCompiling()) {
      cortexModelCompiler.cancelCompilation();
    }

    // Remove session
    activeCompilations.delete(compilationId);

    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: true,
      message: 'Compilation cancelled'})
    };

  } catch (error) {
    console.error('Cancel compilation error:', error);
    return {
      statusCode: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({success: false, 
        error: 'Failed to cancel compilation',
        details: error instanceof Error ? error.message : 'Unknown error'})
    };
  }
}

// Main Netlify function handler
exports.handler = async (event, context) => {
  // Handle preflight requests
  if (event.httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers: corsHeaders,
      body: ''
    };
  }

  try {
    const method = event.httpMethod;
    
    // Add route parameters to event if dynamic route
    
    
    // Route to appropriate handler
    switch (method) {
      
      case 'POST':
        if (typeof POST === 'function') {
          const result = await POST(event);
          return {
            ...result,
            headers: { ...corsHeaders, ...(result.headers || {}) }
          };
        }
        break;
      case 'GET':
        if (typeof GET === 'function') {
          const result = await GET(event);
          return {
            ...result,
            headers: { ...corsHeaders, ...(result.headers || {}) }
          };
        }
        break;
      case 'DELETE':
        if (typeof DELETE === 'function') {
          const result = await DELETE(event);
          return {
            ...result,
            headers: { ...corsHeaders, ...(result.headers || {}) }
          };
        }
        break;
      
      default:
        return {
          statusCode: 405,
          headers: corsHeaders,
          body: JSON.stringify({ error: 'Method not allowed' })
        };
    }
    
    return {
      statusCode: 404,
      headers: corsHeaders,
      body: JSON.stringify({ error: 'Handler not found' })
    };
    
  } catch (error) {
    console.error('Function error:', error);
    return {
      statusCode: 500,
      headers: corsHeaders,
      body: JSON.stringify({ error: 'Internal server error' })
    };
  }
};

// Export individual handlers for testing
exports.post = POST;
exports.get = GET;
exports.delete = DELETE;
