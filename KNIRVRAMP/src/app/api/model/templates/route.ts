import { NextRequest, NextResponse } from 'next/server';
import { cortexModelCompiler } from '@/lib/cortex-compiler/CortexModelCompiler';

export async function GET(request: NextRequest) {
  try {
    const templates = cortexModelCompiler.getAvailableTemplates();
    
    return NextResponse.json({
      success: true,
      templates: templates,
      count: templates.length
    });

  } catch (error) {
    console.error('Get templates error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to get model templates',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const { template_id } = await request.json();
    
    if (!template_id) {
      return NextResponse.json(
        { success: false, error: 'Template ID is required' },
        { status: 400 }
      );
    }

    const template = cortexModelCompiler.getTemplate(template_id);
    
    if (!template) {
      return NextResponse.json(
        { success: false, error: 'Template not found' },
        { status: 404 }
      );
    }

    return NextResponse.json({
      success: true,
      template: template
    });

  } catch (error) {
    console.error('Get template error:', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to get template',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}
