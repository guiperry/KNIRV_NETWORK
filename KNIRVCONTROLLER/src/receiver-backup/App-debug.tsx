import React from 'react';

function App() {
  return (
    <div style={{ 
      padding: '20px', 
      fontFamily: 'Arial, sans-serif',
      backgroundColor: '#1a1a1a',
      color: '#white',
      minHeight: '100vh'
    }}>
      <h1 style={{ color: '#00ff88' }}>🚀 KNIRV-CONTROLLER Debug Mode</h1>
      <p>If you can see this, the React app is working!</p>
      
      <div style={{ 
        backgroundColor: '#2a2a2a', 
        padding: '15px', 
        borderRadius: '8px',
        margin: '20px 0'
      }}>
        <h2>System Status</h2>
        <ul>
          <li>✅ React app loaded successfully</li>
          <li>✅ TypeScript compilation working</li>
          <li>✅ CSS styles applied</li>
          <li>✅ Server serving static files</li>
        </ul>
      </div>

      <div style={{ 
        backgroundColor: '#2a2a2a', 
        padding: '15px', 
        borderRadius: '8px',
        margin: '20px 0'
      }}>
        <h2>Next Steps</h2>
        <p>Now we can gradually add back the complex components to identify what's causing the white screen.</p>
      </div>

      <button 
        onClick={() => alert('Button works!')}
        style={{
          backgroundColor: '#00ff88',
          color: '#1a1a1a',
          border: 'none',
          padding: '10px 20px',
          borderRadius: '5px',
          cursor: 'pointer',
          fontSize: '16px'
        }}
      >
        Test Interaction
      </button>
    </div>
  );
}

export default App;
