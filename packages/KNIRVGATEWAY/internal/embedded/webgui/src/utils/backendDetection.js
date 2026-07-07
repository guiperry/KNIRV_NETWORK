import api from './api';

   export const checkBackendConnection = async () => {
     try {
       // Use the gateway's own API — this runs in the browser so relative
       // URLs resolve to the gateway's host where /api/v1/* is proxied.
       const response = await api.get('/api/v1/health');
       return response.data.status === 'ok' || response.data.status === 'healthy';
     } catch (error) {
       console.warn('Backend connection error:', error.message);
       return false;
     }
   };

   export const getServerInfo = async () => {
     try {
       // Use relative URL — resolves to the gateway's own /api/v1/ path
       const response = await api.get('/api/v1/info');
       return response.data;
     } catch (error) {
       console.warn('Failed to get server info:', error.message);
       return {
         role: 'General',
         network: 'demo',
         mode: 'standalone'
       };
     }
   };