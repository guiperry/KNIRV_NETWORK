import React, { useState, useEffect } from 'react';
import { 
  X, 
  GitBranch, 
  Save, 
  AlertCircle,
  Check,
  Plus,
  Trash2,
  ArrowRight,
  Database,
  Cpu,
  Clock,
  RefreshCw
} from 'lucide-react';

const DataPipelineModal = ({ isOpen, onClose, pipeline, components, onPipelineSaved }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    status: 'draft',
    components: [],
    connections: []
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [availableComponents, setAvailableComponents] = useState([]);
  const [selectedComponents, setSelectedComponents] = useState([]);
  const [activeTab, setActiveTab] = useState('info');

  // Initialize form data when pipeline prop changes
  useEffect(() => {
    if (isOpen) {
      if (pipeline) {
        // Editing existing pipeline
        setFormData({
          name: pipeline.name || '',
          description: pipeline.description || '',
          status: pipeline.status || 'draft',
          components: pipeline.components || [],
          connections: pipeline.connections || []
        });
        setSelectedComponents(pipeline.components || []);
      } else {
        // Creating new pipeline
        setFormData({
          name: '',
          description: '',
          status: 'draft',
          components: [],
          connections: []
        });
        setSelectedComponents([]);
      }
      
      // Reset errors and status
      setErrors({});
      setSubmitStatus(null);
      setActiveTab('info');
      
      // Set available components
      setAvailableComponents(components || []);
    }
  }, [isOpen, pipeline, components]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field if it exists
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
  };

  const handleAddComponent = (component) => {
    // Check if component is already added
    if (selectedComponents.some(c => c.id === component.id)) {
      return;
    }
    
    // Add component with a default position
    const newComponent = {
      ...component,
      position: { x: 100 + selectedComponents.length * 200, y: 150 }
    };
    
    setSelectedComponents(prev => [...prev, newComponent]);
    setFormData(prev => ({
      ...prev,
      components: [...prev.components, newComponent]
    }));
  };

  const handleRemoveComponent = (componentId) => {
    // Remove component
    setSelectedComponents(prev => prev.filter(c => c.id !== componentId));
    
    // Remove component from form data
    setFormData(prev => ({
      ...prev,
      components: prev.components.filter(c => c.id !== componentId),
      // Also remove any connections involving this component
      connections: prev.connections.filter(conn => 
        conn.source !== componentId && conn.target !== componentId
      )
    }));
  };

  const handleAddConnection = (sourceId, targetId) => {
    // Check if connection already exists
    if (formData.connections.some(c => c.source === sourceId && c.target === targetId)) {
      return;
    }
    
    // Add connection
    setFormData(prev => ({
      ...prev,
      connections: [...prev.connections, { source: sourceId, target: targetId }]
    }));
  };

  const handleRemoveConnection = (sourceId, targetId) => {
    // Remove connection
    setFormData(prev => ({
      ...prev,
      connections: prev.connections.filter(conn => 
        !(conn.source === sourceId && conn.target === targetId)
      )
    }));
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Pipeline name is required';
    }
    
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    
    if (formData.components.length === 0) {
      newErrors.components = 'At least one component is required';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    setIsSubmitting(true);
    setSubmitStatus(null);
    
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate a successful API call
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // Prepare the pipeline data for the API
      const pipelineData = {
        ...formData,
        id: pipeline ? pipeline.id : Date.now(),
        lastRun: pipeline ? pipeline.lastRun : 'Never',
        nextRun: formData.status === 'active' ? '5 minutes from now' : 'Not scheduled',
        createdBy: pipeline ? pipeline.createdBy : 'Data Team',
        createdAt: pipeline ? pipeline.createdAt : new Date().toISOString(),
        metrics: pipeline ? pipeline.metrics : {
          eventsProcessed: 0,
          processingTime: 'N/A',
          errorRate: 'N/A',
          throughput: 'N/A'
        }
      };
      
      // Simulate API call
      // const response = await fetch(pipeline ? `/api/v1/pipelines/${pipeline.id}` : '/api/v1/pipelines', {
      //   method: pipeline ? 'PUT' : 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(pipelineData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to save pipeline');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onPipelineSaved) {
        onPipelineSaved(pipelineData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving pipeline:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
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
        return <Clock className="w-4 h-4 text-slate-400" />;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-4xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <GitBranch className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-xl font-bold text-white">
              {pipeline ? 'Edit Data Pipeline' : 'Create Data Pipeline'}
            </h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          <button
            onClick={() => setActiveTab('info')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'info'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <span>Basic Information</span>
          </button>
          <button
            onClick={() => setActiveTab('components')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'components'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <span>Components</span>
          </button>
          <button
            onClick={() => setActiveTab('connections')}
            className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
              activeTab === 'connections'
                ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
            }`}
          >
            <span>Connections</span>
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-220px)]">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            {activeTab === 'info' && (
              <div className="space-y-4">
                <div>
                  <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                    Pipeline Name
                  </label>
                  <input
                    type="text"
                    id="name"
                    name="name"
                    value={formData.name}
                    onChange={handleChange}
                    placeholder="Enter pipeline name"
                    className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                  />
                  {errors.name && (
                    <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="description" className="block text-sm font-medium text-slate-300 mb-2">
                    Description
                  </label>
                  <textarea
                    id="description"
                    name="description"
                    value={formData.description}
                    onChange={handleChange}
                    placeholder="Enter pipeline description"
                    rows="3"
                    className={`w-full bg-slate-700/50 border ${errors.description ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                  ></textarea>
                  {errors.description && (
                    <p className="mt-1 text-sm text-red-400">{errors.description}</p>
                  )}
                </div>
                
                <div>
                  <label htmlFor="status" className="block text-sm font-medium text-slate-300 mb-2">
                    Status
                  </label>
                  <select
                    id="status"
                    name="status"
                    value={formData.status}
                    onChange={handleChange}
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    <option value="draft">Draft</option>
                    <option value="active">Active</option>
                    <option value="paused">Paused</option>
                  </select>
                </div>
                
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <div className="flex items-start space-x-3">
                    <Clock className="w-5 h-5 text-purple-400 mt-0.5" />
                    <div>
                      <h4 className="text-white font-medium">Scheduling</h4>
                      <p className="text-slate-300 text-sm mt-1">Pipelines are automatically scheduled based on their status:</p>
                      <ul className="mt-2 space-y-1 text-sm text-slate-400">
                        <li>• <span className="text-green-400">Active</span>: Runs every 5 minutes</li>
                        <li>• <span className="text-yellow-400">Paused</span>: Manually triggered only</li>
                        <li>• <span className="text-blue-400">Draft</span>: Not scheduled</li>
                      </ul>
                      <p className="text-slate-300 text-sm mt-2">Advanced scheduling options will be available in a future update.</p>
                    </div>
                  </div>
                </div>
              </div>
            )}
            
            {/* Components */}
            {activeTab === 'components' && (
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-white mb-4">Selected Components</h3>
                  
                  {selectedComponents.length === 0 ? (
                    <div className="text-center py-8 bg-slate-700/30 rounded-lg border border-slate-600/30">
                      <GitBranch className="w-12 h-12 text-slate-500 mx-auto mb-3" />
                      <p className="text-slate-400">No components selected</p>
                      <p className="text-slate-500 text-sm mt-1">Add components from the library below</p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {selectedComponents.map((component) => (
                        <div 
                          key={component.id}
                          className="p-3 bg-slate-700/30 rounded-lg border border-slate-600/30 flex items-center justify-between"
                        >
                          <div className="flex items-center space-x-3">
                            <div className={`p-2 rounded-lg ${
                              component.type === 'source' ? 'bg-blue-500/20 border border-blue-500/30' :
                              component.type === 'processor' ? 'bg-purple-500/20 border border-purple-500/30' :
                              'bg-green-500/20 border border-green-500/30'
                            }`}>
                              {getComponentTypeIcon(component.type)}
                            </div>
                            <div>
                              <p className="text-white font-medium">{component.name}</p>
                              <p className="text-slate-400 text-xs">{component.description}</p>
                            </div>
                          </div>
                          <button
                            type="button"
                            onClick={() => handleRemoveComponent(component.id)}
                            className="p-1 text-slate-400 hover:text-red-400 transition-colors duration-200"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  {errors.components && (
                    <p className="mt-2 text-sm text-red-400">{errors.components}</p>
                  )}
                </div>
                
                <div>
                  <h3 className="text-lg font-semibold text-white mb-4">Available Components</h3>
                  
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-[300px] overflow-y-auto pr-2">
                    {availableComponents.map((component) => (
                      <div 
                        key={component.id}
                        className={`p-3 bg-slate-700/30 rounded-lg border border-slate-600/30 flex items-center justify-between cursor-pointer hover:bg-slate-700/50 transition-colors duration-200 ${
                          selectedComponents.some(c => c.id === component.id) ? 'opacity-50' : ''
                        }`}
                        onClick={() => handleAddComponent(component)}
                      >
                        <div className="flex items-center space-x-3">
                          <div className={`p-2 rounded-lg ${
                            component.type === 'source' ? 'bg-blue-500/20 border border-blue-500/30' :
                            component.type === 'processor' ? 'bg-purple-500/20 border border-purple-500/30' :
                            'bg-green-500/20 border border-green-500/30'
                          }`}>
                            {getComponentTypeIcon(component.type)}
                          </div>
                          <div>
                            <p className="text-white font-medium">{component.name}</p>
                            <div className="flex items-center space-x-2 mt-1">
                              <span className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full capitalize">
                                {component.type}
                              </span>
                              <span className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full">
                                {component.category}
                              </span>
                            </div>
                          </div>
                        </div>
                        <button
                          type="button"
                          className={`p-1 ${
                            selectedComponents.some(c => c.id === component.id)
                              ? 'text-purple-400'
                              : 'text-slate-400 hover:text-purple-400'
                          } transition-colors duration-200`}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleAddComponent(component);
                          }}
                        >
                          <Plus className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
            
            {/* Connections */}
            {activeTab === 'connections' && (
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-white mb-4">Component Connections</h3>
                  
                  {selectedComponents.length < 2 ? (
                    <div className="text-center py-8 bg-slate-700/30 rounded-lg border border-slate-600/30">
                      <GitBranch className="w-12 h-12 text-slate-500 mx-auto mb-3" />
                      <p className="text-slate-400">At least two components are required to create connections</p>
                      <p className="text-slate-500 text-sm mt-1">Add more components in the Components tab</p>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                        <h4 className="text-white font-medium mb-3">Current Connections</h4>
                        
                        {formData.connections.length === 0 ? (
                          <p className="text-slate-400 text-center py-3">No connections defined yet</p>
                        ) : (
                          <div className="space-y-2">
                            {formData.connections.map((connection, index) => {
                              const sourceComponent = selectedComponents.find(c => c.id === connection.source);
                              const targetComponent = selectedComponents.find(c => c.id === connection.target);
                              
                              return (
                                <div 
                                  key={index}
                                  className="p-3 bg-slate-800/50 rounded-lg border border-slate-700/50 flex items-center justify-between"
                                >
                                  <div className="flex items-center space-x-2">
                                    <div className="flex items-center space-x-2">
                                      <span className="text-white">{sourceComponent?.name || 'Unknown'}</span>
                                      <ArrowRight className="w-4 h-4 text-purple-400" />
                                      <span className="text-white">{targetComponent?.name || 'Unknown'}</span>
                                    </div>
                                  </div>
                                  <button
                                    type="button"
                                    onClick={() => handleRemoveConnection(connection.source, connection.target)}
                                    className="p-1 text-slate-400 hover:text-red-400 transition-colors duration-200"
                                  >
                                    <Trash2 className="w-4 h-4" />
                                  </button>
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                      
                      <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                        <h4 className="text-white font-medium mb-3">Add New Connection</h4>
                        
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <div>
                            <label className="block text-sm font-medium text-slate-300 mb-2">
                              Source Component
                            </label>
                            <select
                              id="sourceComponent"
                              className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                              defaultValue=""
                            >
                              <option value="" disabled>Select source</option>
                              {selectedComponents
                                .filter(c => c.type === 'source' || c.type === 'processor')
                                .map(component => (
                                  <option key={component.id} value={component.id}>
                                    {component.name} ({component.type})
                                  </option>
                                ))
                              }
                            </select>
                          </div>
                          
                          <div className="flex items-center justify-center">
                            <ArrowRight className="w-6 h-6 text-purple-400" />
                          </div>
                          
                          <div>
                            <label className="block text-sm font-medium text-slate-300 mb-2">
                              Target Component
                            </label>
                            <select
                              id="targetComponent"
                              className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                              defaultValue=""
                            >
                              <option value="" disabled>Select target</option>
                              {selectedComponents
                                .filter(c => c.type === 'processor' || c.type === 'sink')
                                .map(component => (
                                  <option key={component.id} value={component.id}>
                                    {component.name} ({component.type})
                                  </option>
                                ))
                              }
                            </select>
                          </div>
                        </div>
                        
                        <div className="mt-4 flex justify-end">
                          <button
                            type="button"
                            onClick={() => {
                              const sourceSelect = document.getElementById('sourceComponent');
                              const targetSelect = document.getElementById('targetComponent');
                              
                              if (sourceSelect.value && targetSelect.value) {
                                handleAddConnection(
                                  parseInt(sourceSelect.value, 10),
                                  parseInt(targetSelect.value, 10)
                                );
                                
                                // Reset selects
                                sourceSelect.value = '';
                                targetSelect.value = '';
                              }
                            }}
                            className="px-4 py-2 bg-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30 transition-colors duration-200 flex items-center space-x-2"
                          >
                            <Plus className="w-4 h-4" />
                            <span>Add Connection</span>
                          </button>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
                
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <div className="flex items-start space-x-3">
                    <GitBranch className="w-5 h-5 text-purple-400 mt-0.5" />
                    <div>
                      <h4 className="text-white font-medium">Connection Rules</h4>
                      <ul className="mt-2 space-y-1 text-sm text-slate-400">
                        <li>• Source components can connect to processors or sinks</li>
                        <li>• Processor components can connect to other processors or sinks</li>
                        <li>• Sink components cannot have outgoing connections</li>
                        <li>• Circular connections are not allowed</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </form>
        </div>
        
        {/* Footer */}
        <div className="p-6 border-t border-slate-700 bg-slate-800/80 mt-auto">
          <div className="flex items-center justify-between">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
            >
              Cancel
            </button>
            
            <div className="flex items-center space-x-3">
              {submitStatus === 'success' && (
                <div className="flex items-center text-green-400 space-x-1">
                  <Check className="w-4 h-4" />
                  <span>Pipeline saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save pipeline</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Pipeline</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default DataPipelineModal;