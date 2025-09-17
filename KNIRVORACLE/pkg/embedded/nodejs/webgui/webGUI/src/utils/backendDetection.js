import api from './api';

   export const checkBackendConnection = async () => {
     try {
       const response = await api.get('/health');
       return response.data.status === 'ok';
     } catch (error) {
       console.error('Backend connection error:', error);
       return false;
     }
   };

   export const getServerInfo = async () => {
     try {
       const response = await api.get('/info');
       return response.data;
     } catch (error) {
       console.error('Failed to get server info:', error);
       return null;
     }
   };