import React, { useState, useEffect } from 'react';
import { 
  X, 
  Zap, 
  Save, 
  AlertCircle,
  Check,
  Code,
  RefreshCw,
  FileJson,
  Upload
} from 'lucide-react';

const MCPCapabilityModal = ({ isOpen, onClose, server, capability, onCapabilitySaved }) => {
  const [formData, setFormData] = useState({
    name: '',
    type: 'text',
    description: '',
    schema: '',
    parameters: '',
    version: '1.0.0'
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [validatingSchema, setValidatingSchema] = useState(false);
  const [schemaValid, setSchemaValid] = useState(null); // true, false, or null

  // Available capability types
  const capabilityTypes = [
    { id: 'text', name: 'Text Processing' },
    { id: 'vision', name: 'Vision Analysis' },
    { id: 'embedding', name: 'Embedding Generation' },
    { id: 'data-extraction', name: 'Data Extraction' },
    { id: 'analysis', name: 'Data Analysis' },
    { id: 'automation', name: 'Automation' },
    { id: 'security', name: 'Security' },
    { id: 'agent', name: 'Agent' },
    { id: 'image-generation', name: 'Image Generation' }
  ];

  // Initialize form data when capability prop changes
  useEffect(() => {
    if (isOpen && capability) {
      // Editing existing capability
      setFormData({
        name: capability.name || '',
        type: capability.type || 'text',
        description: capability.description || '',
        schema: capability.schema || '',
        parameters: capability.parameters || '',
        version: capability.version || '1.0.0'
      });
    } else if (isOpen && server) {
      // Creating new capability
      setFormData({
        name: '',
        type: 'text',
        description: '',
        schema: '',
        parameters: '',
        version: '1.0.0'
      });
    }
    
    // Reset errors and status
    setErrors({});
    setSubmitStatus(null);
    setSchemaValid(null);
  }, [isOpen, capability, server]);

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

    // Reset schema validation if schema changes
    if (name === 'schema') {
      setSchemaValid(null);
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Capability name is required';
    }
    
    if (!formData.type) {
      newErrors.type = 'Capability type is required';
    }
    
    if (formData.schema.trim() && schemaValid === false) {
      newErrors.schema = 'Schema is invalid JSON';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const validateSchema = () => {
    if (!formData.schema.trim()) {
      setSchemaValid(null);
      return;
    }
    
    setValidatingSchema(true);
    
    try {
      // Try to parse the schema as JSON
      JSON.parse(formData.schema);
      setSchemaValid(true);
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.schema;
        return newErrors;
      });
    } catch (error) {
      setSchemaValid(false);
      setErrors(prev => ({
        ...prev,
        schema: 'Invalid JSON: ' + error.message
      }));
    } finally {
      setValidatingSchema(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    // Validate schema first if it's not empty and hasn't been validated yet
    if (formData.schema.trim() && schemaValid === null) {
      validateSchema();
      if (schemaValid === false) {
        return;
      }
    }

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    setSubmitStatus(null);
    
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate a successful API call
      console.log('Starting API simulation...');
      await new Promise(resolve => setTimeout(resolve, 500)); // Reduced delay for testing

      // Prepare the capability data for the API
      const capabilityData = {
        ...formData,
        id: capability ? capability.id : `${formData.type}-${Date.now()}`, // Use existing ID or generate a new one
        status: 'active',
        serverId: server.id,
        serverName: server.name,
        serverStatus: server.status,
        serverType: server.type
      };

      // Simulate API call
      // const response = await fetch(capability ? `/api/v1/mcp/capabilities/${capability.id}` : '/api/v1/mcp/capabilities', {
      //   method: capability ? 'PUT' : 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(capabilityData),
      // });

      // if (!response.ok) {
      //   throw new Error('Failed to save capability');
      // }

      // const data = await response.json();

      console.log('Setting success status...');
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onCapabilitySaved) {
        onCapabilitySaved(capabilityData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving capability:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSchemaUpload = (e) => {
    const file = e.target.files[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = (event) => {
      try {
        const content = event.target.result;
        // Try to parse it to validate
        JSON.parse(content);
        // If it's valid, update the form
        setFormData(prev => ({ ...prev, schema: content }));
        setSchemaValid(true);
        setErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors.schema;
          return newErrors;
        });
      } catch (error) {
        setSchemaValid(false);
        setErrors(prev => ({
          ...prev,
          schema: 'Invalid JSON file: ' + error.message
        }));
      }
    };
    reader.readAsText(file);
  };

  if (!isOpen || !server) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div
        data-testid="mcp-capability-modal"
        className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-3xl max-h-[90vh] overflow-hidden"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <Zap className="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">
                {capability ? 'Edit Capability' : 'Add Capability'}
              </h2>
              <p className="text-slate-400 text-sm">Server: {server.name}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Basic Information</h3>
              
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                  Capability Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter capability name (e.g., GPT-4 Vision)"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-slate-300 mb-2">
                  Capability Type
                </label>
                <select
                  id="type"
                  name="type"
                  value={formData.type}
                  onChange={handleChange}
                  className={`w-full bg-slate-700/50 border ${errors.type ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500`}
                >
                  {capabilityTypes.map((type) => (
                    <option key={type.id} value={type.id}>{type.name}</option>
                  ))}
                </select>
                {errors.type && (
                  <p className="mt-1 text-sm text-red-400">{errors.type}</p>
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
                  placeholder="Enter a description of this capability"
                  rows="3"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                ></textarea>
              </div>
              
              <div>
                <label htmlFor="version" className="block text-sm font-medium text-slate-300 mb-2">
                  Version
                </label>
                <input
                  type="text"
                  id="version"
                  name="version"
                  value={formData.version}
                  onChange={handleChange}
                  placeholder="Enter capability version (e.g., 1.0.0)"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
              </div>
            </div>
            
            {/* Schema and Parameters */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Schema and Parameters</h3>
              
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label htmlFor="schema" className="block text-sm font-medium text-slate-300">
                    JSON Schema
                  </label>
                  <div className="flex items-center space-x-2">
                    <button
                      type="button"
                      onClick={validateSchema}
                      disabled={!formData.schema.trim() || validatingSchema}
                      className="text-xs text-purple-400 hover:text-purple-300 disabled:text-slate-500 flex items-center space-x-1"
                    >
                      {validatingSchema ? (
                        <RefreshCw className="w-3 h-3 animate-spin" />
                      ) : (
                        <Code className="w-3 h-3" />
                      )}
                      <span>Validate</span>
                    </button>
                    <label className="text-xs text-purple-400 hover:text-purple-300 cursor-pointer flex items-center space-x-1">
                      <Upload className="w-3 h-3" />
                      <span>Upload</span>
                      <input
                        type="file"
                        accept=".json"
                        className="hidden"
                        onChange={handleSchemaUpload}
                      />
                    </label>
                  </div>
                </div>
                <textarea
                  id="schema"
                  name="schema"
                  value={formData.schema}
                  onChange={handleChange}
                  placeholder="Enter JSON schema for this capability (optional)"
                  rows="6"
                  className={`w-full bg-slate-700/50 border ${
                    errors.schema 
                      ? 'border-red-500/50' 
                      : schemaValid === true 
                        ? 'border-green-500/50' 
                        : 'border-slate-600/50'
                  } rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono text-sm`}
                ></textarea>
                {errors.schema && (
                  <p className="mt-1 text-sm text-red-400">{errors.schema}</p>
                )}
                {schemaValid === true && !errors.schema && (
                  <p className="mt-1 text-sm text-green-400 flex items-center space-x-1">
                    <Check className="w-3 h-3" />
                    <span>Schema is valid JSON</span>
                  </p>
                )}
                <p className="mt-1 text-xs text-slate-400">Define the JSON schema for this capability's input/output</p>
              </div>
              
              <div>
                <label htmlFor="parameters" className="block text-sm font-medium text-slate-300 mb-2">
                  Parameters (JSON)
                </label>
                <textarea
                  id="parameters"
                  name="parameters"
                  value={formData.parameters}
                  onChange={handleChange}
                  placeholder="Enter parameters as JSON (optional)"
                  rows="4"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono text-sm"
                ></textarea>
                <p className="mt-1 text-xs text-slate-400">Define default parameters for this capability</p>
              </div>
            </div>
            
            {/* Integration with ADK */}
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
              <div className="flex items-start space-x-3">
                <FileJson className="w-5 h-5 text-purple-400 mt-0.5" />
                <div>
                  <h4 className="text-white font-medium">Google ADK Integration</h4>
                  <p className="text-slate-300 text-sm mt-1">This capability will be registered with the Google Agent Development Kit for use with agent orchestration.</p>
                  <div className="mt-2 text-xs text-slate-400">
                    <p>• Capabilities are exposed as tools to ADK agents</p>
                    <p>• JSON schema is used to validate inputs and outputs</p>
                    <p>• Parameters define default behavior</p>
                  </div>
                </div>
              </div>
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
                  <span>Capability saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save capability</span>
                </div>
              )}
              
              <button
                type="submit"
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
                    <span>Save Capability</span>
                  </>
                )}
              </button>
            </div>
          </div>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default MCPCapabilityModal;