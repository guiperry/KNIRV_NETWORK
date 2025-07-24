import React, { useState, useEffect, useRef } from 'react';
import {
  Search,
  Filter,
  Plus,
  Save,
  Play,
  Pause,
  Settings,
  Trash2,
  Database,
  Server,
  ArrowRight,
  ArrowUpRight,
  ArrowDownRight,
  FileText,
  BarChart3,
  AlertTriangle,
  CheckCircle,
  RefreshCw,
  Eye,
  Download,
  Upload,
  Code,
  Layers,
  GitBranch,
  Zap,
  Cpu,
  Clock,
  Maximize2,
  Minimize2,
  X,
  Copy
} from 'lucide-react';
import DataPipelineModal from './modals/DataPipelineModal';
import DataComponentModal from './modals/DataComponentModal';
import AdvancedFilterModal from './modals/AdvancedFilterModal';

export const DataEngineeringInterface = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [isPipelineModalOpen, setIsPipelineModalOpen] = useState(false);
  const [isComponentModalOpen, setIsComponentModalOpen] = useState(false);
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false);
  const [selectedPipeline, setSelectedPipeline] = useState(null);
  const [selectedComponent, setSelectedComponent] = useState(null);
  const [activeFilters, setActiveFilters] = useState([]);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const canvasRef = useRef(null);
  const [canvasScale, setCanvasScale] = useState(1);
  const [canvasOffset, setCanvasOffset] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const categories = [
    { id: 'all', name: 'All Pipelines' },
    { id: 'active', name: 'Active' },
    { id: 'paused', name: 'Paused' },
    { id: 'draft', name: 'Draft' },
    { id: 'error', name: 'Error' }
  ];

  const [pipelines, setPipelines] = useState([
    {
      id: 1,
      name: 'User Interaction Pipeline',
      description: 'Processes clickstream data and user interactions',
      status: 'active',
      lastRun: '2 minutes ago',
      nextRun: '5 minutes from now',
      createdBy: 'Data Team',
      createdAt: '2024-01-15T10:30:00Z',
      components: [
        { id: 101, type: 'source', name: 'Clickstream Ingestion', position: { x: 100, y: 150 } },
        { id: 102, type: 'processor', name: 'Event Filtering', position: { x: 300, y: 150 } },
        { id: 103, type: 'processor', name: 'User Session Aggregation', position: { x: 500, y: 150 } },
        { id: 104, type: 'sink', name: 'Dashboard Output', position: { x: 700, y: 150 } }
      ],
      connections: [
        { source: 101, target: 102 },
        { source: 102, target: 103 },
        { source: 103, target: 104 }
      ],
      metrics: {
        eventsProcessed: 15420,
        processingTime: '1.2s',
        errorRate: '0.02%',
        throughput: '5200 events/min'
      }
    },
    {
      id: 2,
      name: 'Content Analysis Pipeline',
      description: 'Analyzes content and extracts insights',
      status: 'paused',
      lastRun: '1 hour ago',
      nextRun: 'Manually triggered',
      createdBy: 'Content Team',
      createdAt: '2024-01-10T14:45:00Z',
      components: [
        { id: 201, type: 'source', name: 'Content API', position: { x: 100, y: 150 } },
        { id: 202, type: 'processor', name: 'Text Extraction', position: { x: 300, y: 100 } },
        { id: 203, type: 'processor', name: 'Image Analysis', position: { x: 300, y: 200 } },
        { id: 204, type: 'processor', name: 'Sentiment Analysis', position: { x: 500, y: 100 } },
        { id: 205, type: 'processor', name: 'Topic Classification', position: { x: 500, y: 200 } },
        { id: 206, type: 'sink', name: 'Analytics Database', position: { x: 700, y: 150 } }
      ],
      connections: [
        { source: 201, target: 202 },
        { source: 201, target: 203 },
        { source: 202, target: 204 },
        { source: 203, target: 205 },
        { source: 204, target: 206 },
        { source: 205, target: 206 }
      ],
      metrics: {
        eventsProcessed: 8750,
        processingTime: '3.5s',
        errorRate: '0.5%',
        throughput: '2500 events/min'
      }
    },
    {
      id: 3,
      name: 'Real-time Monitoring Pipeline',
      description: 'Monitors system metrics and generates alerts',
      status: 'error',
      lastRun: '15 minutes ago',
      nextRun: 'Error recovery in progress',
      createdBy: 'Operations Team',
      createdAt: '2023-12-05T09:15:00Z',
      components: [
        { id: 301, type: 'source', name: 'System Metrics', position: { x: 100, y: 150 } },
        { id: 302, type: 'processor', name: 'Anomaly Detection', position: { x: 300, y: 150 } },
        { id: 303, type: 'processor', name: 'Alert Generation', position: { x: 500, y: 100 } },
        { id: 304, type: 'sink', name: 'Notification Service', position: { x: 700, y: 100 } },
        { id: 305, type: 'sink', name: 'Metrics Database', position: { x: 700, y: 200 } }
      ],
      connections: [
        { source: 301, target: 302 },
        { source: 302, target: 303 },
        { source: 302, target: 305 },
        { source: 303, target: 304 }
      ],
      metrics: {
        eventsProcessed: 12340,
        processingTime: '0.8s',
        errorRate: '4.2%',
        throughput: '4100 events/min'
      },
      error: {
        component: 'Anomaly Detection',
        message: 'Memory allocation exceeded',
        timestamp: '2024-01-15T15:30:00Z',
        details: 'Component crashed due to insufficient memory when processing large metric batch'
      }
    },
    {
      id: 4,
      name: 'Data Warehouse ETL Pipeline',
      description: 'Extracts, transforms, and loads data into the warehouse',
      status: 'draft',
      lastRun: 'Never',
      nextRun: 'Not scheduled',
      createdBy: 'Data Engineering',
      createdAt: '2024-01-14T16:20:00Z',
      components: [
        { id: 401, type: 'source', name: 'Database Connector', position: { x: 100, y: 150 } },
        { id: 402, type: 'processor', name: 'Data Transformation', position: { x: 300, y: 150 } },
        { id: 403, type: 'processor', name: 'Data Validation', position: { x: 500, y: 150 } },
        { id: 404, type: 'sink', name: 'Data Warehouse', position: { x: 700, y: 150 } }
      ],
      connections: [
        { source: 401, target: 402 },
        { source: 402, target: 403 },
        { source: 403, target: 404 }
      ],
      metrics: {
        eventsProcessed: 0,
        processingTime: 'N/A',
        errorRate: 'N/A',
        throughput: 'N/A'
      }
    }
  ]);

  const [components, setComponents] = useState([
    // Sources
    { id: 1, type: 'source', name: 'Kafka Source', description: 'Ingest data from Kafka topics', category: 'streaming' },
    { id: 2, type: 'source', name: 'Database Source', description: 'Extract data from SQL databases', category: 'database' },
    { id: 3, type: 'source', name: 'File Source', description: 'Read data from files (CSV, JSON, etc.)', category: 'file' },
    { id: 4, type: 'source', name: 'API Source', description: 'Fetch data from REST APIs', category: 'api' },
    { id: 5, type: 'source', name: 'Event Source', description: 'Capture application events', category: 'event' },
    
    // Processors
    { id: 6, type: 'processor', name: 'Filter', description: 'Filter data based on conditions', category: 'transform' },
    { id: 7, type: 'processor', name: 'Map', description: 'Transform data fields', category: 'transform' },
    { id: 8, type: 'processor', name: 'Aggregate', description: 'Group and aggregate data', category: 'transform' },
    { id: 9, type: 'processor', name: 'Join', description: 'Join multiple data streams', category: 'transform' },
    { id: 10, type: 'processor', name: 'Window', description: 'Process data in time windows', category: 'transform' },
    { id: 11, type: 'processor', name: 'ML Predictor', description: 'Apply machine learning models', category: 'ml' },
    { id: 12, type: 'processor', name: 'Anomaly Detector', description: 'Detect anomalies in data', category: 'ml' },
    
    // Sinks
    { id: 13, type: 'sink', name: 'Database Sink', description: 'Write data to databases', category: 'database' },
    { id: 14, type: 'sink', name: 'File Sink', description: 'Write data to files', category: 'file' },
    { id: 15, type: 'sink', name: 'Kafka Sink', description: 'Send data to Kafka topics', category: 'streaming' },
    { id: 16, type: 'sink', name: 'Dashboard Sink', description: 'Visualize data in dashboards', category: 'visualization' },
    { id: 17, type: 'sink', name: 'Alert Sink', description: 'Generate alerts and notifications', category: 'notification' }
  ]);

  // Filter options for the advanced filter modal
  const filterOptions = {
    fields: [
      { id: 'name', name: 'Name' },
      { id: 'description', name: 'Description' },
      { id: 'status', name: 'Status' },
      { id: 'createdBy', name: 'Created By' },
      { id: 'createdAt', name: 'Created Date' },
      { id: 'metrics.eventsProcessed', name: 'Events Processed' },
      { id: 'metrics.errorRate', name: 'Error Rate' }
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

  // Check if a pipeline matches the advanced filters
  const matchesAdvancedFilters = (pipeline) => {
    if (activeFilters.length === 0) return true;
    
    return activeFilters.every(filter => {
      const { field, operator, value } = filter;
      
      // Handle nested fields like metrics.eventsProcessed
      let pipelineValue;
      if (field.includes('.')) {
        const [parent, child] = field.split('.');
        pipelineValue = pipeline[parent] ? pipeline[parent][child] : undefined;
      } else {
        pipelineValue = pipeline[field];
      }
      
      // Handle null/undefined values
      if (pipelineValue === null || pipelineValue === undefined) {
        return operator === 'not_equals' || operator === 'not_contains';
      }
      
      // Convert to string for comparison
      const pipelineValueStr = String(pipelineValue).toLowerCase();
      const filterValueStr = String(value).toLowerCase();
      
      switch (operator) {
        case 'equals':
          return pipelineValueStr === filterValueStr;
        case 'not_equals':
          return pipelineValueStr !== filterValueStr;
        case 'contains':
          return pipelineValueStr.includes(filterValueStr);
        case 'not_contains':
          return !pipelineValueStr.includes(filterValueStr);
        case 'greater_than':
          return parseFloat(pipelineValueStr) > parseFloat(filterValueStr);
        case 'less_than':
          return parseFloat(pipelineValueStr) < parseFloat(filterValueStr);
        case 'starts_with':
          return pipelineValueStr.startsWith(filterValueStr);
        case 'ends_with':
          return pipelineValueStr.endsWith(filterValueStr);
        default:
          return true;
      }
    });
  };

  const filteredPipelines = pipelines.filter(pipeline => {
    const matchesSearch = pipeline.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         pipeline.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || pipeline.status === selectedCategory;
    const matchesAdvanced = matchesAdvancedFilters(pipeline);
    return matchesSearch && matchesCategory && matchesAdvanced;
  });

  // Initialize canvas when component mounts
  useEffect(() => {
    setIsLoading(true);
    // Simulate loading data
    setTimeout(() => {
      setIsLoading(false);
    }, 1000);
  }, []);

  // Draw pipeline on canvas when selectedPipeline changes
  useEffect(() => {
    if (!selectedPipeline || !canvasRef.current) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    const pipeline = pipelines.find(p => p.id === selectedPipeline);
    
    if (!pipeline) return;
    
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    // Apply scale and offset transformations
    ctx.save();
    ctx.translate(canvasOffset.x, canvasOffset.y);
    ctx.scale(canvasScale, canvasScale);
    
    // Draw connections first (so they appear behind components)
    ctx.lineWidth = 2;
    ctx.strokeStyle = '#8b5cf6'; // Purple
    
    pipeline.connections.forEach(connection => {
      const source = pipeline.components.find(c => c.id === connection.source);
      const target = pipeline.components.find(c => c.id === connection.target);
      
      if (source && target) {
        // Calculate connection points
        const sourceX = source.position.x + 80; // Right side of source
        const sourceY = source.position.y + 40; // Middle of source
        const targetX = target.position.x; // Left side of target
        const targetY = target.position.y + 40; // Middle of target
        
        // Draw connection line with arrow
        ctx.beginPath();
        ctx.moveTo(sourceX, sourceY);
        
        // Calculate control points for curve
        const controlX = (sourceX + targetX) / 2;
        
        ctx.bezierCurveTo(
          controlX, sourceY, // First control point
          controlX, targetY, // Second control point
          targetX, targetY // End point
        );
        
        ctx.stroke();
        
        // Draw arrow at end
        const arrowSize = 8;
        const angle = Math.atan2(targetY - (controlX, targetY).y, targetX - (controlX, targetY).x);
        
        ctx.beginPath();
        ctx.moveTo(targetX, targetY);
        ctx.lineTo(
          targetX - arrowSize * Math.cos(angle - Math.PI / 6),
          targetY - arrowSize * Math.sin(angle - Math.PI / 6)
        );
        ctx.lineTo(
          targetX - arrowSize * Math.cos(angle + Math.PI / 6),
          targetY - arrowSize * Math.sin(angle + Math.PI / 6)
        );
        ctx.closePath();
        ctx.fillStyle = '#8b5cf6';
        ctx.fill();
      }
    });
    
    // Draw components
    pipeline.components.forEach(component => {
      // Component background based on type
      let bgColor;
      let borderColor;
      
      switch (component.type) {
        case 'source':
          bgColor = 'rgba(59, 130, 246, 0.2)'; // Blue
          borderColor = 'rgba(59, 130, 246, 0.5)';
          break;
        case 'processor':
          bgColor = 'rgba(139, 92, 246, 0.2)'; // Purple
          borderColor = 'rgba(139, 92, 246, 0.5)';
          break;
        case 'sink':
          bgColor = 'rgba(16, 185, 129, 0.2)'; // Green
          borderColor = 'rgba(16, 185, 129, 0.5)';
          break;
        default:
          bgColor = 'rgba(107, 114, 128, 0.2)'; // Gray
          borderColor = 'rgba(107, 114, 128, 0.5)';
      }
      
      // Draw component box
      ctx.fillStyle = bgColor;
      ctx.strokeStyle = borderColor;
      ctx.lineWidth = 2;
      
      // Rounded rectangle
      const x = component.position.x;
      const y = component.position.y;
      const width = 160;
      const height = 80;
      const radius = 8;
      
      ctx.beginPath();
      ctx.moveTo(x + radius, y);
      ctx.lineTo(x + width - radius, y);
      ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
      ctx.lineTo(x + width, y + height - radius);
      ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
      ctx.lineTo(x + radius, y + height);
      ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
      ctx.lineTo(x, y + radius);
      ctx.quadraticCurveTo(x, y, x + radius, y);
      ctx.closePath();
      
      ctx.fill();
      ctx.stroke();
      
      // Draw component name
      ctx.fillStyle = '#ffffff'; // White text
      ctx.font = '14px Arial';
      ctx.textAlign = 'center';
      ctx.fillText(component.name, x + width / 2, y + height / 2);
      
      // Draw component type
      ctx.fillStyle = 'rgba(255, 255, 255, 0.7)'; // Slightly transparent white
      ctx.font = '12px Arial';
      ctx.fillText(component.type, x + width / 2, y + height / 2 + 20);
    });
    
    // Restore canvas context
    ctx.restore();
  }, [selectedPipeline, canvasScale, canvasOffset, pipelines]);

  const handleAddPipeline = () => {
    setSelectedPipeline(null);
    setIsPipelineModalOpen(true);
  };

  const handleEditPipeline = (pipeline) => {
    setSelectedPipeline(pipeline.id);
    setIsPipelineModalOpen(true);
  };

  const handlePipelineSaved = (pipelineData) => {
    if (selectedPipeline) {
      // Update existing pipeline
      setPipelines(prevPipelines => 
        prevPipelines.map(pipeline => 
          pipeline.id === pipelineData.id ? pipelineData : pipeline
        )
      );
    } else {
      // Add new pipeline
      setPipelines(prevPipelines => [...prevPipelines, pipelineData]);
    }
    setIsPipelineModalOpen(false);
    setSelectedPipeline(null);
  };

  const handleAddComponent = () => {
    setSelectedComponent(null);
    setIsComponentModalOpen(true);
  };

  const handleEditComponent = (component) => {
    setSelectedComponent(component);
    setIsComponentModalOpen(true);
  };

  const handleComponentSaved = (componentData) => {
    if (selectedComponent) {
      // Update existing component
      setComponents(prevComponents => 
        prevComponents.map(component => 
          component.id === componentData.id ? componentData : component
        )
      );
    } else {
      // Add new component
      setComponents(prevComponents => [...prevComponents, componentData]);
    }
    setIsComponentModalOpen(false);
    setSelectedComponent(null);
  };

  const handleStartStopPipeline = (pipeline) => {
    setPipelines(prevPipelines => 
      prevPipelines.map(p => 
        p.id === pipeline.id 
          ? { 
              ...p, 
              status: p.status === 'active' ? 'paused' : p.status === 'paused' ? 'active' : p.status,
              lastRun: p.status === 'paused' ? 'just now' : p.lastRun,
              nextRun: p.status === 'paused' ? '5 minutes from now' : 'Manually triggered'
            } 
          : p
      )
    );
  };

  const handleViewPipeline = (pipeline) => {
    setSelectedPipeline(pipeline.id);
  };

  const handleCanvasMouseDown = (e) => {
    if (e.button === 0) { // Left mouse button
      setIsDragging(true);
      setDragStart({
        x: e.clientX - canvasOffset.x,
        y: e.clientY - canvasOffset.y
      });
    }
  };

  const handleCanvasMouseMove = (e) => {
    if (isDragging) {
      setCanvasOffset({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y
      });
    }
  };

  const handleCanvasMouseUp = () => {
    setIsDragging(false);
  };

  const handleCanvasWheel = (e) => {
    e.preventDefault();
    const scaleFactor = 0.1;
    const delta = e.deltaY < 0 ? scaleFactor : -scaleFactor;
    const newScale = Math.max(0.5, Math.min(2, canvasScale + delta));
    setCanvasScale(newScale);
  };

  const resetCanvasView = () => {
    setCanvasScale(1);
    setCanvasOffset({ x: 0, y: 0 });
  };

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
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

  const getStatusColor = (status) => {
    switch (status) {
      case 'active':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'paused':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/30';
      case 'error':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      case 'draft':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'active':
        return <Play className="w-4 h-4" />;
      case 'paused':
        return <Pause className="w-4 h-4" />;
      case 'error':
        return <AlertTriangle className="w-4 h-4" />;
      case 'draft':
        return <FileText className="w-4 h-4" />;
      default:
        return <Clock className="w-4 h-4" />;
    }
  };

  const getComponentTypeIcon = (type) => {
    switch (type) {
      case 'source':
        return <Database className="w-4 h-4 text-blue-400" />;
      case 'processor':
        return <Cpu className="w-4 h-4 text-purple-400" />;
      case 'sink':
        return <Save className="w-4 h-4 text-green-400" />;
      default:
        return <Layers className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className={`p-6 space-y-6 ${isFullscreen ? 'fixed inset-0 z-50 bg-slate-900' : ''}`}>
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Data Engineering Interface</h1>
          <p className="text-slate-400">Design, deploy, and monitor data pipelines for agent orchestration.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button 
            onClick={handleAddComponent}
            className="bg-slate-800/50 border border-slate-700/50 text-slate-300 px-4 py-2 rounded-lg hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>Add Component</span>
          </button>
          <button 
            onClick={handleAddPipeline}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <GitBranch className="w-5 h-5" />
            <span>Create Pipeline</span>
          </button>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col lg:flex-row lg:items-center space-y-4 lg:space-y-0 lg:space-x-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search pipelines..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-3 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
        <button 
          onClick={() => setIsFilterModalOpen(true)}
          className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-3 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
        >
          <Filter className="w-5 h-5" />
          <span>Filter</span>
        </button>
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

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Pipelines List */}
        <div className="lg:col-span-1 space-y-4">
          <h2 className="text-xl font-bold text-white">Data Pipelines</h2>
          
          {isLoading ? (
            <div className="flex items-center justify-center py-12 bg-slate-800/50 rounded-xl border border-slate-700/50">
              <div className="w-8 h-8 border-4 border-purple-500/30 border-t-purple-500 rounded-full animate-spin"></div>
              <span className="ml-3 text-slate-400">Loading pipelines...</span>
            </div>
          ) : filteredPipelines.length === 0 ? (
            <div className="text-center py-12 bg-slate-800/50 rounded-xl border border-slate-700/50">
              <GitBranch className="w-12 h-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400">No pipelines found</p>
              <button 
                onClick={handleAddPipeline}
                className="mt-4 px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30 transition-colors duration-200"
              >
                Create your first pipeline
              </button>
            </div>
          ) : (
            <div className="space-y-3 max-h-[calc(100vh-350px)] overflow-y-auto pr-2">
              {filteredPipelines.map((pipeline) => (
                <div 
                  key={pipeline.id}
                  className={`p-4 bg-slate-800/50 rounded-xl border border-slate-700/50 hover:border-slate-600/50 transition-all duration-200 cursor-pointer ${
                    selectedPipeline === pipeline.id ? 'ring-2 ring-purple-500/50' : ''
                  }`}
                  onClick={() => handleViewPipeline(pipeline)}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="text-white font-medium">{pipeline.name}</h3>
                      <p className="text-slate-400 text-sm mt-1">{pipeline.description}</p>
                    </div>
                    <span className={`px-2 py-1 rounded-full text-xs font-medium flex items-center space-x-1 ${getStatusColor(pipeline.status)}`}>
                      {getStatusIcon(pipeline.status)}
                      <span className="capitalize">{pipeline.status}</span>
                    </span>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-2 mt-3 text-xs">
                    <div>
                      <p className="text-slate-500">Last Run</p>
                      <p className="text-slate-300">{pipeline.lastRun}</p>
                    </div>
                    <div>
                      <p className="text-slate-500">Next Run</p>
                      <p className="text-slate-300">{pipeline.nextRun}</p>
                    </div>
                    <div>
                      <p className="text-slate-500">Components</p>
                      <p className="text-slate-300">{pipeline.components.length}</p>
                    </div>
                    <div>
                      <p className="text-slate-500">Created By</p>
                      <p className="text-slate-300">{pipeline.createdBy}</p>
                    </div>
                  </div>
                  
                  <div className="flex items-center justify-between mt-4 pt-3 border-t border-slate-700/50">
                    {pipeline.status === 'active' ? (
                      <button 
                        onClick={(e) => {
                          e.stopPropagation();
                          handleStartStopPipeline(pipeline);
                        }}
                        className="px-3 py-1 bg-yellow-500/20 text-yellow-400 rounded-lg hover:bg-yellow-500/30 transition-colors duration-200 flex items-center space-x-1"
                      >
                        <Pause className="w-3 h-3" />
                        <span>Pause</span>
                      </button>
                    ) : pipeline.status === 'paused' ? (
                      <button 
                        onClick={(e) => {
                          e.stopPropagation();
                          handleStartStopPipeline(pipeline);
                        }}
                        className="px-3 py-1 bg-green-500/20 text-green-400 rounded-lg hover:bg-green-500/30 transition-colors duration-200 flex items-center space-x-1"
                      >
                        <Play className="w-3 h-3" />
                        <span>Start</span>
                      </button>
                    ) : (
                      <div className="px-3 py-1 text-slate-400 text-xs">
                        {pipeline.status === 'error' ? 'Error state' : 'Draft'}
                      </div>
                    )}
                    
                    <div className="flex items-center space-x-2">
                      <button 
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEditPipeline(pipeline);
                        }}
                        className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                      >
                        <Settings className="w-4 h-4" />
                      </button>
                      <button 
                        onClick={(e) => {
                          e.stopPropagation();
                          // In a real implementation, this would show a confirmation dialog
                          setPipelines(prevPipelines => prevPipelines.filter(p => p.id !== pipeline.id));
                          if (selectedPipeline === pipeline.id) {
                            setSelectedPipeline(null);
                          }
                        }}
                        className="p-1 text-slate-400 hover:text-red-400 transition-colors duration-200"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
        
        {/* Pipeline Visualization */}
        <div className="lg:col-span-2 bg-slate-800/50 rounded-xl border border-slate-700/50 overflow-hidden relative">
          <div className="absolute top-4 right-4 flex items-center space-x-2 z-10">
            <button 
              onClick={resetCanvasView}
              className="p-2 bg-slate-700/50 rounded-lg text-slate-300 hover:text-white transition-colors duration-200"
              title="Reset View"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button 
              onClick={toggleFullscreen}
              className="p-2 bg-slate-700/50 rounded-lg text-slate-300 hover:text-white transition-colors duration-200"
              title={isFullscreen ? "Exit Fullscreen" : "Fullscreen"}
            >
              {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
            </button>
          </div>
          
          {selectedPipeline ? (
            <div 
              className="w-full h-[500px] overflow-hidden"
              onMouseDown={handleCanvasMouseDown}
              onMouseMove={handleCanvasMouseMove}
              onMouseUp={handleCanvasMouseUp}
              onMouseLeave={handleCanvasMouseUp}
              onWheel={handleCanvasWheel}
            >
              <canvas 
                ref={canvasRef} 
                width={800} 
                height={500}
                className="w-full h-full bg-slate-900/50"
              />
              
              <div className="absolute bottom-4 left-4 bg-slate-800/80 backdrop-blur-sm p-3 rounded-lg border border-slate-700/50">
                <div className="text-white font-medium mb-1">
                  {pipelines.find(p => p.id === selectedPipeline)?.name}
                </div>
                <div className="flex items-center space-x-2 text-xs text-slate-400">
                  <span>Scale: {Math.round(canvasScale * 100)}%</span>
                  <span>•</span>
                  <span>Components: {pipelines.find(p => p.id === selectedPipeline)?.components.length}</span>
                </div>
              </div>
              
              {/* Pipeline metrics */}
              <div className="absolute bottom-4 right-4 bg-slate-800/80 backdrop-blur-sm p-3 rounded-lg border border-slate-700/50">
                <div className="text-white font-medium mb-1">Pipeline Metrics</div>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                  <div>
                    <span className="text-slate-400">Events: </span>
                    <span className="text-slate-300">{pipelines.find(p => p.id === selectedPipeline)?.metrics.eventsProcessed.toLocaleString()}</span>
                  </div>
                  <div>
                    <span className="text-slate-400">Avg Time: </span>
                    <span className="text-slate-300">{pipelines.find(p => p.id === selectedPipeline)?.metrics.processingTime}</span>
                  </div>
                  <div>
                    <span className="text-slate-400">Error Rate: </span>
                    <span className="text-slate-300">{pipelines.find(p => p.id === selectedPipeline)?.metrics.errorRate}</span>
                  </div>
                  <div>
                    <span className="text-slate-400">Throughput: </span>
                    <span className="text-slate-300">{pipelines.find(p => p.id === selectedPipeline)?.metrics.throughput}</span>
                  </div>
                </div>
              </div>
              
              {/* Error alert if pipeline has error */}
              {pipelines.find(p => p.id === selectedPipeline)?.status === 'error' && (
                <div className="absolute top-4 left-4 bg-red-500/20 backdrop-blur-sm p-3 rounded-lg border border-red-500/30 max-w-md">
                  <div className="flex items-start space-x-2">
                    <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5" />
                    <div>
                      <div className="text-red-400 font-medium">Pipeline Error</div>
                      <div className="text-red-300 text-sm">
                        {pipelines.find(p => p.id === selectedPipeline)?.error.component}: {pipelines.find(p => p.id === selectedPipeline)?.error.message}
                      </div>
                      <div className="text-red-300/70 text-xs mt-1">
                        {pipelines.find(p => p.id === selectedPipeline)?.error.details}
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-[500px] text-center">
              <GitBranch className="w-16 h-16 text-slate-700 mb-4" />
              <h3 className="text-xl font-semibold text-white mb-2">No Pipeline Selected</h3>
              <p className="text-slate-400 max-w-md">Select a pipeline from the list to visualize its components and connections.</p>
              <button 
                onClick={handleAddPipeline}
                className="mt-6 px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30 transition-colors duration-200 flex items-center space-x-2"
              >
                <Plus className="w-4 h-4" />
                <span>Create New Pipeline</span>
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Component Library */}
      <div className="bg-slate-800/50 rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-white">Component Library</h2>
          <button 
            onClick={handleAddComponent}
            className="text-purple-400 hover:text-purple-300 text-sm font-medium flex items-center space-x-1"
          >
            <Plus className="w-4 h-4" />
            <span>Add Component</span>
          </button>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {components.map((component) => (
            <div 
              key={component.id}
              className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30 hover:bg-slate-700/50 transition-colors duration-200 cursor-pointer"
              onClick={() => handleEditComponent(component)}
            >
              <div className="flex items-start space-x-3">
                <div className={`p-2 rounded-lg ${
                  component.type === 'source' ? 'bg-blue-500/20 border border-blue-500/30' :
                  component.type === 'processor' ? 'bg-purple-500/20 border border-purple-500/30' :
                  'bg-green-500/20 border border-green-500/30'
                }`}>
                  {getComponentTypeIcon(component.type)}
                </div>
                <div>
                  <h3 className="text-white font-medium">{component.name}</h3>
                  <p className="text-slate-400 text-xs mt-1">{component.description}</p>
                  <div className="flex items-center mt-2 space-x-2">
                    <span className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full capitalize">
                      {component.type}
                    </span>
                    <span className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full">
                      {component.category}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Monitoring Dashboard */}
      <div className="bg-slate-800/50 rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-white">System Monitoring</h2>
          <div className="flex items-center space-x-3">
            <select className="bg-slate-700/50 border border-slate-600/50 text-white px-3 py-1 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500">
              <option>Last 24 hours</option>
              <option>Last 7 days</option>
              <option>Last 30 days</option>
            </select>
            <button className="text-purple-400 hover:text-purple-300 text-sm font-medium flex items-center space-x-1">
              <RefreshCw className="w-4 h-4" />
              <span>Refresh</span>
            </button>
          </div>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
            <div className="flex items-center justify-between mb-2">
              <p className="text-slate-400 text-sm">Total Events Processed</p>
              <ArrowUpRight className="w-4 h-4 text-green-400" />
            </div>
            <p className="text-2xl font-bold text-white mb-1">36,510</p>
            <div className="flex items-center">
              <span className="text-green-400 text-sm font-medium">+12.5%</span>
              <span className="text-slate-500 text-xs ml-1">vs yesterday</span>
            </div>
            <div className="mt-3 h-10 flex items-end space-x-1">
              {[35, 42, 38, 45, 40, 48, 52, 48, 55, 60].map((value, index) => (
                <div 
                  key={index}
                  className="flex-1 bg-purple-500/50 rounded-t"
                  style={{ height: `${value}%` }}
                ></div>
              ))}
            </div>
          </div>
          
          <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
            <div className="flex items-center justify-between mb-2">
              <p className="text-slate-400 text-sm">Average Processing Time</p>
              <ArrowDownRight className="w-4 h-4 text-green-400" />
            </div>
            <p className="text-2xl font-bold text-white mb-1">2.4s</p>
            <div className="flex items-center">
              <span className="text-green-400 text-sm font-medium">-0.3s</span>
              <span className="text-slate-500 text-xs ml-1">vs yesterday</span>
            </div>
            <div className="mt-3 h-10 flex items-end space-x-1">
              {[60, 55, 58, 50, 45, 40, 35, 30, 25, 20].map((value, index) => (
                <div 
                  key={index}
                  className="flex-1 bg-blue-500/50 rounded-t"
                  style={{ height: `${value}%` }}
                ></div>
              ))}
            </div>
          </div>
          
          <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
            <div className="flex items-center justify-between mb-2">
              <p className="text-slate-400 text-sm">Error Rate</p>
              <ArrowDownRight className="w-4 h-4 text-green-400" />
            </div>
            <p className="text-2xl font-bold text-white mb-1">1.2%</p>
            <div className="flex items-center">
              <span className="text-green-400 text-sm font-medium">-0.5%</span>
              <span className="text-slate-500 text-xs ml-1">vs yesterday</span>
            </div>
            <div className="mt-3 h-10 flex items-end space-x-1">
              {[40, 35, 30, 25, 20, 15, 18, 12, 10, 8].map((value, index) => (
                <div 
                  key={index}
                  className="flex-1 bg-red-500/50 rounded-t"
                  style={{ height: `${value}%` }}
                ></div>
              ))}
            </div>
          </div>
          
          <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
            <div className="flex items-center justify-between mb-2">
              <p className="text-slate-400 text-sm">Active Pipelines</p>
              <div className="w-4 h-4"></div>
            </div>
            <p className="text-2xl font-bold text-white mb-1">{pipelines.filter(p => p.status === 'active').length}</p>
            <div className="flex items-center">
              <span className="text-slate-400 text-sm">of {pipelines.length} total</span>
            </div>
            <div className="mt-3 flex items-center space-x-2">
              <div 
                className="h-2 bg-green-500 rounded"
                style={{ width: `${(pipelines.filter(p => p.status === 'active').length / pipelines.length) * 100}%` }}
              ></div>
              <div 
                className="h-2 bg-yellow-500 rounded"
                style={{ width: `${(pipelines.filter(p => p.status === 'paused').length / pipelines.length) * 100}%` }}
              ></div>
              <div 
                className="h-2 bg-red-500 rounded"
                style={{ width: `${(pipelines.filter(p => p.status === 'error').length / pipelines.length) * 100}%` }}
              ></div>
              <div 
                className="h-2 bg-blue-500 rounded"
                style={{ width: `${(pipelines.filter(p => p.status === 'draft').length / pipelines.length) * 100}%` }}
              ></div>
            </div>
          </div>
        </div>
        
        {/* Alert Section */}
        {pipelines.some(p => p.status === 'error') && (
          <div className="mt-6 p-4 bg-red-500/20 rounded-lg border border-red-500/30">
            <div className="flex items-start space-x-3">
              <AlertTriangle className="w-6 h-6 text-red-400 mt-0.5" />
              <div>
                <h3 className="text-red-400 font-medium">Pipeline Alerts</h3>
                <div className="mt-2 space-y-2">
                  {pipelines.filter(p => p.status === 'error').map(pipeline => (
                    <div key={pipeline.id} className="flex items-start space-x-2">
                      <div className="w-1.5 h-1.5 rounded-full bg-red-400 mt-1.5"></div>
                      <div>
                        <p className="text-red-300 font-medium">{pipeline.name}</p>
                        <p className="text-red-300/70 text-sm">{pipeline.error.component}: {pipeline.error.message}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Data Pipeline Modal */}
      <DataPipelineModal
        isOpen={isPipelineModalOpen}
        onClose={() => {
          setIsPipelineModalOpen(false);
          setSelectedPipeline(null);
        }}
        pipeline={selectedPipeline ? pipelines.find(p => p.id === selectedPipeline) : null}
        components={components}
        onPipelineSaved={handlePipelineSaved}
      />

      {/* Data Component Modal */}
      <DataComponentModal
        isOpen={isComponentModalOpen}
        onClose={() => {
          setIsComponentModalOpen(false);
          setSelectedComponent(null);
        }}
        component={selectedComponent}
        onComponentSaved={handleComponentSaved}
      />

      {/* Advanced Filter Modal */}
      <AdvancedFilterModal
        isOpen={isFilterModalOpen}
        onClose={() => setIsFilterModalOpen(false)}
        onApplyFilters={applyAdvancedFilters}
        initialFilters={activeFilters}
        filterOptions={filterOptions}
        entityType="pipeline"
      />
    </div>
  );
};

export default DataEngineeringInterface;