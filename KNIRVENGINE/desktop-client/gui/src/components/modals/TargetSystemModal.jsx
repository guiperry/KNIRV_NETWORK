import React, { useState, useEffect } from 'react';
import { 
  X, 
  Target, 
  Save, 
  AlertCircle,
  Check,
  Globe,
  Folder,
  Terminal,
  Database,
  Smartphone,
  Server,
  Wifi,
  Code,
  Shield
} from 'lucide-react';

const TargetSystemModal = ({ isOpen, onClose, target, onTargetSaved }) => {
  // Map target types to icons
  const getIconForTargetType = (type) => {
    const iconMap = {
      browser: Globe,
      filesystem: Folder,
      terminal: Terminal,
      database: Database,
      mobile: Smartphone,
      server: Server,
      network: Wifi,
      api: Code,
      security: Shield
    };
    return iconMap[type] || Server; // Default to Server icon
  };

  // Map target types to categories
  const getCategoryForTargetType = (type) => {
    const categoryMap = {
      browser: 'Web Browser',
      filesystem: 'File System',
      terminal: 'Terminal',
      database: 'Database',
      mobile: 'Mobile Device',
      server: 'Server',
      network: 'Network',
      api: 'API Service',
      security: 'Security System'
    };
    return categoryMap[type] || 'Other'; // Default to Other
  };

  const [formData, setFormData] = useState({
    name: '',
    type: 'browser',
    description: '',
    capabilities: [],
    permissions: [],
    security: 'medium',
    connectionMethod: '',
    dataAccess: []
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [testingConnection, setTestingConnection] = useState(false);
  const [testResult, setTestResult] = useState(null); // 'success', 'error', or null

  // Available target types
  const targetTypes = [
    { id: 'browser', name: 'Web Browser', icon: Globe, description: 'Web browser for accessing and interacting with web content' },
    { id: 'filesystem', name: 'File System', icon: Folder, description: 'Access to local files and directories' },
    { id: 'application', name: 'Application', icon: Code, description: 'Integration with desktop or web applications' },
    { id: 'system', name: 'System', icon: Terminal, description: 'System-level access and operations' },
    { id: 'network', name: 'Network', icon: Wifi, description: 'Network monitoring and analysis' },
    { id: 'database', name: 'Database', icon: Database, description: 'Database connections and operations' },
    { id: 'mobile', name: 'Mobile Device', icon: Smartphone, description: 'Mobile device integration' },
    { id: 'server', name: 'Server', icon: Server, description: 'Remote server access and management' }
  ];

  // Available capabilities by target type
  const capabilitiesByType = {
    browser: ['Web Scraping', 'DOM Analysis', 'Screenshot Capture', 'Form Automation', 'Navigation', 'Cookie Management'],
    filesystem: ['File Analysis', 'Directory Scanning', 'Content Indexing', 'Metadata Extraction', 'File Creation', 'File Modification'],
    application: ['UI Interaction', 'Data Extraction', 'Workflow Automation', 'Event Monitoring', 'API Integration'],
    system: ['Process Monitoring', 'Resource Tracking', 'Command Execution', 'Service Management', 'Log Analysis'],
    network: ['Traffic Analysis', 'Connection Monitoring', 'Bandwidth Tracking', 'Security Scanning', 'Protocol Analysis'],
    database: ['Query Execution', 'Schema Analysis', 'Data Mining', 'Performance Monitoring', 'Backup Management'],
    mobile: ['App Monitoring', 'Notification Analysis', 'Usage Tracking', 'Photo Analysis', 'Location Services'],
    server: ['Resource Monitoring', 'Service Management', 'Log Analysis', 'Security Auditing', 'Backup Management']
  };

  // Available permissions
  const availablePermissions = ['read', 'write', 'execute', 'delete', 'admin'];

  // Available security levels
  const securityLevels = [
    { id: 'low', name: 'Low', description: 'Minimal security restrictions, suitable for non-sensitive environments' },
    { id: 'medium', name: 'Medium', description: 'Balanced security with reasonable restrictions, suitable for most use cases' },
    { id: 'high', name: 'High', description: 'Strict security controls, suitable for sensitive data and operations' }
  ];

  // Initialize form data when target prop changes
  useEffect(() => {
    if (isOpen && target) {
      // Editing existing target
      setFormData({
        name: target.name || '',
        type: target.type || 'browser',
        description: target.description || '',
        capabilities: target.capabilities || [],
        permissions: target.permissions || [],
        security: target.security || 'medium',
        connectionMethod: target.connectionMethod || '',
        dataAccess: target.dataAccess || []
      });
    } else if (isOpen) {
      // Creating new target
      setFormData({
        name: '',
        type: 'browser',
        description: '',
        capabilities: [],
        permissions: ['read'],
        security: 'medium',
        connectionMethod: '',
        dataAccess: []
      });
    }
    
    // Reset errors and status
    setErrors({});
    setSubmitStatus(null);
    setTestResult(null);
  }, [isOpen, target]);

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

    // If type changes, reset capabilities to avoid incompatible selections
    if (name === 'type') {
      setFormData(prev => ({ ...prev, capabilities: [] }));
    }
  };

  const handleCapabilityToggle = (capability) => {
    setFormData(prev => {
      const newCapabilities = prev.capabilities.includes(capability)
        ? prev.capabilities.filter(cap => cap !== capability)
        : [...prev.capabilities, capability];
      
      return { ...prev, capabilities: newCapabilities };
    });
    
    // Clear error for capabilities if it exists
    if (errors.capabilities) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.capabilities;
        return newErrors;
      });
    }
  };

  const handlePermissionToggle = (permission) => {
    setFormData(prev => {
      const newPermissions = prev.permissions.includes(permission)
        ? prev.permissions.filter(perm => perm !== permission)
        : [...prev.permissions, permission];
      
      return { ...prev, permissions: newPermissions };
    });
    
    // Clear error for permissions if it exists
    if (errors.permissions) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.permissions;
        return newErrors;
      });
    }
  };

  const handleDataAccessChange = (e) => {
    const { value } = e.target;
    const dataAccessArray = value.split(',').map(item => item.trim()).filter(item => item !== '');
    setFormData(prev => ({ ...prev, dataAccess: dataAccessArray }));
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Target name is required';
    }
    
    if (!formData.type) {
      newErrors.type = 'Target type is required';
    }
    
    if (formData.capabilities.length === 0) {
      newErrors.capabilities = 'Select at least one capability';
    }
    
    if (formData.permissions.length === 0) {
      newErrors.permissions = 'Select at least one permission';
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
      
      // Prepare the target data for the API
      const targetData = {
        ...formData,
        status: 'connected', // Default status for new targets
        lastActivity: new Date().toISOString(),
        activeAgents: 0,
        id: target ? target.id : Date.now().toString(), // Use existing ID or generate a new one
        icon: getIconForTargetType(formData.type), // Add the appropriate icon
        category: getCategoryForTargetType(formData.type) // Add the category
      };
      
      // Simulate API call
      // const response = await fetch(target ? `/api/v1/targets/${target.id}` : '/api/v1/targets', {
      //   method: target ? 'PUT' : 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(targetData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to save target');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onTargetSaved) {
        onTargetSaved(targetData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving target:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTestConnection = async () => {
    if (!formData.name.trim() || !formData.type) {
      setErrors(prev => ({
        ...prev,
        name: !formData.name.trim() ? 'Target name is required' : prev.name,
        type: !formData.type ? 'Target type is required' : prev.type
      }));
      return;
    }
    
    setTestingConnection(true);
    setTestResult(null);
    
    try {
      // In a real implementation, this would be an API call to test the connection
      // For now, we'll simulate a successful connection test
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Simulate API call
      // const response = await fetch('/api/v1/targets/test-connection', {
      //   method: 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(formData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Connection test failed');
      // }
      
      // const data = await response.json();
      
      // 80% chance of success for simulation
      const success = Math.random() < 0.8;
      
      if (success) {
        setTestResult('success');
      } else {
        throw new Error('Connection test failed');
      }
    } catch (error) {
      console.error('Error testing connection:', error);
      setTestResult('error');
    } finally {
      setTestingConnection(false);
    }
  };

  const getTargetTypeIcon = (typeId) => {
    const targetType = targetTypes.find(type => type.id === typeId);
    return targetType ? targetType.icon : Server;
  };

  if (!isOpen) return null;

  const TargetTypeIcon = getTargetTypeIcon(formData.type);

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
            <div className="p-2 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 rounded-lg border border-blue-500/30">
              <Target className="w-5 h-5 text-blue-400" />
            </div>
            <h2 className="text-xl font-bold text-white">
              {target ? 'Edit Target System' : 'Add Target System'}
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
                  Target Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter target system name (e.g., Chrome Browser)"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-slate-300 mb-2">
                  Target Type
                </label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  {targetTypes.map((type) => {
                    const TypeIcon = type.icon;
                    return (
                      <div
                        key={type.id}
                        onClick={() => handleChange({ target: { name: 'type', value: type.id } })}
                        className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                          formData.type === type.id
                            ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                            : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                        }`}
                      >
                        <div className="flex flex-col items-center text-center">
                          <TypeIcon className={`w-6 h-6 mb-2 ${formData.type === type.id ? 'text-blue-400' : 'text-slate-400'}`} />
                          <span className={`text-sm font-medium ${formData.type === type.id ? 'text-white' : 'text-slate-300'}`}>
                            {type.name}
                          </span>
                        </div>
                      </div>
                    );
                  })}
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
                  placeholder="Enter a description of this target system"
                  rows="3"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                ></textarea>
              </div>
            </div>
            
            {/* Capabilities */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Capabilities</h3>
              <p className="text-slate-400 text-sm">Select the capabilities this target system supports:</p>
              
              <div className="grid grid-cols-2 gap-3">
                {capabilitiesByType[formData.type]?.map((capability) => (
                  <div 
                    key={capability}
                    onClick={() => handleCapabilityToggle(capability)}
                    className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                      formData.capabilities.includes(capability)
                        ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                        : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-white font-medium">{capability}</span>
                      {formData.capabilities.includes(capability) && (
                        <Check className="w-4 h-4 text-blue-400" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
              {errors.capabilities && (
                <p className="text-sm text-red-400">{errors.capabilities}</p>
              )}
            </div>
            
            {/* Permissions */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Permissions</h3>
              <p className="text-slate-400 text-sm">Select the permissions to grant to agents:</p>
              
              <div className="flex flex-wrap gap-3">
                {availablePermissions.map((permission) => (
                  <div 
                    key={permission}
                    onClick={() => handlePermissionToggle(permission)}
                    className={`px-4 py-2 rounded-lg cursor-pointer transition-all duration-200 border ${
                      formData.permissions.includes(permission)
                        ? 'bg-gradient-to-r from-green-500/20 to-emerald-500/20 border-green-500/30'
                        : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                    }`}
                  >
                    <div className="flex items-center space-x-2">
                      <span className={`text-sm font-medium ${formData.permissions.includes(permission) ? 'text-green-400' : 'text-slate-300'}`}>
                        {permission}
                      </span>
                      {formData.permissions.includes(permission) && (
                        <Check className="w-4 h-4 text-green-400" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
              {errors.permissions && (
                <p className="text-sm text-red-400">{errors.permissions}</p>
              )}
            </div>
            
            {/* Security Level */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Security Level</h3>
              <p className="text-slate-400 text-sm">Select the security level for this target system:</p>
              
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                {securityLevels.map((level) => (
                  <div 
                    key={level.id}
                    onClick={() => handleChange({ target: { name: 'security', value: level.id } })}
                    className={`p-4 rounded-lg cursor-pointer transition-all duration-200 border ${
                      formData.security === level.id
                        ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                        : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                    }`}
                  >
                    <div className="flex flex-col items-center text-center">
                      <Shield className={`w-6 h-6 mb-2 ${
                        formData.security === level.id 
                          ? 'text-blue-400' 
                          : level.id === 'high' 
                            ? 'text-green-400' 
                            : level.id === 'medium' 
                              ? 'text-yellow-400' 
                              : 'text-red-400'
                      }`} />
                      <span className={`text-sm font-medium mb-1 ${formData.security === level.id ? 'text-white' : 'text-slate-300'}`}>
                        {level.name}
                      </span>
                      <span className="text-xs text-slate-400">
                        {level.description}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            
            {/* Connection Details */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Connection Details</h3>
              
              <div>
                <label htmlFor="connectionMethod" className="block text-sm font-medium text-slate-300 mb-2">
                  Connection Method
                </label>
                <input
                  type="text"
                  id="connectionMethod"
                  name="connectionMethod"
                  value={formData.connectionMethod}
                  onChange={handleChange}
                  placeholder={`Enter connection method (e.g., ${
                    formData.type === 'browser' ? 'Chrome Extension API' :
                    formData.type === 'filesystem' ? 'File System API' :
                    formData.type === 'application' ? 'Application API' :
                    formData.type === 'system' ? 'System API' :
                    formData.type === 'network' ? 'Network API' :
                    formData.type === 'database' ? 'Database Driver' :
                    formData.type === 'mobile' ? 'Mobile App Bridge' :
                    'API Connection'
                  })`}
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              
              <div>
                <label htmlFor="dataAccess" className="block text-sm font-medium text-slate-300 mb-2">
                  Data Access (comma-separated)
                </label>
                <textarea
                  id="dataAccess"
                  name="dataAccess"
                  value={formData.dataAccess.join(', ')}
                  onChange={handleDataAccessChange}
                  placeholder={`Enter data access paths (e.g., ${
                    formData.type === 'browser' ? 'Browsing History, Bookmarks, Active Tabs' :
                    formData.type === 'filesystem' ? 'Documents, Downloads, Desktop' :
                    formData.type === 'application' ? 'Open Files, Workspace, Settings' :
                    formData.type === 'system' ? 'System Processes, Environment Variables' :
                    formData.type === 'network' ? 'Connection Status, Traffic Stats' :
                    formData.type === 'database' ? 'Public Schema, User Tables' :
                    formData.type === 'mobile' ? 'Photos, Contacts, Location' :
                    'Data Paths'
                  })`}
                  rows="2"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                ></textarea>
                <p className="mt-1 text-xs text-slate-400">Specify which data this target system can access</p>
              </div>
              
              <div>
                <button
                  type="button"
                  onClick={handleTestConnection}
                  disabled={testingConnection}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {testingConnection ? (
                    <>
                      <div className="w-4 h-4 border-2 border-slate-400 border-t-transparent rounded-full animate-spin"></div>
                      <span>Testing Connection...</span>
                    </>
                  ) : (
                    <>
                      <TargetTypeIcon className="w-4 h-4" />
                      <span>Test Connection</span>
                    </>
                  )}
                </button>
                
                {testResult === 'success' && (
                  <div className="mt-2 p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <Check className="w-5 h-5 text-green-400" />
                      <span className="text-green-400 font-medium">Connection successful!</span>
                    </div>
                    <p className="text-green-300/70 text-sm mt-1">Target system is accessible and ready for agent deployment.</p>
                  </div>
                )}
                
                {testResult === 'error' && (
                  <div className="mt-2 p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <AlertCircle className="w-5 h-5 text-red-400" />
                      <span className="text-red-400 font-medium">Connection failed!</span>
                    </div>
                    <p className="text-red-300/70 text-sm mt-1">Unable to connect to the target system. Please check your connection details and try again.</p>
                  </div>
                )}
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
                  <span>Target system saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save target system</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-blue-500 to-cyan-500 text-white rounded-lg font-medium hover:from-blue-600 hover:to-cyan-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Target System</span>
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

export default TargetSystemModal;