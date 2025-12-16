import { useState } from 'react';
import React from 'react';
import { useBackend } from '../contexts/BackendContext';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './add-capability.module.css';

export default function AddCapability() {
  const { activePage } = useNavigation('add-capability');
  const { isRunning } = useBackend();
  const [formData, setFormData] = useState({
    capabilityType: '',
    desiredName: '',
    description: '',
    descriptor: {},
    fee: 10
  });
  const [customFields, setCustomFields] = useState([{ key: '', value: '' }]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData({
      ...formData,
      [name]: value
    });
  };

  const handleCustomFieldChange = (index, field, value) => {
    const updatedFields = [...customFields];
    updatedFields[index][field] = value;
    setCustomFields(updatedFields);
  };

  const addCustomField = () => {
    setCustomFields([...customFields, { key: '', value: '' }]);
  };

  const removeCustomField = (index) => {
    const updatedFields = [...customFields];
    updatedFields.splice(index, 1);
    setCustomFields(updatedFields);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');
    setSuccess('');

    try {
      // Build descriptor from custom fields
      const descriptor = {};
      customFields.forEach(field => {
        if (field.key && field.value) {
          descriptor[field.key] = field.value;
        }
      });

      const payload = {
        ...formData,
        descriptor,
        fee: parseInt(formData.fee)
      };

      // Call the MCP capability registration endpoint
      const response = await api.post('/mcp/capability/prepare_registration', payload);
      
      setSuccess(`Capability registration prepared successfully! ID: ${response.data.capability_id}`);
      setFormData({
        capabilityType: '',
        desiredName: '',
        description: '',
        descriptor: {},
        fee: 10
      });
      setCustomFields([{ key: '', value: '' }]);
    } catch (error) {
      setError(error.response?.data?.message || 'Failed to register capability');
    } finally {
      setIsLoading(false);
    }
  };

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="Add Capability">
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVCHAIN node.
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="Add Capability">
      <PageHeader
        title="Add New Capability"
        subtitle="Register a new capability with the network"
      />
      
      {error && <GlassyCard darker className={styles.error}>{error}</GlassyCard>}
      {success && <GlassyCard darker className={styles.success}>{success}</GlassyCard>}
      
      <GlassyCard darker className={styles.formContainer}>
        <form onSubmit={handleSubmit} className={styles.form}>
        <div className={styles.formGroup}>
          <label htmlFor="capabilityType">Capability Type</label>
          <select
            id="capabilityType"
            name="capabilityType"
            value={formData.capabilityType}
            onChange={handleInputChange}
            required
          >
            <option value="">Select a type</option>
            <option value="NFT">NFT</option>
            <option value="DeFi">DeFi</option>
            <option value="DAO">DAO</option>
            <option value="Custom">Custom</option>
          </select>
        </div>
        
        <div className={styles.formGroup}>
          <label htmlFor="desiredName">Name</label>
          <input
            id="desiredName"
            name="desiredName"
            type="text"
            value={formData.desiredName}
            onChange={handleInputChange}
            placeholder="Enter capability name"
            required
          />
        </div>
        
        <div className={styles.formGroup}>
          <label htmlFor="description">Description</label>
          <textarea
            id="description"
            name="description"
            value={formData.description}
            onChange={handleInputChange}
            placeholder="Describe what this capability does"
            rows={4}
            required
          />
        </div>
        
        <div className={styles.formGroup}>
          <label htmlFor="fee">Registration Fee</label>
          <input
            id="fee"
            name="fee"
            type="number"
            value={formData.fee}
            onChange={handleInputChange}
            min={1}
            required
          />
        </div>
        
        <div className={styles.customFieldsSection}>
          <h3>Custom Properties</h3>
          {customFields.map((field, index) => (
            <div key={index} className={styles.customFieldRow}>
              <input
                type="text"
                placeholder="Property name"
                value={field.key}
                onChange={(e) => handleCustomFieldChange(index, 'key', e.target.value)}
              />
              <input
                type="text"
                placeholder="Value"
                value={field.value}
                onChange={(e) => handleCustomFieldChange(index, 'value', e.target.value)}
              />
              <button
                type="button"
                onClick={() => removeCustomField(index)}
                className={`${styles.button} ${styles.danger}`}
              >
                Remove
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={addCustomField}
            className={`${styles.button} ${styles.secondary}`}
          >
            Add Property
          </button>
        </div>
        
        <button
          type="submit"
          className={`${styles.button} ${styles.primary}`}
          disabled={isLoading}
        >
          {isLoading ? 'Registering...' : 'Register Capability'}
        </button>
      </form>
     </GlassyCard>
   </PageLayout>
  );
}