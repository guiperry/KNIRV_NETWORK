import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import styles from './OnboardingFlowUpdated.module.css';

// Define constants that were previously inline
const darkBlue = '#000c2e';
const brightBlue = '#007bff';

const OnboardingFlow = ({ onComplete }) => {
  const [currentStep, setCurrentStep] = useState(1);
  const [inventoryData, setInventoryData] = useState(null);
  const [totalValue, setTotalValue] = useState(0);
  const [tokenName, setTokenName] = useState('');
  const [tokenId, setTokenId] = useState(1); // This would be fetched from the blockchain
  const [isUploading, setIsUploading] = useState(false);
  const [isMinting, setIsMinting] = useState(false);
  const [inputMethod, setInputMethod] = useState('upload'); // 'upload' or 'form'
  const [formData, setFormData] = useState({
    'product.partNumber': 'ABC-123',
    'location.locationIdentifier': 'Warehouse-A',
    'inventoryType': 'Raw Material',
    'quantity': '100',
    'quantityUnits': 'kg',
    'expirationDate': '2025-12-31',
    'inventoryParentType': 'Material',
    'class': 'Class A',
    'segment': 'Manufacturing',
    'lotCode': 'LOT-2023-001',
    'status': 'Available',
    'value': '1500',
    'valueCurrency': 'USD',
    'sourceLink': 'https://supplier.com/order/123',
    'storageDate': '2023-01-15'
  });
  const [formItems, setFormItems] = useState([]);
  
  // Inventory mint pool
  const [mintPool, setMintPool] = useState([]);
  const [mintPoolTotalValue, setMintPoolTotalValue] = useState(0);
  const [mintPoolTotalItems, setMintPoolTotalItems] = useState(0);
  
  const router = useRouter();

  // CSS classes are now used instead of inline styles

  const handleInputMethodChange = (method) => {
    setInputMethod(method);
    // Reset data if changing methods
    if (method === 'upload') {
      setFormItems([]);
    } else {
      setInventoryData(null);
    }
  };

  const handleFormChange = (e, field) => {
    setFormData({
      ...formData,
      [field]: e.target.value
    });
  };

  const handleAddFormItem = () => {
    // Validate that required fields are filled
    if (!formData['product.partNumber'] || !formData.quantity || !formData.value) {
      alert('Please fill in at least Part Number, Quantity, and Value fields');
      return;
    }

    const newItem = {
      id: formItems.length + 1,
      ...formData
    };

    const updatedItems = [...formItems, newItem];
    setFormItems(updatedItems);

    // Calculate total value
    const calculatedTotal = updatedItems.reduce(
      (sum, item) => sum + (parseFloat(item.value) * parseFloat(item.quantity)), 0
    );
    
    setTotalValue(calculatedTotal);
    
    // Generate token name based on first item if not already set
    if (!tokenName && updatedItems.length === 1) {
      setTokenName(`Sac ${updatedItems[0]['product.partNumber']} #${tokenId}`);
    }

    // Reset form to default values
    setFormData({
      'product.partNumber': 'ABC-' + (Math.floor(Math.random() * 900) + 100),
      'location.locationIdentifier': 'Warehouse-A',
      'inventoryType': 'Raw Material',
      'quantity': '100',
      'quantityUnits': 'kg',
      'expirationDate': '2025-12-31',
      'inventoryParentType': 'Material',
      'class': 'Class A',
      'segment': 'Manufacturing',
      'lotCode': 'LOT-2023-' + (Math.floor(Math.random() * 900) + 100),
      'status': 'Available',
      'value': '1500',
      'valueCurrency': 'USD',
      'sourceLink': 'https://supplier.com/order/123',
      'storageDate': '2023-01-15'
    });
  };

  const handleRemoveFormItem = (id) => {
    const updatedItems = formItems.filter(item => item.id !== id);
    setFormItems(updatedItems);
    
    // Recalculate total value
    const calculatedTotal = updatedItems.reduce(
      (sum, item) => sum + (parseFloat(item.value) * parseFloat(item.quantity)), 0
    );
    
    setTotalValue(calculatedTotal);
    
    // Update token name if first item was removed
    if (updatedItems.length > 0 && id === 1) {
      setTokenName(`Sac ${updatedItems[0]['product.partNumber']} #${tokenId}`);
    } else if (updatedItems.length === 0) {
      setTokenName('');
    }
  };

  const handleFileUpload = (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploading(true);
    
    // Simulate file processing
    setTimeout(() => {
      // This would be replaced with actual file parsing logic
      const mockInventoryData = [
        { id: Date.now() + 1, 'product.partNumber': 'ABC-123', name: 'Product A', value: 100.00, price: 100.00, quantity: 5, 'location.locationIdentifier': 'Warehouse-A', 'inventoryType': 'Finished Good' },
        { id: Date.now() + 2, 'product.partNumber': 'DEF-456', name: 'Product B', value: 75.50, price: 75.50, quantity: 10, 'location.locationIdentifier': 'Warehouse-B', 'inventoryType': 'Raw Material' },
        { id: Date.now() + 3, 'product.partNumber': 'GHI-789', name: 'Product C', value: 250.00, price: 250.00, quantity: 2, 'location.locationIdentifier': 'Warehouse-A', 'inventoryType': 'Component' },
        { id: Date.now() + 4, 'product.partNumber': 'JKL-012', name: 'Product D', value: 30.25, price: 30.25, quantity: 20, 'location.locationIdentifier': 'Warehouse-C', 'inventoryType': 'Spare Part' },
      ];
      
      // Calculate total using either price or value field
      const calculatedTotal = mockInventoryData.reduce(
        (sum, item) => sum + ((item.value || item.price) * item.quantity), 0
      );
      
      setInventoryData(mockInventoryData);
      setTotalValue(calculatedTotal);
      setIsUploading(false);
      
      // Generate a token name based on the first product if not already set
      if (!tokenName && mockInventoryData.length > 0) {
        const firstItem = mockInventoryData[0];
        const identifier = firstItem['product.partNumber'] || firstItem.name.split(' ')[0];
        setTokenName(`Sac ${identifier} #${tokenId}`);
      }
    }, 1500);
  };
  
  const addToMintPool = () => {
    if (inputMethod === 'upload' && inventoryData) {
      // Create a batch entry for the uploaded spreadsheet
      const spreadsheetBatch = {
        id: Date.now(),
        type: 'spreadsheet',
        name: `Spreadsheet Upload ${new Date().toLocaleTimeString()}`,
        items: inventoryData,
        itemCount: inventoryData.length,
        totalValue: inventoryData.reduce(
          (sum, item) => sum + ((item.value || item.price) * item.quantity), 0
        )
      };
      
      const updatedMintPool = [...mintPool, spreadsheetBatch];
      setMintPool(updatedMintPool);
      
      // Update mint pool totals
      updateMintPoolTotals(updatedMintPool);
      
      // Reset the current inventory data
      setInventoryData(null);
      setTotalValue(0);
      
    } else if (inputMethod === 'form' && formItems.length > 0) {
      // Create a batch entry for the form items
      const formBatch = {
        id: Date.now(),
        type: 'form',
        name: `Manual Entry ${new Date().toLocaleTimeString()}`,
        items: formItems,
        itemCount: formItems.length,
        totalValue: formItems.reduce(
          (sum, item) => sum + (parseFloat(item.value) * parseFloat(item.quantity)), 0
        )
      };
      
      const updatedMintPool = [...mintPool, formBatch];
      setMintPool(updatedMintPool);
      
      // Update mint pool totals
      updateMintPoolTotals(updatedMintPool);
      
      // Reset the form items
      setFormItems([]);
      setTotalValue(0);
    }
  };
  
  const updateMintPoolTotals = (pool) => {
    const totalItems = pool.reduce((sum, batch) => sum + batch.itemCount, 0);
    const totalValue = pool.reduce((sum, batch) => sum + batch.totalValue, 0);
    
    setMintPoolTotalItems(totalItems);
    setMintPoolTotalValue(totalValue);
    
    // Update token name if not already set
    if (!tokenName && pool.length > 0) {
      // Find the first item in the first batch
      const firstBatch = pool[0];
      if (firstBatch.items && firstBatch.items.length > 0) {
        const firstItem = firstBatch.items[0];
        const identifier = firstItem['product.partNumber'] || 
                          (firstItem.name ? firstItem.name.split(' ')[0] : 'Inventory');
        setTokenName(`Sac ${identifier} #${tokenId}`);
      }
    }
  };
  
  const removeBatchFromMintPool = (batchId) => {
    const updatedMintPool = mintPool.filter(batch => batch.id !== batchId);
    setMintPool(updatedMintPool);
    updateMintPoolTotals(updatedMintPool);
  };

  const handleMintInventory = () => {
    // Check if we have items in the mint pool
    if (mintPool.length === 0) {
      // If no items in mint pool, check if we have current inventory data to add
      if (inputMethod === 'upload' && inventoryData) {
        addToMintPool();
      } else if (inputMethod === 'form' && formItems.length > 0) {
        addToMintPool();
      } else {
        alert('Please add inventory items before minting');
        return;
      }
    }
    
    setIsMinting(true);
    
    // Simulate minting process
    setTimeout(() => {
      setIsMinting(false);
      setCurrentStep(3);
    }, 2000);
  };

  const handleCompleteOnboarding = () => {
    if (onComplete) {
      onComplete();
    }
  };

  const renderStep1 = () => (
    <div className={`${styles.glassyContainer} ${styles.step1Container}`}>
      <h2 className={styles.stepTitle}>Step 1: Add Inventory</h2>
      <p className={styles.stepDescription}>
        Add your tradable inventory using one of the methods below.
        <br />
        <strong>Important:</strong> Each item in your inventory should have a &quotvalue&quot or &quotprice&quot field.
        This will be used to calculate the total value of your inventory.
      </p>
      
      {/* Input Method Selection */}
      <div className={styles.inputMethodSelector}>
        <button
          onClick={() => handleInputMethodChange('upload')}
          className={`${styles.methodButton} ${inputMethod === 'upload' ? styles.activeMethod : styles.inactiveMethod}`}
        >
          Upload Spreadsheet
        </button>
        <button
          onClick={() => handleInputMethodChange('form')}
          className={`${styles.methodButton} ${inputMethod === 'form' ? styles.activeMethod : styles.inactiveMethod}`}
        >
          Manual Entry Form
        </button>
      </div>
      
      {/* Three-column layout */}
      <div className={styles.columnsContainer}>
        {/* Left Column - Input Methods */}
        <div className={styles.column}>
          {/* File Upload Method */}
          {inputMethod === 'upload' && (
            <div className={styles.fileUploadContainer}>
              <div className={`${styles.glassyContainer} ${styles.dropzone}`}>
                <span className={styles.fileIcon}>📄</span>
                <p>Drag and drop your inventory spreadsheet here</p>
                <p className={styles.fileHint}>
                  Supported formats: .xlsx, .csv
                </p>
                <input
                  type="file"
                  accept=".xlsx,.csv"
                  onChange={handleFileUpload}
                  className={styles.fileInput}
                />
              </div>
              
              {isUploading && (
                <div className={styles.uploadProgress}>
                  <p>Uploading and processing your inventory...</p>
                  <div className={styles.progressBarContainer}>
                    <div className={styles.progressBar}></div>
                  </div>
                </div>
              )}
              

            </div>
          )}
          
          {/* Form Entry Method */}
          {inputMethod === 'form' && (
            <div className={styles.formContainer}>
              <div className={`${styles.glassyContainer} ${styles.columnContent}`}>
                <h3 className={styles.summaryTitle}>Add Inventory Item</h3>
                
                <div className={styles.formFieldsContainer}>
                  <div className={styles.formField}>
                    <label htmlFor="partNumber" className={styles.fieldLabel}>
                      Part Number*
                    </label>
                    <input
                      id="partNumber"
                      type="text"
                      value={formData['product.partNumber']}
                      onChange={(e) => handleFormChange(e, 'product.partNumber')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="location" className={styles.fieldLabel}>
                      Location
                    </label>
                    <input
                      id="location"
                      type="text"
                      value={formData['location.locationIdentifier']}
                      onChange={(e) => handleFormChange(e, 'location.locationIdentifier')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="inventoryType" className={styles.fieldLabel}>
                      Inventory Type
                    </label>
                    <input
                      id="inventoryType"
                      type="text"
                      value={formData.inventoryType}
                      onChange={(e) => handleFormChange(e, 'inventoryType')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="quantity" className={styles.fieldLabel}>
                      Quantity*
                    </label>
                    <input
                      id="quantity"
                      type="number"
                      value={formData.quantity}
                      onChange={(e) => handleFormChange(e, 'quantity')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="quantityUnits" className={styles.fieldLabel}>
                      Quantity Units
                    </label>
                    <input
                      id="quantityUnits"
                      type="text"
                      value={formData.quantityUnits}
                      onChange={(e) => handleFormChange(e, 'quantityUnits')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="expirationDate" className={styles.fieldLabel}>
                      Expiration Date
                    </label>
                    <input
                      id="expirationDate"
                      type="date"
                      value={formData.expirationDate}
                      onChange={(e) => handleFormChange(e, 'expirationDate')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="parentType" className={styles.fieldLabel}>
                      Parent Type
                    </label>
                    <input
                      id="parentType"
                      type="text"
                      value={formData.inventoryParentType}
                      onChange={(e) => handleFormChange(e, 'inventoryParentType')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="class" className={styles.fieldLabel}>
                      Class
                    </label>
                    <input
                      id="class"
                      type="text"
                      value={formData.class}
                      onChange={(e) => handleFormChange(e, 'class')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="segment" className={styles.fieldLabel}>
                      Segment
                    </label>
                    <input
                      id="segment"
                      type="text"
                      value={formData.segment}
                      onChange={(e) => handleFormChange(e, 'segment')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="lotCode" className={styles.fieldLabel}>
                      Lot Code
                    </label>
                    <input
                      id="lotCode"
                      type="text"
                      value={formData.lotCode}
                      onChange={(e) => handleFormChange(e, 'lotCode')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="status" className={styles.fieldLabel}>
                      Status
                    </label>
                    <input
                      id="status"
                      type="text"
                      value={formData.status}
                      onChange={(e) => handleFormChange(e, 'status')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="value" className={styles.fieldLabel}>
                      Value*
                    </label>
                    <input
                      id="value"
                      type="number"
                      value={formData.value}
                      onChange={(e) => handleFormChange(e, 'value')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="currency" className={styles.fieldLabel}>
                      Currency
                    </label>
                    <input
                      id="currency"
                      type="text"
                      value={formData.valueCurrency}
                      onChange={(e) => handleFormChange(e, 'valueCurrency')}
                      className={styles.fieldInput}
                    />
                  </div>

                  <div className={styles.formField}>
                    <label htmlFor="sourceLink" className={styles.fieldLabel}>
                      Source Link
                    </label>
                    <input
                      id="sourceLink"
                      type="text"
                      value={formData.sourceLink}
                      onChange={(e) => handleFormChange(e, 'sourceLink')}
                      className={styles.fieldInput}
                    />
                  </div>
                  
                  <div className={styles.formField}>
                    <label htmlFor="storageDate" className={styles.fieldLabel}>
                      Storage Date
                    </label>
                    <input
                      id="storageDate"
                      type="date"
                      value={formData.storageDate}
                      onChange={(e) => handleFormChange(e, 'storageDate')}
                      className={styles.fieldInput}
                    />
                  </div>
                </div>
                
                <div className={styles.addButtonContainer}>
                  <button
                    onClick={handleAddFormItem}
                    className={styles.addButton}
                  >
                    Add Item to Inventory
                  </button>
                </div>
              </div>
              

            </div>
          )}
        </div>

        {/* Middle Column - Inventory Summary */}
        {(formItems.length > 0 || inventoryData) && (
          <div className={styles.column}>
            <div className={`${styles.glassyContainer} ${styles.columnContent}`}>
              <h3 className={styles.summaryTitle}>Inventory Summary</h3>
              <p className={styles.summaryStats}>
                Total Items: {inventoryData ? inventoryData.length : formItems.length} |
                Total Value: ${totalValue.toFixed(2)}
              </p>

              <div className={`${styles.glassyContainer} ${styles.tableContainer}`}>
                {formItems.length > 0 && (
                  <table className={styles.inventoryTable}>
                    <thead className={styles.tableHeader}>
                      <tr className={styles.tableHeaderRow}>
                        <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Part Number</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Type</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Quantity</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Value</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Total</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textCenter}`}>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {formItems.map(item => (
                        <tr key={item.id} className={styles.tableRow}>
                          <td className={`${styles.tableCell} ${styles.textLeft}`}>{item['product.partNumber']}</td>
                          <td className={`${styles.tableCell} ${styles.textLeft}`}>{item.inventoryType}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>{item.quantity} {item.quantityUnits}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>${parseFloat(item.value).toFixed(2)}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>${(parseFloat(item.value) * parseFloat(item.quantity)).toFixed(2)}</td>
                          <td className={`${styles.tableCell} ${styles.textCenter}`}>
                            <button
                              onClick={() => handleRemoveFormItem(item.id)}
                              className={styles.removeButton}
                            >
                              Remove
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}

                {inventoryData && (
                  <table className={styles.inventoryTable}>
                    <thead className={styles.tableHeader}>
                      <tr className={styles.tableHeaderRow}>
                        <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Part Number</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Name</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Quantity</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Value</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      {inventoryData.map(item => (
                        <tr key={item.id} className={styles.tableRow}>
                          <td className={`${styles.tableCell} ${styles.textLeft}`}>{item['product.partNumber'] || '-'}</td>
                          <td className={`${styles.tableCell} ${styles.textLeft}`}>{item.name || '-'}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>{item.quantity}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>${(item.value || item.price).toFixed(2)}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>${((item.value || item.price) * item.quantity).toFixed(2)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>

              <div className={styles.addToPoolButtonContainer}>
                <button
                  onClick={addToMintPool}
                  className={styles.addToPoolButton}
                >
                  Add to Mint Pool
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Right Column - Mint Pool */}
        <div className={styles.column}>
          <div className={`${styles.glassyContainer} ${mintPool.length > 0 ? styles.mintPoolContainer : styles.emptyMintPool}`}>
            <h3 className={styles.mintPoolTitle}>
              Inventory Mint Pool
            </h3>

            {mintPool.length > 0 ? (
              <>
                <p className={styles.mintPoolStats}>
                  Items ready to be minted: <strong>{mintPoolTotalItems}</strong> |
                  Total Value: <strong>${mintPoolTotalValue.toFixed(2)}</strong>
                </p>
                
                <div className={`${styles.glassyContainer} ${styles.mintPoolTableContainer}`}>
                  <table className={styles.inventoryTable}>
                    <thead className={styles.tableHeader}>
                      <tr className={styles.tableHeaderRow}>
                        <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Batch Name</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textCenter}`}>Type</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Items</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Total Value</th>
                        <th className={`${styles.tableHeaderCell} ${styles.textCenter}`}>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {mintPool.map(batch => (
                        <tr key={batch.id} className={styles.tableRow}>
                          <td className={`${styles.tableCell} ${styles.textLeft}`}>{batch.name}</td>
                          <td className={`${styles.tableCell} ${styles.textCenter}`}>
                            <span className={batch.type === 'spreadsheet' ? styles.spreadsheetBadge : styles.manualBadge}>
                              {batch.type === 'spreadsheet' ? 'Spreadsheet' : 'Manual Entry'}
                            </span>
                          </td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>{batch.itemCount}</td>
                          <td className={`${styles.tableCell} ${styles.textRight}`}>${batch.totalValue.toFixed(2)}</td>
                          <td className={`${styles.tableCell} ${styles.textCenter}`}>
                            <button
                              onClick={() => removeBatchFromMintPool(batch.id)}
                              className={styles.removeButton}
                            >
                              Remove
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            ) : (
              <div className={styles.emptyState}>
                <span className={styles.emptyStateIcon}>🧮</span>
                <p className={styles.emptyStateText}>
                  Your mint pool is empty
                </p>
                <p className={styles.emptyStateText}>
                  Add inventory items from the left panel to prepare for minting
                </p>
              </div>
            )}
            
            <div className={styles.addToPoolButtonContainer}>
              <button
                onClick={() => setCurrentStep(2)}
                disabled={mintPool.length === 0 && !inventoryData && formItems.length === 0}
                className={`${styles.mintButton} ${mintPool.length === 0 && !inventoryData && formItems.length === 0 ? styles.disabledButton : ''}`}
              >
                Continue to Minting
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderStep2 = () => {
    // Use mint pool data if available, otherwise use current inventory data
    const totalItems = mintPool.length > 0
      ? mintPoolTotalItems
      : (inventoryData ? inventoryData.length : formItems.length);

    const displayValue = mintPool.length > 0
      ? mintPoolTotalValue
      : totalValue;

    // Check if we need to add current inventory to mint pool
    const hasCurrentInventory = (inputMethod === 'upload' && inventoryData) ||
                               (inputMethod === 'form' && formItems.length > 0);

    return (
      <div className={`${styles.glassyContainer} ${styles.step2Container}`}>
        <h2 className={styles.stepTitle}>Step 2: Mint Inventory Contract</h2>
        <p className={styles.stepDescription}>
          Click the button below to mint your inventory as an ERC-721 NFT contract.
          <br /><br />
          This will:
        </p>
        
        <ul className={styles.benefitsList}>
          <li className={styles.benefitsItem}>
            <span className={styles.bulletPoint}>•</span>
            <span>Generate a contract agreement (ERC-721 NFT) that guarantees settlement of the items on your inventory list when redeemed by its corresponding serialized token.</span>
          </li>
          <li className={styles.benefitsItem}>
            <span className={styles.bulletPoint}>•</span>
            <span>Create a Sac serialized token named <strong>{tokenName}</strong> of equal value to your inventory.</span>
          </li>
          <li className={styles.benefitsItem}>
            <span className={styles.bulletPoint}>•</span>
            <span>Contribute to the starting value of SAC2-PYTHON token in the blockchain.</span>
          </li>
        </ul>
        
        {/* Mint Pool Summary */}
        <div className={`${styles.glassyContainer} ${styles.contractDetails}`}>
          <h3 className={styles.summaryTitle}>Inventory Contract Details</h3>
          <div className={styles.detailRow}>
            <span>Token Name:</span>
            <span>{tokenName}</span>
          </div>
          <div className={styles.detailRow}>
            <span>Total Items:</span>
            <span>{totalItems}</span>
          </div>
          <div className={styles.detailRow}>
            <span>Total Value:</span>
            <span>${displayValue.toFixed(2)}</span>
          </div>
          <div className={styles.detailRow}>
            <span>Batches:</span>
            <span>{mintPool.length}</span>
          </div>
        </div>
        
        {/* Mint Pool Details */}
        {mintPool.length > 0 && (
          <div className={`${styles.glassyContainer} ${styles.mintPoolDetails}`}>
            <h4 className={styles.summaryTitle}>Mint Pool Contents</h4>
            <div className={styles.tableScrollContainer}>
              <table className={styles.inventoryTable}>
                <thead className={styles.tableHeader}>
                  <tr className={styles.tableHeaderRow}>
                    <th className={`${styles.tableHeaderCell} ${styles.textLeft}`}>Batch</th>
                    <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Items</th>
                    <th className={`${styles.tableHeaderCell} ${styles.textRight}`}>Value</th>
                  </tr>
                </thead>
                <tbody>
                  {mintPool.map(batch => (
                    <tr key={batch.id} className={styles.tableRow}>
                      <td className={`${styles.tableCell} ${styles.textLeft}`}>{batch.name}</td>
                      <td className={`${styles.tableCell} ${styles.textRight}`}>{batch.itemCount}</td>
                      <td className={`${styles.tableCell} ${styles.textRight}`}>${batch.totalValue.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
        
        {/* Warning if there's current inventory not added to mint pool */}
        {hasCurrentInventory && mintPool.length > 0 && (
          <div className={styles.warningContainer}>
            <p>You have unsaved inventory items that are not in the mint pool.</p>
            <button
              onClick={addToMintPool}
              className={styles.warningButton}
            >
              Add Current Items to Mint Pool
            </button>
          </div>
        )}
        
        <button
          onClick={handleMintInventory}
          disabled={isMinting}
          className={`${styles.mintButton} ${isMinting ? styles.disabledButton : ''}`}
        >
          {isMinting ? 'Minting...' : 'Mint Inventory Contract'}
        </button>
        
        {isMinting && (
          <div className={styles.progressContainer}>
            <div className={styles.progressBarContainer}>
              <div className={styles.progressBarAnimation}></div>
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderStep3 = () => (
    <div className={`${styles.glassyContainer} ${styles.step3Container}`}>
      <div className={styles.completionIcon}>
        ✓
      </div>

      <h2 className={styles.stepTitle}>Inventory Contract Minted!</h2>
      <p className={styles.completionText}>
        Your inventory has been successfully minted as an ERC-721 NFT contract.
        <br /><br />
        The token <strong>{tokenName}</strong> has been created with a value of <strong>${mintPoolTotalValue > 0 ? mintPoolTotalValue.toFixed(2) : totalValue.toFixed(2)}</strong>.
      </p>
      
      <div className={`${styles.glassyContainer} ${styles.transactionDetails}`}>
        <h3 className={styles.summaryTitle}>Transaction Details</h3>
        <div className={styles.detailRow}>
          <span>Transaction Hash:</span>
          <span className={styles.dimText}>0x71c...9f3e</span>
        </div>
        <div className={styles.detailRow}>
          <span>Contract Address:</span>
          <span className={styles.dimText}>0x8d2...4a7b</span>
        </div>
        <div className={styles.detailRow}>
          <span>Gas Used:</span>
          <span className={styles.dimText}>0.0042 ETH</span>
        </div>
      </div>
      
      <button
        onClick={handleCompleteOnboarding}
        className={styles.completionButton}
      >
        Continue to Dashboard
      </button>
    </div>
  );

  return (
    <div className={styles.onboardingOverlay}>
      <div className={styles.onboardingModal}>
        {/* Close button */}
        <button
          onClick={handleCompleteOnboarding}
          className={styles.closeButton}
        >
          ×
        </button>
        <div className={styles.progressSteps}>
          <div className={styles.progressStepsContainer}>
            {[1, 2, 3].map(step => (
              <React.Fragment key={step}>
                <div
                  className={`${styles.stepIndicator} ${
                    currentStep > step
                      ? styles.completed
                      : currentStep === step
                        ? styles.active
                        : styles.pending
                  }`}
                  onClick={() => currentStep > step && setCurrentStep(step)}
                >
                  {currentStep > step ? '✓' : step}
                </div>
                {step < 3 && (
                  <div className={`${styles.stepConnector} ${currentStep > step ? styles.connectorActive : styles.connectorPending}`} />
                )}
              </React.Fragment>
            ))}
          </div>
        </div>
        
        {currentStep === 1 && renderStep1()}
        {currentStep === 2 && renderStep2()}
        {currentStep === 3 && renderStep3()}
      </div>
    </div>
  );
};

export default OnboardingFlow;