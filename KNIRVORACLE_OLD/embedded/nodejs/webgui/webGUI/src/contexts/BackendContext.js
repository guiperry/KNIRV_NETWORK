import React, { createContext, useState, useEffect, useContext } from 'react';
   import axios from 'axios';
   import { checkBackendConnection, getServerInfo } from '../utils/backendDetection';

   const BackendContext = createContext();

   export const BackendProvider = ({ children }) => {
     const [isRunning, setIsRunning] = useState(false);
     const [serverInfo, setServerInfo] = useState(null);
     const [isLoading, setIsLoading] = useState(true);

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
         }
         
         setIsLoading(false);
       } catch (error) {
         console.error('Error checking backend status:', error);
         setIsRunning(false);
         setIsLoading(false);
       }
     };

     return (
       <BackendContext.Provider
         value={{
           isRunning,
           serverInfo,
           isLoading,
           refreshStatus: checkBackendStatus,
         }}
       >
         {children}
       </BackendContext.Provider>
     );
   };

   export const useBackend = () => useContext(BackendContext);