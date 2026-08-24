import React, { useState, useEffect } from 'react';
import {
  Search,
  Filter,
  Grid,
  List,
  Plus,
  Bot,
  Zap,
  Eye,
  MoreVertical,
  Play,
  Pause,
  Square,
  Settings,
  Target,
  Activity,
  Clock,
  Download,
  CheckCircle,
  AlertCircle,
  X,
  Terminal,
  MessageSquare,
  ChevronDown
} from 'lucide-react';
import AgentCreationModal from './modals/AgentCreationModal';
import AgentDeploymentModal from './modals/AgentDeploymentModal';
import AgentConfigModal from './modals/AgentConfigModal';
import AdvancedFilterModal from './modals/AdvancedFilterModal';
import AgentDetailModal from './modals/AgentDetailModal';
import PluginImportModal from './modals/PluginImportModal';
import AgentTerminal from './AgentTerminal';
import AgentChatModal from './modals/AgentChatModal';
import { fetchAgents, fetchAllAgents, updateAgent, deleteAgent, stopAgent } from '../utils/api';
import { useDefaultAgentImage } from '../hooks/useAssetPath';
import { subscribeToAgentStatus } from '../utils/websocket';
import { KeyboardNavigation, AriaUtils } from '../utils/accessibility';
import AgentImage from './common/AgentImage';

export const AgentManager = () => {
  const defaultAgentImage = useDefaultAgentImage();
  const [viewMode, setViewMode] = useState('grid');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [isCreationModalOpen, setIsCreationModalOpen] = useState(false);
  const [isDeploymentModalOpen, setIsDeploymentModalOpen] = useState(false);
  const [isConfigModalOpen, setIsConfigModalOpen] = useState(false);
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [isPluginImportModalOpen, setIsPluginImportModalOpen] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [activeFilters, setActiveFilters] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [notification, setNotification] = useState(null);
  const [agentTerminals, setAgentTerminals] = useState({}); // Track terminal state for each agent
  const [agentSessions, setAgentSessions] = useState({}); // Track session IDs for each agent
  const [isChatModalOpen, setIsChatModalOpen] = useState(false);
  const [selectedChatAgent, setSelectedChatAgent] = useState(null);
  // Terminal dropdown removed in favor of direct terminal access
  const [focusedAgentIndex, setFocusedAgentIndex] = useState(-1);
  const [keyboardNavigationActive, setKeyboardNavigationActive] = useState(false);

  const categories = [
    { id: 'all', name: 'All Agents' },
    { id: 'active', name: 'Active' },
    { id: 'idle', name: 'Idle' },
    { id: 'deployed', name: 'Deployed' },
    { id: 'maintenance', name: 'Maintenance' },
  ];

  const [agents, setAgents] = useState([]);

  // Fetch agents from the API
  const loadAgents = async () => {
      try {
        setIsLoading(true);
        // Fetch all agents (database + discovered)
        const fetchedAgents = await fetchAllAgents();
        
        // Transform the API response to match our UI format
        const formattedAgents = fetchedAgents.map(agent => {
          // Handle different agent sources (database vs discovered)
          let configData = {};

          if (agent.source === 'database') {
            // Database agents have config field that needs parsing
            try {
              if (agent.config) {
                configData = JSON.parse(agent.config);
              }
            } catch (e) {
              console.warn('Failed to parse agent config:', e);
            }

            return {
              id: agent.id,
              name: agent.name,
              collection: configData.collection || agent.collection || '',
              image: configData.image_url || agent.image_url || defaultAgentImage,
              status: configData.status || agent.status || 'idle',
              currentTarget: null,
              capability: null,
              deployedSince: null,
              totalInferences: 0,
              successRate: 0,
              lastActivity: agent.updated_at ? new Date(agent.updated_at).toLocaleString() : 'Never',
              capabilities: configData.capabilities || agent.capabilities || [],
              targetTypes: configData.target_types || agent.target_types || [],
              createdAt: agent.created_at || new Date().toISOString(),
              owner: 'User',
              type: agent.type || 'standard',
              config: agent.config || '{}',
              source: 'database'
            };
          } else {
            // Discovered agents have direct fields
            return {
              id: agent.id,
              name: agent.name,
              collection: agent.collection || 'Discovered Agents',
              image: agent.image_url || defaultAgentImage,
              status: agent.status || 'available',
              currentTarget: null,
              capability: null,
              deployedSince: null,
              totalInferences: 0,
              successRate: 0,
              lastActivity: 'Available',
              capabilities: agent.capabilities || [],
              targetTypes: agent.target_types || [],
              createdAt: agent.created_at || new Date().toISOString(),
              owner: 'System',
              type: agent.type || 'discovered',
              config: '{}',
              source: 'discovered',
              file_type: agent.file_type,
              version: agent.version
            };
          }
        });
        
        setAgents(formattedAgents);
        setError(null);
      } catch (err) {
        console.error('Failed to fetch agents:', err);
        setError('Failed to load agents. Please try again later.');
        
        // Fallback to sample data for demo purposes
        setAgents([
          {
            id: '1',
            name: 'CyberPunk Agent #7804',
            collection: 'CyberPunk Collective',
            image: defaultAgentImage,
            status: 'idle',
            currentTarget: null,
            capability: null,
            deployedSince: null,
            totalInferences: 0,
            successRate: 0,
            lastActivity: 'Never',
            capabilities: ['Web Analysis', 'Data Extraction', 'Content Monitoring'],
            targetTypes: ['Browser', 'Web APIs', 'Social Media'],
            createdAt: new Date().toISOString(),
            owner: 'User'
          }
        ]);
      } finally {
        setIsLoading(false);
      }
    };

  useEffect(() => {
    loadAgents();

    // Subscribe to agent status updates
    const handleAgentStatusUpdate = (update) => {
      setAgents(prevAgents =>
        prevAgents.map(agent =>
          agent.id === update.agentId
            ? {
                ...agent,
                status: update.status,
                currentTarget: update.currentTarget || agent.currentTarget,
                capability: update.capability || agent.capability,
                lastActivity: update.lastActivity
              }
            : agent
        )
      );
    };

    const unsubscribe = subscribeToAgentStatus(handleAgentStatusUpdate);

    // The application owns the shared WebSocket lifecycle. This page only
    // manages its own subscription, avoiding competing connections/retries.
    return () => {
      unsubscribe();
    };
  }, []);

  const getStatusColor = (status) => {
    switch (status) {
      case 'active':
        return 'from-green-400 to-emerald-500';
      case 'deployed':
        return 'from-blue-400 to-indigo-500';
      case 'monitoring':
        return 'from-purple-400 to-pink-500';
      case 'idle':
        return 'from-gray-400 to-gray-500';
      case 'maintenance':
        return 'from-yellow-400 to-orange-500';
      default:
        return 'from-gray-400 to-gray-500';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'active':
        return <Play className="w-4 h-4" />;
      case 'deployed':
        return <Target className="w-4 h-4" />;
      case 'monitoring':
        return <Activity className="w-4 h-4" />;
      case 'idle':
        return <Pause className="w-4 h-4" />;
      case 'maintenance':
        return <Settings className="w-4 h-4" />;
      default:
        return <Square className="w-4 h-4" />;
    }
  };

  // Filter options for the advanced filter modal
  const filterOptions = {
    fields: [
      { id: 'name', name: 'Name' },
      { id: 'collection', name: 'Collection' },
      { id: 'status', name: 'Status' },
      { id: 'capability', name: 'Capability' },
      { id: 'currentTarget', name: 'Current Target' },
      { id: 'successRate', name: 'Success Rate' },
      { id: 'totalInferences', name: 'Total Inferences' },
      { id: 'createdAt', name: 'Created Date' },
      { id: 'owner', name: 'Owner' }
    ],
    operators: [
      { id: 'equals', name: 'Equals' },
      { id: 'not_equals', name: 'Not Equals' },
      { id: 'contains', name: 'Contains' },
      { id: 'not_contains', name: 'Not Contains' },
      { id: 'greater_than', name: 'Greater Than' },
      { id: 'less_than', name: 'Less Than' },
      { id: 'starts_with', name: 'Starts With' },
      { id: 'ends_with', name: 'Ends With' }
    ]
  };

  // Apply advanced filters
  const applyAdvancedFilters = (filters) => {
    setActiveFilters(filters);
  };

  // Check if an agent matches the advanced filters
  const matchesAdvancedFilters = (agent) => {
    if (activeFilters.length === 0) return true;
    
    return activeFilters.every(filter => {
      const { field, operator, value } = filter;
      const agentValue = agent[field];
      
      // Handle null/undefined values
      if (agentValue === null || agentValue === undefined) {
        return operator === 'not_equals' || operator === 'not_contains';
      }
      
      // Convert to string for comparison
      const agentValueStr = String(agentValue).toLowerCase();
      const filterValueStr = String(value).toLowerCase();
      
      switch (operator) {
        case 'equals':
          return agentValueStr === filterValueStr;
        case 'not_equals':
          return agentValueStr !== filterValueStr;
        case 'contains':
          return agentValueStr.includes(filterValueStr);
        case 'not_contains':
          return !agentValueStr.includes(filterValueStr);
        case 'greater_than':
          return parseFloat(agentValueStr) > parseFloat(filterValueStr);
        case 'less_than':
          return parseFloat(agentValueStr) < parseFloat(filterValueStr);
        case 'starts_with':
          return agentValueStr.startsWith(filterValueStr);
        case 'ends_with':
          return agentValueStr.endsWith(filterValueStr);
        default:
          return true;
      }
    });
  };

  const filteredAgents = agents.filter(agent => {
    // Add defensive checks to prevent undefined errors
    if (!agent || !agent.name) {
      return false;
    }

    const matchesSearch = agent.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         (agent.collection || '').toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || agent.status === selectedCategory;
    const matchesAdvanced = matchesAdvancedFilters(agent);
    return matchesSearch && matchesCategory && matchesAdvanced;
  });

  const handleAgentCreated = (newAgent) => {
    // Parse the config to extract properties
    let configData = {};
    try {
      if (newAgent.config) {
        configData = JSON.parse(newAgent.config);
      }
    } catch (e) {
      console.warn('Failed to parse new agent config:', e);
    }

    // Add the new agent to the UI state
    setAgents(prevAgents => [...prevAgents, {
      id: newAgent.id,
      name: newAgent.name,
      collection: configData.collection || '',
      image: configData.image_url || defaultAgentImage,
      status: configData.status || 'idle',
      currentTarget: null,
      capability: null,
      deployedSince: null,
      totalInferences: 0,
      successRate: 0,
      lastActivity: 'Just now',
      capabilities: configData.capabilities || [],
      targetTypes: configData.target_types || [],
      createdAt: newAgent.created_at || new Date().toISOString(),
      owner: 'User',
      type: newAgent.type || 'standard',
      config: newAgent.config || '{}'
    }]);
    setIsCreationModalOpen(false);
  };

  const handlePluginImported = () => {
    // Refresh the agents list after plugin import
    loadAgents();
    setIsPluginImportModalOpen(false);

    // Show success notification
    setNotification({
      type: 'success',
      message: 'Plugin imported successfully! The new agent should appear in your list.'
    });

    // Auto-hide notification after 5 seconds
    setTimeout(() => {
      setNotification(null);
    }, 5000);
  };

  const handleAgentDeployed = async (updatedAgent) => {
    try {
      // Parse the current config if it exists
      let configObj = {};
      try {
        const agent = agents.find(a => a.id === updatedAgent.id);
        if (agent && agent.config) {
          configObj = JSON.parse(agent.config);
        }
      } catch (e) {
        console.warn('Failed to parse current agent config:', e);
      }
      
      // Update the config with deployment information
      configObj = {
        ...configObj,
        collection: updatedAgent.collection,
        image_url: updatedAgent.image,
        capabilities: updatedAgent.capabilities,
        target_types: updatedAgent.targetTypes,
        status: 'active',
        current_target: updatedAgent.currentTarget,
        capability: updatedAgent.capability,
        deployed_at: new Date().toISOString()
      };
      
      // Update the agent in the backend
      const apiAgent = {
        name: updatedAgent.name,
        type: updatedAgent.type || 'standard',
        config: JSON.stringify(configObj)
      };
      
      await updateAgent(updatedAgent.id, apiAgent);
      
      // Update the UI with the deployed agent
      const deployedAgent = {
        ...updatedAgent,
        status: 'active',
        deployedSince: 'Just now',
        lastActivity: 'Just now'
      };
      
      setAgents(prevAgents => 
        prevAgents.map(agent => 
          agent.id === updatedAgent.id ? deployedAgent : agent
        )
      );
    } catch (error) {
      console.error('Error deploying agent:', error);
      // Handle error (e.g., show notification)
    }
    
    setIsDeploymentModalOpen(false);
    setSelectedAgent(null);
  };

  const handleAgentUpdated = async (updatedAgent) => {
    try {
      // Parse the current config if it exists
      let configObj = {};
      try {
        const agent = agents.find(a => a.id === updatedAgent.id);
        if (agent && agent.config) {
          configObj = JSON.parse(agent.config);
        }
      } catch (e) {
        console.warn('Failed to parse current agent config:', e);
      }
      
      // Update the config with new information
      configObj = {
        ...configObj,
        collection: updatedAgent.collection,
        image_url: updatedAgent.image,
        capabilities: updatedAgent.capabilities,
        target_types: updatedAgent.targetTypes,
        status: updatedAgent.status || 'idle'
      };
      
      // Update the agent in the backend
      const apiAgent = {
        name: updatedAgent.name,
        type: updatedAgent.type || 'standard',
        config: JSON.stringify(configObj)
      };
      
      await updateAgent(updatedAgent.id, apiAgent);
      
      // Update the UI
      const updatedUIAgent = {
        ...updatedAgent,
        lastActivity: 'Just now'
      };
      
      setAgents(prevAgents => 
        prevAgents.map(agent => 
          agent.id === updatedAgent.id ? updatedUIAgent : agent
        )
      );
    } catch (error) {
      console.error('Error updating agent:', error);

      // If agent not found (404), refresh the agent list
      if (error.message && error.message.includes('Not Found')) {
        console.log('Agent not found, refreshing agent list...');
        // Reload agents from the API
        try {
          const fetchedAgents = await fetchAllAgents();
          const formattedAgents = fetchedAgents.map(agent => {
            // Handle different agent sources (database vs discovered)
            let configData = {};

            if (agent.source === 'database') {
              // Database agents have config field that needs parsing
              try {
                if (agent.config) {
                  configData = JSON.parse(agent.config);
                }
              } catch (e) {
                console.warn('Failed to parse agent config:', e);
              }

              return {
                id: agent.id,
                name: agent.name,
                collection: configData.collection || agent.collection || '',
                image: configData.image_url || agent.image_url || defaultAgentImage,
                status: configData.status || agent.status || 'idle',
                currentTarget: null,
                capability: null,
                deployedSince: null,
                totalInferences: 0,
                successRate: 0,
                lastActivity: agent.updated_at ? new Date(agent.updated_at).toLocaleString() : 'Never',
                capabilities: configData.capabilities || agent.capabilities || [],
                targetTypes: configData.target_types || agent.target_types || [],
                createdAt: agent.created_at || new Date().toISOString(),
                owner: 'User',
                type: agent.type || 'standard',
                config: agent.config || '{}',
                source: 'database'
              };
            } else {
              // Discovered agents have direct fields
              return {
                id: agent.id,
                name: agent.name,
                collection: agent.collection || 'Discovered Agents',
                image: agent.image_url || defaultAgentImage,
                status: agent.status || 'available',
                currentTarget: null,
                capability: null,
                deployedSince: null,
                totalInferences: 0,
                successRate: 0,
                lastActivity: 'Available',
                capabilities: agent.capabilities || [],
                targetTypes: agent.target_types || [],
                createdAt: agent.created_at || new Date().toISOString(),
                owner: 'System',
                type: agent.type || 'discovered',
                config: '{}',
                source: 'discovered',
                file_type: agent.file_type,
                version: agent.version
              };
            }
          });
          setAgents(formattedAgents);
          setError('Agent list refreshed. The agent you were trying to update no longer exists.');
        } catch (refreshError) {
          console.error('Failed to refresh agents:', refreshError);
          setError('Failed to update agent and refresh list. Please refresh the page.');
        }
      } else {
        setError('Failed to update agent. Please try again.');
      }
    }
    
    setIsConfigModalOpen(false);
    setSelectedAgent(null);
  };

  const handleDeployClick = (agent) => {
    setSelectedAgent(agent);
    setIsDeploymentModalOpen(true);
  };

  const handleConfigClick = (agent) => {
    setSelectedAgent(agent);
    setIsConfigModalOpen(true);
  };

  const handleViewDetails = (agent) => {
    console.log('handleViewDetails called for agent:', agent.name);
    setSelectedAgent(agent);
    setIsDetailModalOpen(true);
  };

  const handleDeleteAgent = async (agent) => {
    try {
      // Remove the agent from the local state immediately for better UX
      setAgents(prevAgents => prevAgents.filter(a => a.id !== agent.id));

      // Show success message
      setError(`Agent "${agent.name}" has been deleted successfully.`);

      // Clear the success message after a few seconds
      setTimeout(() => {
        setError(null);
      }, 3000);

    } catch (error) {
      console.error('Error deleting agent:', error);
      setError(`Failed to delete agent: ${error.message}`);

      // Refresh the agents list to restore the agent if deletion failed
      try {
        const response = await fetchAllAgents();
        const formattedAgents = response.map(agent => ({
          ...agent,
          imageURL: agent.image_url || getDefaultAgentImage(),
          capabilities: agent.capabilities || [],
          targetTypes: agent.target_types || []
        }));
        setAgents(formattedAgents);
      } catch (refreshError) {
        console.error('Failed to refresh agents after delete error:', refreshError);
      }
    }
  };

  const handleStopAgent = async (agent) => {
    try {
      // Call the stop agent API
      await stopAgent(agent.id);

      // The stop endpoint is authoritative and persists the idle state. Do
      // not follow it with the generic update endpoint: legacy agents returned
      // by /agents/all live in the compatibility repository, not unified
      // storage, and that second call used to turn a successful stop into 404.
      const updatedAgent = {
        ...agent,
        status: 'idle',
        currentTarget: null,
        capability: null,
        deployedSince: null,
        lastActivity: 'Just now'
      };
      
      // Update the UI
      setAgents(prevAgents => 
        prevAgents.map(a => 
          a.id === agent.id ? updatedAgent : a
        )
      );
    } catch (error) {
      console.error('Error stopping agent:', error);
      // Handle error (e.g., show notification)
    }
  };

  // Terminal management functions
  const handleTerminalToggle = async (agent) => {
    console.log('handleTerminalToggle called for agent:', agent.name);

    const terminalState = agentTerminals[agent.id];

    if (terminalState?.isVisible) {
      // Hide terminal
      setAgentTerminals(prev => ({
        ...prev,
        [agent.id]: { ...terminalState, isVisible: false, isMinimized: true }
      }));
    } else {
      // Show terminal - first activate agent if needed
      if (agent.status === 'idle') {
        await activateAgentForTerminal(agent);
      }

      // Show terminal
      setAgentTerminals(prev => ({
        ...prev,
        [agent.id]: { isVisible: true, isMinimized: false }
      }));
    }
  };

  const activateAgentForTerminal = async (agent) => {
    try {
      // Generate a session ID for this agent
      const sessionId = `session-${agent.id}-${Date.now()}`;

      // Store the session ID
      setAgentSessions(prev => ({
        ...prev,
        [agent.id]: sessionId
      }));

      // First, discover available agents to get the proper agent ID and version
      const discoverResponse = await fetch('http://localhost:8081/api/v1/adk/agents', {
        method: 'GET',
        headers: {
          'Authorization': 'Bearer test-api-key'
        }
      });

      if (!discoverResponse.ok) {
        throw new Error(`Failed to discover agents: ${discoverResponse.statusText}`);
      }

      const discoverData = await discoverResponse.json();
      console.log('Available agents:', discoverData.agents);

      // Extract agent ID from the agent object
      // The agent ID is stored in the agent object or can be derived from the name
      const agentId = agent.id || agent.name.toLowerCase().replace(/[^a-z0-9]/g, '');
      
      // Use a consistent version format (1.0 instead of 1.0.0)
      const version = "1.0";

      console.log(`Activating agent: ${agentId} with version: ${version} for session: ${sessionId}`);

      // Activate the agent with the correct ID and version
      const activateResponse = await fetch('http://localhost:8081/api/v1/adk/agents/activate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-api-key'
        },
        body: JSON.stringify({
          agentId: agentId,
          version: version,
          sessionId: sessionId,
          config: {}
        })
      });

      if (!activateResponse.ok) {
        throw new Error(`Failed to activate agent: ${activateResponse.statusText}`);
      }

      const activateData = await activateResponse.json();
      console.log('Agent activated:', activateData);

      // Update the UI to show the agent as active
      const updatedAgent = {
        ...agent,
        status: 'active',
        lastActivity: 'Just now'
      };

      setAgents(prevAgents =>
        prevAgents.map(a =>
          a.id === agent.id ? updatedAgent : a
        )
      );
    } catch (error) {
      console.error('Error activating agent for terminal:', error);
      // Show error to user
      setNotification({
        type: 'error',
        message: `Failed to activate agent: ${error.message}`
      });
    }
  };

  const handleTerminalMinimize = (agentId) => {
    setAgentTerminals(prev => ({
      ...prev,
      [agentId]: { ...prev[agentId], isVisible: false, isMinimized: true }
    }));
  };

  const handleTerminalMaximize = (agentId) => {
    setAgentTerminals(prev => ({
      ...prev,
      [agentId]: { ...prev[agentId], isVisible: true, isMinimized: false }
    }));
  };

  // Chat modal handlers
  const handleOpenChat = (agent) => {
    setSelectedChatAgent(agent);
    setIsChatModalOpen(true);
    // Terminal dropdown removed
  };

  const handleCloseChat = () => {
    setIsChatModalOpen(false);
    setSelectedChatAgent(null);
  };

  // Terminal dropdown functionality removed in favor of direct terminal access

  const handleTerminalClose = (agentId) => {
    setAgentTerminals(prev => {
      const newState = { ...prev };
      delete newState[agentId];
      return newState;
    });

    setAgentSessions(prev => {
      const newState = { ...prev };
      delete newState[agentId];
      return newState;
    });
  };

  // Format the active filters for display
  const getActiveFiltersDisplay = () => {
    if (activeFilters.length === 0) return null;
    
    return (
      <div className="flex items-center space-x-2 text-sm text-slate-400">
        <span>Active filters:</span>
        <div className="flex flex-wrap gap-2">
          {activeFilters.map((filter, index) => {
            const fieldName = filterOptions.fields.find(f => f.id === filter.field)?.name || filter.field;
            const operatorName = filterOptions.operators.find(o => o.id === filter.operator)?.name || filter.operator;
            
            return (
              <span 
                key={filter.id} 
                className="px-2 py-1 bg-purple-500/20 text-purple-300 text-xs rounded-full border border-purple-500/30"
              >
                {fieldName} {operatorName} {filter.value}
              </span>
            );
          })}
          <button 
            onClick={() => setActiveFilters([])}
            className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full hover:bg-slate-700 transition-colors duration-200"
          >
            Clear
          </button>
        </div>
      </div>
    );
  };

  // Keyboard navigation handlers
  const handleKeyDown = (e) => {
    if (!keyboardNavigationActive) return;

    const itemsPerRow = viewMode === 'grid' ? 3 : 1; // Adjust based on your grid layout
    const totalItems = filteredAgents.length;

    if (totalItems === 0) return;

    const newIndex = KeyboardNavigation.handleGridNavigation(
      e,
      focusedAgentIndex,
      itemsPerRow,
      totalItems
    );

    if (newIndex !== focusedAgentIndex) {
      setFocusedAgentIndex(newIndex);

      // Focus the agent card
      const agentCards = document.querySelectorAll('[data-agent-card]');
      if (agentCards[newIndex]) {
        agentCards[newIndex].focus();
      }

      // Announce to screen readers
      const agent = filteredAgents[newIndex];
      if (agent) {
        AriaUtils.announce(`Focused on ${agent.name}, ${agent.status}`, 'polite');
      }
    }

    // Handle Enter/Space to open agent details
    if ((e.key === 'Enter' || e.key === ' ') && focusedAgentIndex >= 0) {
      e.preventDefault();
      const agent = filteredAgents[focusedAgentIndex];
      if (agent) {
        handleViewDetails(agent);
      }
    }
  };

  const handleAgentCardFocus = (index) => {
    setFocusedAgentIndex(index);
    setKeyboardNavigationActive(true);
  };

  const handleAgentCardBlur = () => {
    // Small delay to check if focus moved to another agent card
    setTimeout(() => {
      const focusedElement = document.activeElement;
      if (!focusedElement || !focusedElement.hasAttribute('data-agent-card')) {
        setKeyboardNavigationActive(false);
        setFocusedAgentIndex(-1);
      }
    }, 100);
  };

  return (
    <div
      className="p-6 space-y-6"
      onKeyDown={handleKeyDown}
      tabIndex={-1}
    >
      {/* Notification */}
      {notification && (
        <div className={`fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg border flex items-center space-x-2 ${
          notification.type === 'success'
            ? 'bg-green-900/90 border-green-500 text-green-200'
            : 'bg-red-900/90 border-red-500 text-red-200'
        }`}>
          {notification.type === 'success' ? (
            <CheckCircle className="w-5 h-5 text-green-400" />
          ) : (
            <AlertCircle className="w-5 h-5 text-red-400" />
          )}
          <span>{notification.message}</span>
          <button
            onClick={() => setNotification(null)}
            className="ml-2 text-slate-400 hover:text-white"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">NFT-Agent Manager</h1>
          <p className="text-slate-400">Deploy and manage your agentic NFTs across target systems and environments.</p>
          {error && (
            <div className="mt-2 p-2 bg-red-500/20 border border-red-500/30 rounded-lg text-red-300">
              {error}
            </div>
          )}
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button
            onClick={() => setIsPluginImportModalOpen(true)}
            className="bg-gradient-to-r from-green-500 to-teal-500 text-white px-6 py-3 rounded-lg font-medium hover:from-green-600 hover:to-teal-600 transition-all duration-200 flex items-center space-x-2"
            disabled={isLoading}
          >
            <Download className="w-5 h-5" />
            <span>Import Agent</span>
          </button>
          <button
            onClick={() => setIsCreationModalOpen(true)}
            className="create-agent-button bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
            disabled={isLoading}
          >
            <Plus className="w-5 h-5" />
            <span>Create Agent</span>
          </button>
        </div>
      </div>

      {/* Controls */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between space-y-4 lg:space-y-0">
        <div className="flex items-center space-x-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
            <input
              type="text"
              placeholder="Search agents..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent w-64"
            />
          </div>
          <button 
            onClick={() => setIsFilterModalOpen(true)}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-2 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
          >
            <Filter className="w-5 h-5" />
            <span>Filter</span>
          </button>
        </div>
        
        <div className="flex items-center space-x-2">
          <button
            onClick={() => setViewMode('grid')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'grid' 
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30' 
                : 'bg-slate-800/50 text-slate-400 border border-slate-700/50 hover:text-white'
            }`}
            aria-label="Grid view"
          >
            <Grid className="w-5 h-5" />
          </button>
          <button
            onClick={() => setViewMode('list')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'list'
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                : 'bg-slate-800/50 text-slate-400 border border-slate-700/50 hover:text-white'
            }`}
            aria-label="List view"
          >
            <List className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Active Filters Display */}
      {getActiveFiltersDisplay()}

      {/* Categories */}
      <div className="flex flex-wrap gap-3">
        {categories.map((category) => (
          <button
            key={category.id}
            onClick={() => setSelectedCategory(category.id)}
            className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all duration-200 ${
              selectedCategory === category.id
                ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700/50 hover:text-white hover:border-slate-600/50'
            }`}
          >
            <span className="font-medium">{category.name}</span>
          </button>
        ))}
      </div>

      {/* Agent Grid/List */}
      {isLoading ? (
        <div className="flex items-center justify-center p-12">
          <div className="flex flex-col items-center space-y-4">
            <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin"></div>
            <p className="text-slate-300">Loading agents...</p>
          </div>
        </div>
      ) : filteredAgents.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-12 bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50">
          <Bot className="w-16 h-16 text-slate-500 mb-4" />
          <h3 className="text-xl font-bold text-white mb-2">No Agents Found</h3>
          <p className="text-slate-400 text-center max-w-md mb-6">
            {searchTerm || activeFilters.length > 0 || selectedCategory !== 'all' 
              ? "No agents match your current filters. Try adjusting your search criteria."
              : "You don't have any agents yet. Create your first agent to get started."}
          </p>
          <button
            onClick={() => setIsCreationModalOpen(true)}
            className="create-agent-button bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>Create Agent</span>
          </button>
        </div>
      ) : viewMode === 'grid' ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6" role="grid" aria-label="Agent cards">
          {filteredAgents.map((agent, index) => (
            <div
              key={agent.id}
              className={`bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 hover:border-slate-600/50 transition-all duration-300 group overflow-hidden cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-900 ${
                focusedAgentIndex === index ? 'ring-2 ring-blue-500' : ''
              }`}
              data-agent-card
              tabIndex={0}
              role="gridcell"
              aria-label={`Agent ${agent.name}, status: ${agent.status}`}
              onFocus={() => handleAgentCardFocus(index)}
              onBlur={handleAgentCardBlur}
              onClick={(event) => {
                // Check if the click is on a button, terminal, or other interactive element
                const target = event.target;
                
                // Check for various elements we don't want to trigger the card click
                const isButton = target.closest('button');
                const isTerminal = target.closest('.AgentTerminal') || target.closest('[data-terminal]');
                const isXtermElement = target.closest('.xterm') || target.closest('.xterm-viewport') || target.closest('.xterm-screen');
                const isConfigButton = target.closest('[data-config-button]');
                
                // Only open details if clicking on the card itself, not on interactive elements
                if (!isButton && !isTerminal && !isXtermElement) {
                  // If it's the config button, let its own click handler work
                  if (!isConfigButton) {
                    handleViewDetails(agent);
                  }
                }
              }}
            >
              <div className="relative overflow-hidden">
                <AgentImage
                  src={agent.image}
                  alt={agent.name}
                  className="w-full h-48 object-contain p-2.5 group-hover:scale-105 transition-transform duration-300"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                <div className="absolute top-3 right-3">
                  <span className={`px-3 py-1 rounded-full text-xs font-medium bg-gradient-to-r ${getStatusColor(agent.status)} text-white flex items-center space-x-1`}>
                    {getStatusIcon(agent.status)}
                    <span className="capitalize">{agent.status}</span>
                  </span>
                </div>
                <div className="absolute bottom-3 right-3 flex space-x-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                  <button
                    className="p-2 bg-black/50 backdrop-blur-sm rounded-lg text-white hover:bg-black/70 transition-colors duration-200"
                    aria-label={`View details for ${agent.name}`}
                    onClick={() => handleViewDetails(agent)}
                  >
                    <Eye className="w-4 h-4" />
                  </button>
                  <button
                    className="p-2 bg-black/50 backdrop-blur-sm rounded-lg text-white hover:bg-black/70 transition-colors duration-200"
                    onClick={() => handleConfigClick(agent)}
                    aria-label={`Settings for ${agent.name}`}
                    data-config-button="true"
                  >
                    <Settings className="w-4 h-4" />
                  </button>
                </div>
              </div>
              
              <div className="p-5 space-y-4">
                <div>
                  <h3 className="text-lg font-bold text-white mb-1">{agent.name}</h3>
                  <p className="text-slate-400 text-sm">{agent.collection}</p>
                </div>
                
                {agent.currentTarget && (
                  <div className="p-3 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-slate-400 text-sm">Current Target:</span>
                      <Target className="w-4 h-4 text-purple-400" />
                    </div>
                    <p className="text-white font-medium text-sm">{agent.currentTarget}</p>
                    <p className="text-slate-300 text-xs">{agent.capability}</p>
                  </div>
                )}
                
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-slate-400">Inferences:</span>
                    <p className="text-white font-medium">{agent.totalInferences.toLocaleString()}</p>
                  </div>
                  <div>
                    <span className="text-slate-400">Success Rate:</span>
                    <p className="text-white font-medium">{agent.successRate}%</p>
                  </div>
                </div>
                
                <div className="space-y-2">
                  <p className="text-slate-400 text-sm">Capabilities:</p>
                  <div className="flex flex-wrap gap-1">
                    {agent.capabilities.slice(0, 2).map((capability, index) => (
                      <span key={index} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                        {capability}
                      </span>
                    ))}
                    {agent.capabilities.length > 2 && (
                      <span className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                        +{agent.capabilities.length - 2} more
                      </span>
                    )}
                  </div>
                </div>

                {/* Build Target Information */}
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Build Target:</span>
                  <div className="flex items-center space-x-1">
                    {agent.type === 'wasm' ||
                     (agent.config && JSON.parse(agent.config).build_target === 'wasm') ||
                     (agent.plugin_info && agent.plugin_info.filename && agent.plugin_info.filename.endsWith('.wasm')) ? (
                      <>
                        <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                        <span className="text-green-400 font-medium">WASM</span>
                      </>
                    ) : (
                      <>
                        <div className="w-2 h-2 bg-blue-400 rounded-full"></div>
                        <span className="text-blue-400 font-medium">Plugin</span>
                      </>
                    )}
                  </div>
                </div>

                <div className="flex items-center space-x-2">
                  {agent.status === 'idle' || agent.status === 'inactive' ? (
                    <button
                      onClick={(event) => {
                        event.stopPropagation();
                        handleDeployClick(agent);
                      }}
                      className="flex-1 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
                    >
                      <Play className="w-4 h-4" />
                      <span>Deploy</span>
                    </button>
                  ) : (
                    <button
                      onClick={(event) => {
                        event.stopPropagation();
                        handleStopAgent(agent);
                      }}
                      className="flex-1 bg-gradient-to-r from-red-500/20 to-orange-500/20 hover:from-red-500/30 hover:to-orange-500/30 border border-red-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
                    >
                      <Square className="w-4 h-4" />
                      <span>Stop</span>
                    </button>
                  )}
                  {/* Terminal Button - Direct Access */}
                  <div className="relative">
                    <button
                      data-terminal-button="true"
                      onClick={(event) => {
                        event.stopPropagation();
                        handleTerminalToggle(agent);
                      }}
                      className={`border p-2 rounded-lg transition-all duration-200 flex items-center ${
                        agentTerminals[agent.id]?.isVisible
                          ? 'bg-green-500/20 border-green-500/30 text-green-400 hover:text-green-300'
                          : 'bg-slate-700/50 border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50'
                      }`}
                      title="Open Terminal"
                    >
                      <Terminal className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                {/* Agent Terminal */}
                {agentTerminals[agent.id] && (
                  <div className="mt-4">
                    <div>
                      <AgentTerminal
                        agent={agent}
                        sessionId={agentSessions[agent.id]}
                        isMinimized={agentTerminals[agent.id].isMinimized}
                        onMinimize={() => handleTerminalMinimize(agent.id)}
                        onMaximize={() => handleTerminalMaximize(agent.id)}
                        onClose={() => handleTerminalClose(agent.id)}
                      />
                      
                      {/* Chat Button */}
                      <button
                        onClick={() => handleOpenChat(agent)}
                        className="mt-2 w-full flex items-center justify-center space-x-2 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white py-2 px-4 rounded-lg border border-slate-700 transition-colors duration-200"
                      >
                        <MessageSquare className="w-4 h-4" />
                        <span className="text-sm">Open Chat</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-700/30 border-b border-slate-600/50">
                <tr>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Agent</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Status</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Current Target</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Inferences</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Success Rate</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Last Activity</th>
                  <th className="text-left py-4 px-6 text-slate-300 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700/50">
                {filteredAgents.map((agent) => (
                  <tr key={agent.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-3">
                        <AgentImage src={agent.image} alt={agent.name} className="w-12 h-12 rounded-lg object-contain p-1" />
                        <div>
                          <p className="text-white font-medium">{agent.name}</p>
                          <p className="text-slate-400 text-sm">{agent.collection}</p>
                        </div>
                      </div>
                    </td>
                    <td className="py-4 px-6">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium bg-gradient-to-r ${getStatusColor(agent.status)} text-white flex items-center space-x-1 w-fit`}>
                        {getStatusIcon(agent.status)}
                        <span className="capitalize">{agent.status}</span>
                      </span>
                    </td>
                    <td className="py-4 px-6">
                      {agent.currentTarget ? (
                        <div>
                          <p className="text-white font-medium">{agent.currentTarget}</p>
                          <p className="text-slate-400 text-sm">{agent.capability}</p>
                        </div>
                      ) : (
                        <span className="text-slate-500">—</span>
                      )}
                    </td>
                    <td className="py-4 px-6 text-white font-medium">{agent.totalInferences.toLocaleString()}</td>
                    <td className="py-4 px-6 text-white font-medium">{agent.successRate}%</td>
                    <td className="py-4 px-6 text-slate-300">{agent.lastActivity}</td>
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-2">
                        {agent.status === 'idle' || agent.status === 'inactive' ? (
                          <button 
                            onClick={() => handleDeployClick(agent)}
                            className="p-2 text-green-400 hover:text-green-300 transition-colors duration-200"
                          >
                            <Play className="w-4 h-4" />
                          </button>
                        ) : (
                          <button
                            data-stop-button="true"
                            onClick={() => handleStopAgent(agent)}
                            className="p-2 text-red-400 hover:text-red-300 transition-colors duration-200"
                          >
                            <Square className="w-4 h-4" />
                          </button>
                        )}
                        <button
                          onClick={() => handleViewDetails(agent)}
                          className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                          aria-label={`View details for ${agent.name}`}
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleConfigClick(agent)}
                          className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                          aria-label={`Settings for ${agent.name}`}
                          data-config-button="true"
                        >
                          <Settings className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Modals */}
      <AgentCreationModal 
        isOpen={isCreationModalOpen} 
        onClose={() => setIsCreationModalOpen(false)} 
        onAgentCreated={handleAgentCreated} 
      />
      
      <AgentDeploymentModal 
        isOpen={isDeploymentModalOpen} 
        onClose={() => {
          setIsDeploymentModalOpen(false);
          setSelectedAgent(null);
        }} 
        agent={selectedAgent} 
        onAgentDeployed={handleAgentDeployed} 
      />
      
      <AgentConfigModal 
        isOpen={isConfigModalOpen} 
        onClose={() => {
          setIsConfigModalOpen(false);
          setSelectedAgent(null);
        }} 
        agent={selectedAgent} 
        onAgentUpdated={handleAgentUpdated} 
        onDelete={handleDeleteAgent}
      />

      <AdvancedFilterModal
        isOpen={isFilterModalOpen}
        onClose={() => setIsFilterModalOpen(false)}
        onApplyFilters={applyAdvancedFilters}
        initialFilters={activeFilters}
        filterOptions={filterOptions}
        entityType="agent"
      />

      <AgentDetailModal
        isOpen={isDetailModalOpen}
        onClose={() => {
          setIsDetailModalOpen(false);
          setSelectedAgent(null);
        }}
        agent={selectedAgent}
        onDeploy={handleDeployClick}
        onStop={handleStopAgent}
        onConfigure={handleConfigClick}
        onDelete={handleDeleteAgent}
      />

      <PluginImportModal
        isOpen={isPluginImportModalOpen}
        onClose={() => setIsPluginImportModalOpen(false)}
        onPluginImported={handlePluginImported}
      />

      <AgentChatModal
        isOpen={isChatModalOpen}
        onClose={handleCloseChat}
        agent={selectedChatAgent}
      />
    </div>
  );
};
