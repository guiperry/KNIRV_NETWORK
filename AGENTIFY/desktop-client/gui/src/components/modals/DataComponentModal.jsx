import React, { useState, useEffect } from 'react';
import { 
  X, 
  Layers, 
  Save, 
  AlertCircle,
  Check,
  Database,
  Cpu,
  Code,
  RefreshCw,
  FileJson,
  Upload
} from 'lucide-react';

const DataComponentModal = ({ isOpen, onClose, component, onComponentSaved }) => {
  const [formData, setFormData] = useState({
    name: '',
    type: 'processor',
    description: '',
    category: '',
    schema: '',
    configuration: ''
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [validatingSchema, setValidatingSchema] = useState(false);
  const [schemaValid, setSchemaValid] = useState(null); // true, false, or null

  // Available component types
  const componentTypes = [
    { id: 'source', name: 'Source', description: 'Data input component' },
    { id: 'processor', name: 'Processor', description: 'Data transformation component' },
    { id: 'sink', name: 'Sink', description: 'Data output component' }
  ];

  // Available categories by type
  const categoriesByType = {
    source: ['streaming', 'database', 'file', 'api', 'event'],
    processor: ['transform', 'filter', 'enrich', 'ml', 'aggregate'],
    sink: ['database', 'file', 'streaming', 'visualization', 'notification']
  };

  // Initialize form data when component prop changes
  useEffect(() => {
    if (isOpen && component) {
      // Editing existing component
      setFormData({
        name: component.name || '',
        type: component.type || 'processor',
        description: component.description || '',
        category: component.category || '',
        schema: component.schema || '',
        configuration: component.configuration || ''
      });
    } else if (isOpen) {
      // Creating new component
      setFormData({
        name: '',
        type: 'processor',
        description: '',
        category: '',
        schema: '',
        configuration: ''
      });
    }
    
    // Reset errors and status
    setErrors({});
    setSubmitStatus(null);
    setSchemaValid(null);
  }, [isOpen, component]);

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
    
    // Reset category if type changes
    if (name === 'type') {
      setFormData(prev => ({ ...prev, category: '' }));
    }
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Component name is required';
    }
    
    if (!formData.type) {
      newErrors.type = 'Component type is required';
    }
    
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    
    if (!formData.category) {
      newErrors.category = 'Category is required';
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
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // Prepare the component data for the API
      const componentData = {
        ...formData,
        id: component ? component.id : Date.now()
      };
      
      // Simulate API call
      // const response = await fetch(component ? `/api/v1/components/${component.id}` : '/api/v1/components', {
      //   method: component ? 'PUT' : 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(componentData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to save component');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onComponentSaved) {
        onComponentSaved(componentData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving component:', error);
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

  const getTypeIcon = (type) => {
    switch (type) {
      case 'source':
        return <Database className="w-5 h-5 text-blue-400" />;
      case 'processor':
        return <Cpu className="w-5 h-5 text-purple-400" />;
      case 'sink':
        return <Save className="w-5 h-5 text-green-400" />;
      default:
        return <Layers className="w-5 h-5 text-slate-400" />;
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
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-3xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <Layers className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-xl font-bold text-white">
              {component ? 'Edit Component' : 'Add Component'}
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
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Basic Information</h3>
              
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                  Component Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter component name"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-slate-300 mb-2">
                  Component Type
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {componentTypes.map((type) => (
                    <div
                      key={type.id}
                      onClick={() => handleChange({ target: { name: 'type', value: type.id } })}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.type === type.id
                          ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex flex-col items-center text-center">
                        {getTypeIcon(type.id)}
                        <span className={`text-sm font-medium mt-2 ${formData.type === type.id ? 'text-white' : 'text-slate-300'}`}>
                          {type.name}
                        </span>
                        <span className="text-xs text-slate-400 mt-1">{type.description}</span>
                      </div>
                    </div>
                  ))}
                </div>
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
                  placeholder="Enter component description"
                  rows="3"
                  className={`w-full bg-slate-700/50 border ${errors.description ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                ></textarea>
                {errors.description && (
                  <p className="mt-1 text-sm text-red-400">{errors.description}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="category" className="block text-sm font-medium text-slate-300 mb-2">
                  Category
                </label>
                <select
                  id="category"
                  name="category"
                  value={formData.category}
                  onChange={handleChange}
                  className={`w-full bg-slate-700/50 border ${errors.category ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500`}
                >
                  <option value="">Select category</option>
                  {formData.type && categoriesByType[formData.type]?.map(category => (
                    <option key={category} value={category}>
                      {category.charAt(0).toUpperCase() + category.slice(1)}
                    </option>
                  ))}
                </select>
                {errors.category && (
                  <p className="mt-1 text-sm text-red-400">{errors.category}</p>
                )}
              </div>
            </div>
            
            {/* Schema and Configuration */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Schema and Configuration</h3>
              
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
                  placeholder="Enter JSON schema for this component (optional)"
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
                <p className="mt-1 text-xs text-slate-400">Define the JSON schema for this component's input/output</p>
              </div>
              
              <div>
                <label htmlFor="configuration" className="block text-sm font-medium text-slate-300 mb-2">
                  Default Configuration (JSON)
                </label>
                <textarea
                  id="configuration"
                  name="configuration"
                  value={formData.configuration}
                  onChange={handleChange}
                  placeholder="Enter default configuration as JSON (optional)"
                  rows="4"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 font-mono text-sm"
                ></textarea>
                <p className="mt-1 text-xs text-slate-400">Define default configuration parameters for this component</p>
              </div>
            </div>
            
            {/* Integration Information */}
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
              <div className="flex items-start space-x-3">
                <FileJson className="w-5 h-5 text-purple-400 mt-0.5" />
                <div>
                  <h4 className="text-white font-medium">Component Integration</h4>
                  <p className="text-slate-300 text-sm mt-1">This component will be available for use in data pipelines.</p>
                  <div className="mt-2 text-xs text-slate-400">
                    <p>• Components can be connected to form data processing pipelines</p>
                    <p>• JSON schema defines the data structure for input/output</p>
                    <p>• Configuration parameters define default behavior</p>
                  </div>
                </div>
              </div>
            </div>
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
                  <span>Component saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save component</span>
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
                    <span>Save Component</span>
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

export default DataComponentModal;