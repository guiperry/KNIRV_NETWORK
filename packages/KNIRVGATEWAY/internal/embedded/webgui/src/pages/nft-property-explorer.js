import React, { useEffect, useState, useRef, useCallback } from 'react';
import Viewport from '../components/Viewport';
import SideNavigation from '../components/SideNavigation';

const ObjectType = {
  id: '',
  name: '',
  src: '',
  object_type: ''
};

const Transaction = {
  id: '',
  operation: '',
  asset: '',
  metadata: '',
  timestamp: new Date()
};

const Block = {
  index: 0,
  id: '',
  data: '',
  timestamp: new Date(),
  prev_hash: '',
  hash: ''
};

// Client-side only component for NFT Property Explorer
export default function NFTPropertyExplorer() {
  const [objects, setObjects] = useState([]);
  const [selectedObject, setSelectedObject] = useState(null);
  const [transactions, setTransactions] = useState([]);
  const [blocks, setBlocks] = useState([]);
  const [consoleLog, setConsoleLog] = useState([]);
  const [markdownContent, setMarkdownContent] = useState('# Welcome to KNIRVCHAIN NFT Property Explorer\n\nSelect an NFT property to view its details.');
  const [uploadFile, setUploadFile] = useState(null);
  const [uploadName, setUploadName] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('objects');
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // Removed unused viewerRef
  const consoleEndRef = useRef(null);
  const fileInputRef = useRef(null);

  // Log function to add messages to the console
  const logToConsole = useCallback((message) => {
    const timestamp = new Date().toLocaleTimeString();
    setConsoleLog(prev => [...prev, `[${timestamp}] ${message}`]);
  }, []);

  // Scroll console to bottom when new logs are added
  useEffect(() => {
    if (consoleEndRef.current) {
      consoleEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [consoleLog]);

  // Fetch data from the server
  useEffect(() => {
    async function fetchData() {
      setIsLoading(true);
      logToConsole('Initializing connection to KNIRVCHAIN...');

      try {
        // Fetch objects
        const objectsResponse = await fetch('/api/objects');
        if (objectsResponse.ok) {
          const objectsData = await objectsResponse.json();
          setObjects(objectsData);
          logToConsole(`Loaded ${objectsData.length} NFT properties from the chain`);
        }

        // Fetch transactions
        const transactionsResponse = await fetch('/api/transactions');
        if (transactionsResponse.ok) {
          const transactionsData = await transactionsResponse.json();
          setTransactions(transactionsData);
          logToConsole(`Loaded ${transactionsData.length} transactions from the chain`);
        }

        // Fetch blocks
        const blocksResponse = await fetch('/api/blocks');
        if (blocksResponse.ok) {
          const blocksData = await blocksResponse.json();
          setBlocks(blocksData);
          logToConsole(`Loaded ${blocksData.length} blocks from the chain`);
        }
      } catch (error) {
        console.error('Failed to fetch data:', error);
        logToConsole(`Error: Failed to connect to the chain - ${error instanceof Error ? error.message : String(error)}`);
      } finally {
        setIsLoading(false);
        logToConsole('KNIRVCHAIN connection established');
      }
    }

    fetchData();

    // Set up polling for updates every 10 seconds
    const intervalId = setInterval(() => {
      fetchData();
    }, 10000);

    return () => clearInterval(intervalId);
  }, [logToConsole]);

  // Handle object selection
  const selectObject = useCallback((object) => {
    setSelectedObject(object);
    logToConsole(`Selected NFT property: ${object.name} (ID: ${object.id})`);

    // If it's a markdown object, fetch the content
    if (object.object_type === 'markdown') {
      // This is a placeholder - in a real app, you'd fetch the markdown content
      setMarkdownContent(`# ${object.name}\n\nThis is a markdown document for NFT property ID: ${object.id}`);
    }
  }, [logToConsole]);

  // Handle file upload
  const handleFileChange = (e) => {
    if (e.target.files && e.target.files.length > 0) {
      const file = e.target.files[0];
      setUploadFile(file);
      if (!uploadName) {
        setUploadName(file.name.split('.')[0]);
      }
      logToConsole(`File selected for upload: ${file.name} (${(file.size / 1024).toFixed(2)} KB)`);
    }
  };

  // Handle file upload submission
  const handleUpload = async () => {
    if (!uploadFile || !uploadName) {
      logToConsole('Error: Please select a file and provide a name');
      return;
    }

    setIsLoading(true);
    logToConsole(`Preparing to upload ${uploadName}...`);

    // In a real app, you would upload the file to your server
    // This is a placeholder for the upload functionality
    setTimeout(() => {
      logToConsole(`Creating transaction for NFT property: ${uploadName}`);
      logToConsole('Transaction added to pending block');

      // Simulate mining
      setTimeout(() => {
        logToConsole('Mining new block...');
        setTimeout(() => {
          logToConsole('Block mined successfully!');
          logToConsole(`NFT property ${uploadName} added to the chain`);

          // Add the new object to the list
          const newObject = {
            id: `nft-${Date.now()}`,
            name: uploadName,
            object_type: uploadFile.type.includes('gltf') ? 'gltf' :
                          uploadFile.type.includes('markdown') ? 'markdown' : 'unknown'
          };

          setObjects(prev => [...prev, newObject]);
          setUploadFile(null);
          setUploadName('');
          if (fileInputRef.current) {
            fileInputRef.current.value = '';
          }
          setIsLoading(false);
        }, 2000);
      }, 3000);
    }, 2000);
  };

  // Handle mining a new block
  const handleMineBlock = () => {
    setIsLoading(true);
    logToConsole('Initiating mining process...');

    // Simulate mining
    setTimeout(() => {
      logToConsole('Validating pending transactions...');
      setTimeout(() => {
        logToConsole('Computing proof of work...');
        setTimeout(() => {
          logToConsole('Block mined successfully!');
          logToConsole('New block added to the chain');

          // Add a new block to the list
          const newBlock = {
            index: blocks.length,
            id: `block-${Date.now()}`,
            data: 'Transaction data',
            timestamp: new Date(),
            prev_hash: blocks.length > 0 ? blocks[blocks.length - 1].hash : '0',
            hash: `hash-${Date.now()}`
          };

          setBlocks(prev => [...prev, newBlock]);
          setIsLoading(false);
        }, 2000);
      }, 2000);
    }, 1000);
  };

  return (
    <div className="nft-property-explorer">
      {/* Burger Menu Sidebar */}
      <div className={`sidebar-overlay ${sidebarOpen ? 'open' : ''}`} onClick={() => setSidebarOpen(false)}>
        <div className="sidebar-content" onClick={(e) => e.stopPropagation()}>
          <div className="sidebar-header">
            <button className="close-sidebar" onClick={() => setSidebarOpen(false)}>×</button>
          </div>
          <SideNavigation activePage="nft-property-explorer" />
        </div>
      </div>

      <div className="header">
        <div className="header-left">
          <div className="burger-menu" onClick={() => setSidebarOpen(true)}>
            <span></span>
            <span></span>
            <span></span>
          </div>
          <div className="logo">
            <h1>NFT Property Explorer</h1>
            <span className="subtitle">KNIRVCHAIN 3D Object Blockchain Viewer</span>
          </div>
        </div>
        <div className="nav-links">
          <button
            className={activeTab === 'objects' ? 'active' : ''}
            onClick={() => setActiveTab('objects')}
          >
            NFT Properties
          </button>
          <button
            className={activeTab === 'transactions' ? 'active' : ''}
            onClick={() => setActiveTab('transactions')}
          >
            Transactions
          </button>
          <button
            className={activeTab === 'blocks' ? 'active' : ''}
            onClick={() => setActiveTab('blocks')}
          >
            Blocks
          </button>
          <button
            className={activeTab === 'console' ? 'active' : ''}
            onClick={() => setActiveTab('console')}
          >
            Console
          </button>
        </div>
      </div>

      <div className="main-content">
        <div className="sidebar">
          <div className="sidebar-header">
            <h3>Chain Explorer</h3>
          </div>

          {activeTab === 'objects' && (
            <div className="object-list">
              <div className="list-header">
                <h4>Available NFT Properties</h4>
                <span className="count">{objects.length}</span>
              </div>
              <div className="objects-container">
                {objects.length > 0 ? (
                  objects.map(obj => (
                    <div
                      key={obj.id}
                      className={`object-item ${selectedObject?.id === obj.id ? 'active' : ''}`}
                      onClick={() => selectObject(obj)}
                    >
                      <span className="object-name">{obj.name}</span>
                      <span className="object-type">{obj.object_type || 'unknown'}</span>
                    </div>
                  ))
                ) : (
                  <div className="empty-state">No NFT properties available</div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'transactions' && (
            <div className="transaction-list">
              <div className="list-header">
                <h4>Transactions</h4>
                <span className="count">{transactions.length}</span>
              </div>
              <div className="transactions-container">
                {transactions.length > 0 ? (
                  transactions.map(tx => (
                    <div key={tx.id} className="transaction-item">
                      <div className="tx-id">{tx.id.substring(0, 8)}...</div>
                      <div className="tx-operation">{tx.operation}</div>
                      <div className="tx-asset">Asset: {tx.asset.substring(0, 6)}...</div>
                      <div className="tx-time">
                        {new Date(tx.timestamp).toLocaleString()}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="empty-state">No transactions available</div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'blocks' && (
            <div className="block-list">
              <div className="list-header">
                <h4>Blocks</h4>
                <span className="count">{blocks.length}</span>
              </div>
              <div className="blocks-container">
                {blocks.length > 0 ? (
                  blocks.map(block => (
                    <div key={block.id} className="block-item">
                      <div className="block-index">Block #{block.index}</div>
                      <div className="block-id">{block.id.substring(0, 8)}...</div>
                      <div className="block-hash">Hash: {block.hash.substring(0, 6)}...</div>
                      <div className="block-time">
                        {new Date(block.timestamp).toLocaleString()}
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="empty-state">No blocks available</div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'console' && (
            <div className="console-full">
              <div className="console-header">
                <h4>Chain Log</h4>
                <button
                  className="clear-console"
                  onClick={() => setConsoleLog([])}
                >
                  Clear
                </button>
              </div>
              <div className="console-content">
                {consoleLog.map((log, index) => (
                  <div key={index} className="log-line">{log}</div>
                ))}
                <div ref={consoleEndRef} />
              </div>
            </div>
          )}

          <div className="upload-section">
            <h4>Upload NFT Property</h4>
            <div className="upload-form">
              <input
                type="text"
                placeholder="NFT Property Name"
                value={uploadName}
                onChange={(e) => setUploadName(e.target.value)}
                className="upload-name"
              />
              <div className="file-input-container">
                <input
                  type="file"
                  ref={fileInputRef}
                  onChange={handleFileChange}
                  className="file-input"
                  accept=".glb,.gltf,.obj,.md,.markdown"
                />
                <button
                  className="browse-button"
                  onClick={() => fileInputRef.current?.click()}
                >
                  Browse
                </button>
              </div>
              <button
                className="upload-button"
                onClick={handleUpload}
                disabled={!uploadFile || !uploadName || isLoading}
              >
                {isLoading ? 'Processing...' : 'Upload to Chain'}
              </button>
            </div>
          </div>

          <div className="mining-section">
            <h4>Blockchain Controls</h4>
            <button
              className="mine-button"
              onClick={handleMineBlock}
              disabled={isLoading}
            >
              {isLoading ? 'Mining...' : 'Mine New Block'}
            </button>
          </div>
        </div>

        <div className="content-area">
          <div className="viewport-container">
            <div className="viewport-header">
              <h3>3D NFT Property Viewport</h3>
              {selectedObject && (
                <div className="object-info">
                  <span className="selected-name">{selectedObject.name}</span>
                  <span className="selected-id">ID: {selectedObject.id}</span>
                </div>
              )}
            </div>
            <div className="viewport">
              {selectedObject ? (
                <div style={{ width: '100%', height: '100%' }}>
                  <Viewport modelId={selectedObject.id} />
                </div>
              ) : (
                <div className="placeholder">Select an NFT property to view</div>
              )}
            </div>
          </div>

          <div className="bottom-section">
            <div className="markdown-viewer">
              <div className="markdown-header">
                <h3>Documentation</h3>
              </div>
              <div className="markdown-content">
                {/* In a real app, you would use a markdown renderer here */}
                <pre>{markdownContent}</pre>
              </div>
            </div>

            <div className="console-container">
              <div className="console-header">
                <h3>Chain Log</h3>
              </div>
              <div className="console">
                {consoleLog.slice(-10).map((log, index) => (
                  <div key={index} className="log-line">{log}</div>
                ))}
                <div ref={consoleEndRef} />
              </div>
            </div>
          </div>
        </div>
      </div>

      <style jsx>{`
        /* Global Styles */
        .nft-property-explorer {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background-color: #0a0e17;
          color: #e0e0ff;
          font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        /* Burger Menu Styles */
        .burger-menu {
          display: flex;
          flex-direction: column;
          justify-content: space-around;
          width: 30px;
          height: 24px;
          cursor: pointer;
          padding: 5px;
          margin-right: 15px;
        }

        .burger-menu span {
          width: 100%;
          height: 3px;
          background: #b0b0ff;
          border-radius: 2px;
          transition: all 0.3s ease;
          transform-origin: center;
        }

        .burger-menu:hover span {
          background: #d0d0ff;
        }

        /* Sidebar Overlay */
        .sidebar-overlay {
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background: rgba(0, 0, 0, 0.5);
          z-index: 1000;
          opacity: 0;
          visibility: hidden;
          transition: all 0.3s ease;
        }

        .sidebar-overlay.open {
          opacity: 1;
          visibility: visible;
        }

        .sidebar-content {
          position: absolute;
          top: 0;
          left: 0;
          width: 320px;
          height: 100%;
          background: rgba(20, 30, 60, 0.95);
          backdrop-filter: blur(10px);
          border-right: 1px solid rgba(100, 130, 255, 0.2);
          transform: translateX(-100%);
          transition: transform 0.3s ease;
        }

        .sidebar-overlay.open .sidebar-content {
          transform: translateX(0);
        }

        .sidebar-header {
          padding: 15px;
          border-bottom: 1px solid rgba(100, 130, 255, 0.2);
          display: flex;
          justify-content: flex-end;
        }

        .close-sidebar {
          background: none;
          border: none;
          color: #b0b0ff;
          font-size: 24px;
          cursor: pointer;
          padding: 5px;
          border-radius: 4px;
          transition: all 0.2s ease;
        }

        .close-sidebar:hover {
          background: rgba(100, 130, 255, 0.2);
          color: #d0d0ff;
        }

        /* Responsive Design */
        @media (max-width: 768px) {
          .sidebar-content {
            width: 280px;
          }

          .header {
            padding: 10px 15px;
          }

          .burger-menu {
            width: 24px;
            height: 20px;
          }

          .logo h1 {
            font-size: 20px;
          }

          .subtitle {
            font-size: 12px;
          }
        }

        /* Header Styles */
        .header {
          background: rgba(16, 24, 48, 0.8);
          backdrop-filter: blur(10px);
          border-bottom: 1px solid rgba(100, 130, 255, 0.2);
          padding: 15px 25px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        }

        .header-left {
          display: flex;
          align-items: center;
        }

        .logo {
          display: flex;
          flex-direction: column;
        }

        .logo h1 {
          margin: 0;
          font-size: 28px;
          font-weight: 700;
          background: linear-gradient(90deg, #64b5f6, #9575cd);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          letter-spacing: 1px;
        }

        .subtitle {
          font-size: 14px;
          color: #a0a0d0;
          margin-top: 4px;
        }

        .nav-links {
          display: flex;
          gap: 10px;
        }

        .nav-links button {
          background: rgba(30, 40, 70, 0.5);
          border: 1px solid rgba(100, 130, 255, 0.3);
          color: #b0b0ff;
          padding: 8px 16px;
          border-radius: 6px;
          cursor: pointer;
          transition: all 0.2s ease;
          font-size: 14px;
        }

        .nav-links button:hover {
          background: rgba(40, 60, 100, 0.7);
          border-color: rgba(120, 150, 255, 0.5);
        }

        .nav-links button.active {
          background: rgba(60, 80, 170, 0.7);
          border-color: rgba(140, 170, 255, 0.7);
          color: white;
        }

        /* Main Content Layout */
        .main-content {
          display: flex;
          flex: 1;
          overflow: hidden;
        }

        /* Sidebar Styles */
        .sidebar {
          width: 320px;
          background: rgba(20, 30, 60, 0.7);
          backdrop-filter: blur(10px);
          border-right: 1px solid rgba(100, 130, 255, 0.2);
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        .sidebar-header {
          padding: 15px;
          border-bottom: 1px solid rgba(100, 130, 255, 0.2);
        }

        .sidebar-header h3 {
          margin: 0;
          font-size: 18px;
          font-weight: 600;
          color: #d0d0ff;
        }

        .list-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 10px 15px;
          border-bottom: 1px solid rgba(100, 130, 255, 0.1);
        }

        .list-header h4 {
          margin: 0;
          font-size: 16px;
          font-weight: 500;
        }

        .count {
          background: rgba(60, 80, 170, 0.3);
          padding: 2px 8px;
          border-radius: 12px;
          font-size: 12px;
          color: #b0b0ff;
        }

        /* Object List Styles */
        .object-list, .transaction-list, .block-list {
          display: flex;
          flex-direction: column;
          flex: 1;
          overflow: hidden;
        }

        .objects-container, .transactions-container, .blocks-container {
          overflow-y: auto;
          padding: 10px;
          flex: 1;
        }

        .object-item, .transaction-item, .block-item {
          background: rgba(30, 40, 80, 0.5);
          border: 1px solid rgba(100, 130, 255, 0.2);
          border-radius: 8px;
          padding: 12px;
          margin-bottom: 8px;
          cursor: pointer;
          transition: all 0.2s ease;
        }

        .object-item:hover, .transaction-item:hover, .block-item:hover {
          background: rgba(40, 60, 120, 0.6);
          border-color: rgba(120, 150, 255, 0.4);
          transform: translateY(-2px);
        }

        .object-item.active {
          background: rgba(60, 90, 200, 0.4);
          border-color: rgba(140, 170, 255, 0.6);
          box-shadow: 0 0 15px rgba(100, 130, 255, 0.3);
        }

        .object-name {
          font-weight: 500;
          display: block;
          margin-bottom: 4px;
        }

        .object-type {
          font-size: 12px;
          color: #8080c0;
          background: rgba(60, 80, 170, 0.2);
          padding: 2px 6px;
          border-radius: 4px;
        }

        .tx-id, .block-id {
          font-family: monospace;
          font-size: 13px;
          color: #a0a0ff;
          margin-bottom: 4px;
        }

        .tx-operation, .block-index {
          font-weight: 500;
          margin-bottom: 4px;
        }

        .tx-asset, .block-hash {
          font-size: 12px;
          color: #8080c0;
          margin-bottom: 4px;
        }

        .tx-time, .block-time {
          font-size: 11px;
          color: #7070a0;
        }

        .empty-state {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100px;
          color: #6060a0;
          font-style: italic;
        }

        /* Upload Section Styles */
        .upload-section, .mining-section {
          padding: 15px;
          border-top: 1px solid rgba(100, 130, 255, 0.2);
          background: rgba(25, 35, 70, 0.5);
        }

        .upload-section h4, .mining-section h4 {
          margin: 0 0 10px 0;
          font-size: 16px;
          font-weight: 500;
        }

        .upload-form {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }

        .upload-name {
          padding: 8px 12px;
          border-radius: 6px;
          border: 1px solid rgba(100, 130, 255, 0.3);
          background: rgba(20, 30, 60, 0.5);
          color: #d0d0ff;
          font-size: 14px;
        }

        .file-input-container {
          display: flex;
          gap: 8px;
        }

        .file-input {
          display: none;
        }

        .browse-button {
          flex: 1;
          padding: 8px 12px;
          border-radius: 6px;
          border: 1px solid rgba(100, 130, 255, 0.3);
          background: rgba(30, 40, 80, 0.5);
          color: #b0b0ff;
          cursor: pointer;
          transition: all 0.2s ease;
          font-size: 14px;
        }

        .browse-button:hover {
          background: rgba(40, 60, 120, 0.6);
          border-color: rgba(120, 150, 255, 0.4);
        }

        .upload-button, .mine-button {
          padding: 10px;
          border-radius: 6px;
          border: none;
          background: linear-gradient(135deg, #4a6bff, #2845e0);
          color: white;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s ease;
          font-size: 14px;
        }

        .upload-button:hover, .mine-button:hover {
          background: linear-gradient(135deg, #5a7bff, #3855f0);
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(40, 69, 224, 0.4);
        }

        .upload-button:disabled, .mine-button:disabled {
          background: linear-gradient(135deg, #3a4a80, #2a3560);
          cursor: not-allowed;
          transform: none;
          box-shadow: none;
          opacity: 0.7;
        }

        /* Content Area Styles */
        .content-area {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
          padding: 15px;
        }

        .viewport-container {
          flex: 2;
          display: flex;
          flex-direction: column;
          background: rgba(25, 35, 70, 0.5);
          border: 1px solid rgba(100, 130, 255, 0.2);
          border-radius: 10px;
          overflow: hidden;
          margin-bottom: 15px;
        }

        .viewport-header {
          padding: 12px 15px;
          border-bottom: 1px solid rgba(100, 130, 255, 0.2);
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .viewport-header h3 {
          margin: 0;
          font-size: 18px;
          font-weight: 600;
        }

        .object-info {
          display: flex;
          gap: 10px;
          align-items: center;
        }

        .selected-name {
          font-weight: 500;
          color: #b0b0ff;
        }

        .selected-id {
          font-size: 12px;
          color: #8080c0;
          font-family: monospace;
        }

        .viewport {
          flex: 1;
          background: #0c1020;
          position: relative;
        }

        iframe {
          width: 100%;
          height: 100%;
          border: none;
        }

        .placeholder {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          color: #6060a0;
          font-size: 16px;
        }

        /* Bottom Section Styles */
        .bottom-section {
          flex: 1;
          display: flex;
          gap: 15px;
          min-height: 200px;
        }

        .markdown-viewer, .console-container {
          flex: 1;
          display: flex;
          flex-direction: column;
          background: rgba(25, 35, 70, 0.5);
          border: 1px solid rgba(100, 130, 255, 0.2);
          border-radius: 10px;
          overflow: hidden;
        }

        .markdown-header, .console-header {
          padding: 10px 15px;
          border-bottom: 1px solid rgba(100, 130, 255, 0.2);
        }

        .markdown-header h3, .console-header h3 {
          margin: 0;
          font-size: 16px;
          font-weight: 600;
        }

        .markdown-content {
          flex: 1;
          padding: 15px;
          overflow-y: auto;
          font-size: 14px;
          line-height: 1.5;
        }

        .markdown-content pre {
          white-space: pre-wrap;
          font-family: inherit;
          margin: 0;
        }

        .console, .console-content {
          flex: 1;
          padding: 10px;
          overflow-y: auto;
          font-family: monospace;
          font-size: 13px;
          background: rgba(10, 15, 30, 0.7);
        }

        .log-line {
          padding: 3px 0;
          border-bottom: 1px solid rgba(100, 130, 255, 0.1);
          color: #a0a0d0;
        }

        /* Full Console View */
        .console-full {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        .console-full .console-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .clear-console {
          background: rgba(60, 80, 170, 0.3);
          border: 1px solid rgba(100, 130, 255, 0.3);
          color: #b0b0ff;
          padding: 4px 8px;
          border-radius: 4px;
          cursor: pointer;
          font-size: 12px;
        }

        .clear-console:hover {
          background: rgba(70, 90, 180, 0.4);
        }
      `}</style>
    </div>
  );
}