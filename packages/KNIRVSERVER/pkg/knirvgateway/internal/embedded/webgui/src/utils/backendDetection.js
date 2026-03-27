import api from './api';

   export const checkBackendConnection = async () => {
     try {
       // Check if backend URL is configured
       if (!process.env.NEXT_PUBLIC_BACKEND_URL) {
         console.warn('Backend URL not configured, running in standalone mode');
         return false;
       }

       const response = await api.get('/health');
       return response.data.status === 'ok' || response.data.status === 'healthy';
     } catch (error) {
       console.warn('Backend connection error:', error.message);
       return false;
     }
   };

   export const getServerInfo = async () => {
     try {
       // Check if backend URL is configured
       if (!process.env.NEXT_PUBLIC_BACKEND_URL) {
         console.warn('Backend URL not configured, returning mock server info');
         return {
           role: 'General',
           network: 'demo',
           mode: 'standalone'
         };
       }

       const response = await api.get('/info');
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