import React, { createContext, useState, useEffect, useContext } from 'react';
   import axios from 'axios';
   import { checkBackendConnection, getServerInfo } from '../utils/backendDetection';

   const BackendContext = createContext();

   export const BackendProvider = ({ children }) => {
     const [isRunning, setIsRunning] = useState(false);
     const [serverInfo, setServerInfo] = useState(null);
     const [isLoading, setIsLoading] = useState(true);
     const [oracleStatus, setOracleStatus] = useState(null);
     const [services, setServices] = useState([]);

     useEffect(() => {
       checkBackendStatus();
       
       // Poll for backend status every 10 seconds
       const interval = setInterval(() => {
         checkBackendStatus();
       }, 10000);
       
       return () => {
         clearInterval(interval);
       };
     }, []);

     const checkBackendStatus = async () => {
       setIsLoading(true);
       try {
         const isConnected = await checkBackendConnection();
         setIsRunning(isConnected);

         if (isConnected) {
           const info = await getServerInfo();
           setServerInfo(info);

           // Fetch Oracle status and services from unified API
           await fetchOracleStatus();
           await fetchServices();
         }

         setIsLoading(false);
       } catch (error) {
         console.error('Error checking backend status:', error);
         setIsRunning(false);
         setIsLoading(false);
       }
     };

     const fetchOracleStatus = async () => {
       try {
         const response = await axios.get('/api/oracle/status');
         setOracleStatus(response.data);
       } catch (error) {
         console.error('Error fetching Oracle status:', error);
       }
     };

     const fetchServices = async () => {
       try {
         const response = await axios.get('/api/services');
         setServices(response.data.services || []);
       } catch (error) {
         console.error('Error fetching services:', error);
       }
     };

     const startService = async (serviceName) => {
       try {
         const response = await axios.post(`/api/services/${serviceName}/start`);
         await fetchServices(); // Refresh services list
         return response.data;
       } catch (error) {
         console.error(`Error starting service ${serviceName}:`, error);
         throw error;
       }
     };

     const stopService = async (serviceName) => {
       try {
         const response = await axios.post(`/api/services/${serviceName}/stop`);
         await fetchServices(); // Refresh services list
         return response.data;
       } catch (error) {
         console.error(`Error stopping service ${serviceName}:`, error);
         throw error;
       }
     };

     const restartService = async (serviceName) => {
       try {
         const response = await axios.post(`/api/services/${serviceName}/restart`);
         await fetchServices(); // Refresh services list
         return response.data;
       } catch (error) {
         console.error(`Error restarting service ${serviceName}:`, error);
         throw error;
       }
     };

     return (
       <BackendContext.Provider
         value={{
           isRunning,
           serverInfo,
           isLoading,
           oracleStatus,
           services,
           refreshStatus: checkBackendStatus,
           fetchOracleStatus,
           fetchServices,
           startService,
           stopService,
           restartService,
         }}
       >
         {children}
       </BackendContext.Provider>
     );
   };

   export const useBackend = () => useContext(BackendContext);