import React from 'react';
import { useRouter } from 'next/router';
import Viewport from '../components/Viewport';

const ViewportPage: React.FC = () => {
  const router = useRouter();
  const { id } = router.query;
  
  return (
    <div style={{ 
      width: '100%', 
      height: '100vh',
      background: '#0a0e17',
      display: 'flex',
      flexDirection: 'column'
    }}>
      <header style={{
        padding: '15px 20px',
        background: 'rgba(16, 24, 48, 0.7)',
        borderBottom: '1px solid rgba(100, 130, 255, 0.2)',
        color: '#fff'
      }}>
        <h1>KNIRVCHAIN 3D Viewer</h1>
      </header>
      
      <div style={{ flex: 1 }}>
        <Viewport modelId={id as string} />
      </div>
    </div>
  );
};

export default ViewportPage;