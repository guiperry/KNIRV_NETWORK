import React from 'react';

export default function Basic() {
  return (
    <div style={{ 
      backgroundColor: '#000c2e', 
      color: 'white', 
      minHeight: '100vh', 
      padding: '20px' 
    }}>
      <h1 style={{ color: '#007bff' }}>Blockchain Dashboard</h1>
      <p>Welcome to the blockchain dashboard.</p>
      
      <div style={{ 
        display: 'flex', 
        flexWrap: 'wrap', 
        gap: '20px', 
        marginTop: '20px' 
      }}>
        <div style={{ 
          backgroundColor: '#343a40', 
          padding: '20px', 
          borderRadius: '10px', 
          flex: '1 1 45%' 
        }}>
          <h2 style={{ color: '#007bff' }}>Dashboard Card</h2>
          <p>This is a simple card component.</p>
        </div>
        
        <div style={{ 
          backgroundColor: '#343a40', 
          padding: '20px', 
          borderRadius: '10px', 
          flex: '1 1 45%' 
        }}>
          <h2 style={{ color: '#007bff' }}>Analytics</h2>
          <p>This card contains analytics information.</p>
        </div>
      </div>
    </div>
  );
}