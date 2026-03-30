import { NextRequest, NextResponse } from 'next/server';
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

export async function POST(request: NextRequest) {
  try {
    const compilationRequest: ModelCompilationRequest = await request.json();
    
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
      (progress: TrainingProgress) => {
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

    return NextResponse.json({
      success: true,
      compilation_id: compilationId,
      message: 'Model compilation started'
    });

  } catch (error) {
    console.error('Model compilation error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to start model compilation',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const compilationId = searchParams.get('id');

    if (!compilationId) {
      return NextResponse.json(
        { success: false, error: 'Compilation ID is required' },
        { status: 400 }
      );
    }

    const session = activeCompilations.get(compilationId);
    
    if (!session) {
      return NextResponse.json(
        { success: false, error: 'Compilation session not found' },
        { status: 404 }
      );
    }

    return NextResponse.json({
      success: true,
      compilation_id: compilationId,
      status: session.status,
      progress: session.progress,
      result: session.result
    });

  } catch (error) {
    console.error('Get compilation status error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to get compilation status',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}

export async function DELETE(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const compilationId = searchParams.get('id');

    if (!compilationId) {
      return NextResponse.json(
        { success: false, error: 'Compilation ID is required' },
        { status: 400 }
      );
    }

    // Cancel compilation if running
    if (cortexModelCompiler.isCompiling()) {
      cortexModelCompiler.cancelCompilation();
    }

    // Remove session
    activeCompilations.delete(compilationId);

    return NextResponse.json({
      success: true,
      message: 'Compilation cancelled'
    });

  } catch (error) {
    console.error('Cancel compilation error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to cancel compilation',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}
