// Onboarding steps covering all major features from Features.md
export const onboardingSteps = [
  {
    stepNumber: 1,
    title: "Welcome to Agentic Engine!",
    description: "Welcome to your AI-powered agent management platform! This tour will guide you through all the major features. You can switch between auto-play and manual mode at any time.",
    target: ".main-header", // Main header/logo area - simplified to match actual class
    features: [
      "Complete AI agent lifecycle management",
      "Multi-provider AI model integration",
      "Advanced workflow orchestration",
      "Secure execution environments"
    ],
    action: null,
    autoDelay: 4000
  },
  {
    stepNumber: 2,
    title: "Dashboard Overview",
    description: "Your central command center displays system status, active agents, recent activities, and key performance metrics at a glance.",
    target: "button[data-page='dashboard']", // Dashboard navigation button - more specific selector
    features: [
      "Real-time system monitoring",
      "Agent performance analytics",
      "Resource usage tracking",
      "Quick action shortcuts"
    ],
    action: { type: 'navigate', path: '/dashboard' },
    autoDelay: 5000
  },
  {
    stepNumber: 3,
    title: "Agent Management Hub",
    description: "Create, configure, and manage your AI agents. Each agent can have unique capabilities, instructions, and AI model configurations.",
    target: "button[data-page='agents']", // Agents navigation button - more specific selector
    features: [
      "NFT-based agent identities",
      "Template-based agent creation",
      "Version control and rollback",
      "Performance monitoring"
    ],
    action: { type: 'navigate', path: '/agents' },
    autoDelay: 5000
  },
  {
    stepNumber: 4,
    title: "Agent Builder",
    description: "Build custom agents from templates with specific configurations. The builder compiles your agent logic into secure, loadable plugins.",
    target: "[data-testid='create-agent-button'], .create-agent-button, button[class*='create'], button[aria-label*='Create Agent']",
    features: [
      "Template selection",
      "Custom configuration",
      "Plugin compilation",
      "Build status monitoring"
    ],
    action: null,
    autoDelay: 4500
  },
  {
    stepNumber: 5,
    title: "Sub-Agent Orchestration",
    description: "Agents can spawn sub-agents for complex tasks using 8 different orchestration patterns, from simple tools to hierarchical parallel processing.",
    target: "[data-agent-card], .agent-card",
    features: [
      "8 orchestration patterns",
      "TEE isolation",
      "Resource limits",
      "Terminal sessions"
    ],
    action: null,
    autoDelay: 5000
  },
  {
    stepNumber: 6,
    title: "Workflow Orchestration",
    description: "Design and execute complex workflows that coordinate multiple agents, handle sequential and parallel processing, and manage conditional logic.",
    target: "[data-page='workflows']",
    features: [
      "Visual workflow designer",
      "Sequential & parallel execution",
      "Conditional branching",
      "Result aggregation"
    ],
    action: { type: 'navigate', path: '/workflows' },
    autoDelay: 5000
  },
  {
    stepNumber: 7,
    title: "Target Systems",
    description: "Define objectives and targets for your agents to pursue. Configure parameters, assign agents, and track progress toward goals.",
    target: "[data-page='targets']",
    features: [
      "Objective definition",
      "Target configuration",
      "Progress tracking",
      "Result evaluation"
    ],
    action: { type: 'navigate', path: '/targets' },
    autoDelay: 4500
  },
  {
    stepNumber: 8,
    title: "MCP Server Integration",
    description: "Model Context Protocol servers extend your agents' capabilities. Install, configure, and manage MCP servers for enhanced functionality.",
    target: "[data-page='capabilities']",
    features: [
      "Automated MCP installation",
      "Capability transformation",
      "Server lifecycle management",
      "Real-time monitoring"
    ],
    action: { type: 'navigate', path: '/capabilities' },
    autoDelay: 5000
  },
  {
    stepNumber: 9,
    title: "AI Model Configuration",
    description: "Configure multiple AI providers including Cerebras, Gemini, and DeepSeek. Set up intelligent fallbacks and mixture-of-agents configurations.",
    target: "[data-page='settings']",
    features: [
      "Multi-provider support",
      "API key management",
      "Intelligent fallback",
      "Usage monitoring"
    ],
    action: { type: 'navigate', path: '/settings' },
    autoDelay: 5000
  },
  {
    stepNumber: 10,
    title: "Inference API Settings",
    description: "Configure your AI model settings, API keys, and inference parameters. Set up primary and fallback providers for reliable operation.",
    target: "[data-tab='inference']",
    features: [
      "Model selection",
      "API configuration",
      "Fallback settings",
      "Performance tuning"
    ],
    action: { type: 'click', target: "[data-tab='inference']" },
    autoDelay: 4000
  },
  {
    stepNumber: 11,
    title: "Security & Authentication",
    description: "Manage user accounts, permissions, and security settings. The platform includes JWT authentication, role-based access control, and TEE isolation.",
    target: "[data-tab='security']",
    features: [
      "JWT authentication",
      "Role-based access control",
      "Permission management",
      "TEE security"
    ],
    action: { type: 'click', target: "[data-tab='security']" },
    autoDelay: 4500
  },
  {
    stepNumber: 12,
    title: "Error Analysis Engine",
    description: "AI-powered error analysis automatically diagnoses issues, provides solutions, and can even implement fixes. Access it through the notification bell.",
    target: ".error-notification-bell, [title*='system errors']",
    features: [
      "LLM-powered diagnosis",
      "Smart categorization",
      "Solution recommendations",
      "Self-healing capabilities"
    ],
    action: null,
    autoDelay: 5000
  },
  {
    stepNumber: 13,
    title: "Debug & Testing Tools",
    description: "Access powerful debugging and testing tools to troubleshoot issues, test functionality, and manage system state.",
    target: "[data-tab='debug']",
    features: [
      "Error engine testing",
      "System diagnostics",
      "Performance monitoring",
      "Demo data management"
    ],
    action: { type: 'click', target: "[data-tab='debug']" },
    autoDelay: 4000
  },
  {
    stepNumber: 14,
    title: "Demo Data Toggle - Your Presentation Tool",
    description: "This powerful toggle lets you instantly hide or show all demo data for clean presentations. When disabled, all sample data is safely backed up and can be restored with one click. Perfect for demos, screenshots, or client presentations!",
    target: ".demo-data-toggle, [data-feature='demo-data-toggle']",
    features: [
      "Instant demo data hiding",
      "Safe backup & restore",
      "Perfect for presentations",
      "One-click toggle operation",
      "No data loss risk"
    ],
    action: null,
    autoDelay: 6000
  }
];

// Navigation helper to determine current page
export const getCurrentPage = () => {
  const path = window.location.pathname;
  if (path === '/' || path === '/dashboard') return 'dashboard';
  if (path.includes('/agents')) return 'agents';
  if (path.includes('/workflows')) return 'workflows';
  if (path.includes('/targets')) return 'targets';
  if (path.includes('/capabilities')) return 'capabilities';
  if (path.includes('/settings')) return 'settings';
  return 'dashboard';
};

// Action executor for onboarding steps
export const executeStepAction = (action, navigate) => {
  if (!action) return;

  switch (action.type) {
    case 'navigate':
      if (navigate) {
        navigate(action.path);
      }
      break;
    case 'click':
      setTimeout(() => {
        const element = document.querySelector(action.target);
        if (element) {
          element.click();
        }
      }, 500);
      break;
    default:
      break;
  }
};
