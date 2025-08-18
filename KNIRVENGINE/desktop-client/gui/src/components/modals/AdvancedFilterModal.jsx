import React, { useState, useEffect } from 'react';
import { 
  X, 
  Filter, 
  Plus,
  Trash2,
  Save,
  Check,
  AlertCircle,
  RefreshCw
} from 'lucide-react';

const AdvancedFilterModal = ({ isOpen, onClose, onApplyFilters, initialFilters = [], filterOptions, entityType }) => {
  const [filters, setFilters] = useState([]);
  const [presets, setPresets] = useState([]);
  const [presetName, setPresetName] = useState('');
  const [savingPreset, setSavingPreset] = useState(false);
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null

  // Initialize filters when modal opens or initialFilters change
  useEffect(() => {
    if (isOpen) {
      if (initialFilters && initialFilters.length > 0) {
        setFilters(initialFilters);
      } else {
        // Start with one empty filter
        setFilters([{ field: '', operator: 'equals', value: '', id: Date.now() }]);
      }
      
      // Load presets (in a real implementation, this would be from API/localStorage)
      loadPresets();
    }
  }, [isOpen, initialFilters]);

  // Load saved filter presets
  const loadPresets = () => {
    // In a real implementation, this would be an API call or localStorage
    const savedPresets = localStorage.getItem(`${entityType}-filter-presets`);
    if (savedPresets) {
      try {
        setPresets(JSON.parse(savedPresets));
      } catch (error) {
        console.error('Error loading presets:', error);
        setPresets([]);
      }
    } else {
      // Sample presets for demonstration
      const samplePresets = [
        { 
          id: 1, 
          name: 'Active Items', 
          filters: [{ field: 'status', operator: 'equals', value: 'active', id: 101 }] 
        },
        { 
          id: 2, 
          name: 'Recent Items', 
          filters: [{ field: 'createdAt', operator: 'greater_than', value: '30 days ago', id: 102 }] 
        }
      ];
      setPresets(samplePresets);
    }
  };

  // Save current filters as a preset
  const saveAsPreset = () => {
    if (!presetName.trim()) {
      setErrors(prev => ({ ...prev, presetName: 'Preset name is required' }));
      return;
    }

    if (filters.length === 0) {
      setErrors(prev => ({ ...prev, filters: 'At least one filter is required' }));
      return;
    }

    setSavingPreset(true);

    // In a real implementation, this would be an API call
    setTimeout(() => {
      const newPreset = {
        id: Date.now(),
        name: presetName,
        filters: [...filters]
      };

      const updatedPresets = [...presets, newPreset];
      setPresets(updatedPresets);

      // Save to localStorage for persistence
      localStorage.setItem(`${entityType}-filter-presets`, JSON.stringify(updatedPresets));

      // Reset form
      setPresetName('');
      setErrors({});
      setSavingPreset(false);
    }, 500);
  };

  // Load a saved preset
  const loadPreset = (preset) => {
    setFilters([...preset.filters]);
  };

  // Delete a saved preset
  const deletePreset = (presetId) => {
    const updatedPresets = presets.filter(preset => preset.id !== presetId);
    setPresets(updatedPresets);
    
    // Save to localStorage
    localStorage.setItem(`${entityType}-filter-presets`, JSON.stringify(updatedPresets));
  };

  // Add a new filter
  const addFilter = () => {
    setFilters([...filters, { field: '', operator: 'equals', value: '', id: Date.now() }]);
  };

  // Remove a filter
  const removeFilter = (id) => {
    setFilters(filters.filter(filter => filter.id !== id));
  };

  // Update a filter
  const updateFilter = (id, field, value) => {
    setFilters(filters.map(filter => 
      filter.id === id ? { ...filter, [field]: value } : filter
    ));
  };

  // Apply filters
  const applyFilters = () => {
    // Validate filters
    const validFilters = filters.filter(filter => 
      filter.field && filter.operator && filter.value
    );

    if (validFilters.length === 0) {
      setErrors({ filters: 'At least one complete filter is required' });
      return;
    }

    setIsSubmitting(true);
    setSubmitStatus(null);

    // In a real implementation, this would be an API call
    setTimeout(() => {
      if (onApplyFilters) {
        onApplyFilters(validFilters);
      }
      
      setSubmitStatus('success');
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1000);
    }, 500);
  };

  // Reset filters
  const resetFilters = () => {
    setFilters([{ field: '', operator: 'equals', value: '', id: Date.now() }]);
    setErrors({});
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
              <Filter className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-xl font-bold text-white">Advanced Filters</h2>
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
          <div className="space-y-6">
            {/* Filter Builder */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Build Filters</h3>
              
              {filters.map((filter, index) => (
                <div key={filter.id} className="flex items-center space-x-2">
                  <div className="flex-1 grid grid-cols-3 gap-2">
                    <select
                      value={filter.field}
                      onChange={(e) => updateFilter(filter.id, 'field', e.target.value)}
                      className="bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="">Select Field</option>
                      {filterOptions.fields.map(field => (
                        <option key={field.id} value={field.id}>{field.name}</option>
                      ))}
                    </select>
                    
                    <select
                      value={filter.operator}
                      onChange={(e) => updateFilter(filter.id, 'operator', e.target.value)}
                      className="bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      {filterOptions.operators.map(operator => (
                        <option key={operator.id} value={operator.id}>{operator.name}</option>
                      ))}
                    </select>
                    
                    <input
                      type="text"
                      value={filter.value}
                      onChange={(e) => updateFilter(filter.id, 'value', e.target.value)}
                      placeholder="Value"
                      className="bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                    />
                  </div>
                  
                  <button
                    onClick={() => removeFilter(filter.id)}
                    className="p-2 text-slate-400 hover:text-red-400 transition-colors duration-200"
                    disabled={filters.length === 1}
                  >
                    <Trash2 className="w-5 h-5" />
                  </button>
                </div>
              ))}
              
              {errors.filters && (
                <p className="text-sm text-red-400">{errors.filters}</p>
              )}
              
              <div className="flex items-center space-x-2">
                <button
                  onClick={addFilter}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 flex items-center space-x-2"
                >
                  <Plus className="w-4 h-4" />
                  <span>Add Filter</span>
                </button>
                
                <button
                  onClick={resetFilters}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
                >
                  Reset
                </button>
              </div>
            </div>
            
            {/* Saved Presets */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Saved Presets</h3>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {presets.map(preset => (
                  <div 
                    key={preset.id}
                    className="p-3 bg-slate-700/30 rounded-lg border border-slate-600/30 hover:bg-slate-700/50 transition-colors duration-200"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-white font-medium">{preset.name}</span>
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => loadPreset(preset)}
                          className="text-purple-400 hover:text-purple-300 text-sm"
                        >
                          Load
                        </button>
                        <button
                          onClick={() => deletePreset(preset.id)}
                          className="text-red-400 hover:text-red-300 text-sm"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                    <div className="text-xs text-slate-400">
                      {preset.filters.map((filter, index) => {
                        const fieldName = filterOptions.fields.find(f => f.id === filter.field)?.name || filter.field;
                        const operatorName = filterOptions.operators.find(o => o.id === filter.operator)?.name || filter.operator;
                        
                        return (
                          <div key={filter.id} className="mb-1">
                            {fieldName} {operatorName} {filter.value}
                            {index < preset.filters.length - 1 && ' AND '}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
              
              {presets.length === 0 && (
                <div className="text-center py-4 text-slate-400">
                  No saved presets. Create and save filters to reuse them later.
                </div>
              )}
              
              <div className="flex items-end space-x-2">
                <div className="flex-1">
                  <label htmlFor="presetName" className="block text-sm font-medium text-slate-300 mb-2">
                    Preset Name
                  </label>
                  <input
                    type="text"
                    id="presetName"
                    value={presetName}
                    onChange={(e) => {
                      setPresetName(e.target.value);
                      if (errors.presetName) {
                        setErrors(prev => {
                          const newErrors = { ...prev };
                          delete newErrors.presetName;
                          return newErrors;
                        });
                      }
                    }}
                    placeholder="Enter a name for this filter preset"
                    className={`w-full bg-slate-700/50 border ${errors.presetName ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                  />
                  {errors.presetName && (
                    <p className="mt-1 text-sm text-red-400">{errors.presetName}</p>
                  )}
                </div>
                
                <button
                  onClick={saveAsPreset}
                  disabled={savingPreset}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 flex items-center space-x-2 h-10"
                >
                  {savingPreset ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Saving...</span>
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      <span>Save Preset</span>
                    </>
                  )}
                </button>
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
                  <span>Filters applied!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to apply filters</span>
                </div>
              )}
              
              <button
                onClick={applyFilters}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Applying...</span>
                  </>
                ) : (
                  <>
                    <Filter className="w-4 h-4" />
                    <span>Apply Filters</span>
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

export default AdvancedFilterModal;