import { NextRequest, NextResponse } from 'next/server';

interface ValidationTask {
  id: string;
  type: "skill_validation" | "llm_update" | "security_audit";
  status: "pending" | "running" | "completed" | "failed";
  priority: "low" | "medium" | "high" | "critical";
  assigned_node: string;
  progress: number;
  created_at: string;
  estimated_completion: string;
  description?: string;
  failure_context?: string;
  result?: any;
}

// Mock data for validation tasks
const mockValidationTasks: ValidationTask[] = [
  {
    id: "task-001",
    type: "skill_validation",
    status: "running",
    priority: "high",
    assigned_node: "dve-001",
    progress: 75,
    created_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    estimated_completion: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
    description: "Validate new SkillNode for NRV-1234: Memory optimization algorithm",
    failure_context: "High memory usage in data processing pipeline"
  },
  {
    id: "task-002",
    type: "llm_update",
    status: "pending",
    priority: "critical",
    assigned_node: "",
    progress: 0,
    created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    estimated_completion: new Date(Date.now() + 4 * 60 * 60 * 1000).toISOString(),
    description: "Validate CodeT5 Base LLM update v2.1.0",
    failure_context: "Model accuracy regression in edge cases"
  },
  {
    id: "task-003",
    type: "security_audit",
    status: "completed",
    priority: "medium",
    assigned_node: "dve-002",
    progress: 100,
    created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    estimated_completion: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    description: "Security audit of DVE node enclave integrity",
    result: {
      passed: true,
      vulnerabilities_found: 0,
      scan_duration: 2456
    }
  }
];

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const status = searchParams.get('status');
    const type = searchParams.get('type');
    const priority = searchParams.get('priority');
    const assigned_node = searchParams.get('assigned_node');
    
    let filteredTasks = [...mockValidationTasks];
    
    if (status) {
      filteredTasks = filteredTasks.filter(task => task.status === status);
    }
    
    if (type) {
      filteredTasks = filteredTasks.filter(task => task.type === type);
    }
    
    if (priority) {
      filteredTasks = filteredTasks.filter(task => task.priority === priority);
    }
    
    if (assigned_node) {
      filteredTasks = filteredTasks.filter(task => task.assigned_node === assigned_node);
    }
    
    return NextResponse.json({
      success: true,
      data: filteredTasks,
      total: filteredTasks.length,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to fetch validation tasks' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { type, priority, description, failure_context } = body;
    
    if (!type || !priority || !description) {
      return NextResponse.json(
        { success: false, error: 'Missing required fields' },
        { status: 400 }
      );
    }
    
    const newTask: ValidationTask = {
      id: `task-${String(mockValidationTasks.length + 1).padStart(3, '0')}`,
      type,
      status: "pending",
      priority,
      assigned_node: "",
      progress: 0,
      created_at: new Date().toISOString(),
      estimated_completion: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
      description,
      failure_context
    };
    
    mockValidationTasks.push(newTask);
    
    return NextResponse.json({
      success: true,
      data: newTask,
      message: 'Validation task created successfully',
      timestamp: new Date().toISOString()
    }, { status: 201 });
  } catch (error) {
    return NextResponse.json(
      { success: false, error: 'Failed to create validation task' },
      { status: 500 }
    );
  }
}