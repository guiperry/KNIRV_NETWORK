import React, { useState } from 'react';
import styles from './CapabilityAttachmentForm.module.css';

const CapabilityAttachmentForm = ({ nft, capability, onAttach }) => {
  const [params, setParams] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Extract parameters from capability descriptor if available
  const parameterDefinitions = capability.descriptor?.parameters || [];

  const handleParamChange = (paramName, value) => {
    setParams({
      ...params,
      [paramName]: value
    });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    
    try {
      await onAttach(params);
      // Reset form
      setParams({});
    } catch (error) {
      console.error('Error attaching capability:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className={styles.attachmentForm}>
      <div className={styles.formHeader}>
        <h4>Attach &quot;{capability.name}&quot; to &quot;{nft.name}&quot;</h4>
      </div>
      
      <form onSubmit={handleSubmit}>
        {parameterDefinitions.length > 0 ? (
          <div className={styles.parametersSection}>
            <h5>Configuration Parameters</h5>
            
            {parameterDefinitions.map((param, index) => (
              <div key={index} className={styles.parameterField}>
                <label htmlFor={`param-${param.name}`}>{param.name}</label>
                
                {param.type === 'boolean' ? (
                  <select
                    id={`param-${param.name}`}
                    value={params[param.name] || ''}
                    onChange={(e) => handleParamChange(param.name, e.target.value === 'true')}
                  >
                    <option value="">Select</option>
                    <option value="true">Yes</option>
                    <option value="false">No</option>
                  </select>
                ) : param.type === 'number' ? (
                  <input
                    id={`param-${param.name}`}
                    type="number"
                    value={params[param.name] || ''}
                    onChange={(e) => handleParamChange(param.name, Number(e.target.value))}
                    placeholder={param.description || param.name}
                  />
                ) : (
                  <input
                    id={`param-${param.name}`}
                    type="text"
                    value={params[param.name] || ''}
                    onChange={(e) => handleParamChange(param.name, e.target.value)}
                    placeholder={param.description || param.name}
                  />
                )}
                
                {param.description && (
                  <div className={styles.paramDescription}>{param.description}</div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.noParameters}>
            This capability doesn&apos;t require any configuration parameters.
          </div>
        )}
        
        <div className={styles.formActions}>
          <button 
            type="submit" 
            className={styles.attachButton}
            disabled={isSubmitting}
          >
            {isSubmitting ? 'Attaching...' : 'Attach Capability'}
          </button>
        </div>
      </form>
    </div>
  );
};

export default CapabilityAttachmentForm;